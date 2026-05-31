package resilience

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"mycourse-io-be/internal/shared/cache"
	"mycourse-io-be/internal/shared/setting"
)

const redisKey = "mycourse:resilience:circuit"

type state int

const (
	stateClosed state = iota
	stateOpen
	stateHalfOpen
)

// CircuitBreaker tracks DB health and HTTP load to fast-fail when the service is degraded.
type CircuitBreaker struct {
	mu sync.RWMutex

	state             state
	consecutiveDBFail int
	openUntil         time.Time
	halfOpenRemaining int

	inFlight    int64
	errorWindow []time.Time

	cfg Config
}

// Config holds circuit breaker thresholds (YAML resilience: block).
type Config struct {
	DBProbeIntervalSec   int
	DBFailuresToOpen     int
	MaxInFlight          int
	HalfOpenProbeQuota   int
	OpenCooldownSec      int
	ErrorWindowSec       int
	ErrorCountToOpen     int
	DegradedAttemptsFactor float64
}

// DefaultConfig returns plan defaults when YAML omits resilience settings.
func DefaultConfig() Config {
	return Config{
		DBProbeIntervalSec:     5,
		DBFailuresToOpen:       3,
		MaxInFlight:            200,
		HalfOpenProbeQuota:     3,
		OpenCooldownSec:        30,
		ErrorWindowSec:         60,
		ErrorCountToOpen:       50,
		DegradedAttemptsFactor: 0.5,
	}
}

// Global is the process-wide circuit breaker used by HTTP middleware and APPCLI.
var Global = NewCircuitBreaker(DefaultConfig())

func NewCircuitBreaker(cfg Config) *CircuitBreaker {
	normalizeConfig(&cfg)
	return &CircuitBreaker{cfg: cfg, halfOpenRemaining: cfg.HalfOpenProbeQuota}
}

func normalizeConfig(cfg *Config) {
	d := DefaultConfig()
	if cfg.DBProbeIntervalSec < 1 {
		cfg.DBProbeIntervalSec = d.DBProbeIntervalSec
	}
	if cfg.DBFailuresToOpen < 1 {
		cfg.DBFailuresToOpen = d.DBFailuresToOpen
	}
	if cfg.MaxInFlight < 1 {
		cfg.MaxInFlight = d.MaxInFlight
	}
	if cfg.HalfOpenProbeQuota < 1 {
		cfg.HalfOpenProbeQuota = d.HalfOpenProbeQuota
	}
	if cfg.OpenCooldownSec < 1 {
		cfg.OpenCooldownSec = d.OpenCooldownSec
	}
	if cfg.ErrorWindowSec < 1 {
		cfg.ErrorWindowSec = d.ErrorWindowSec
	}
	if cfg.ErrorCountToOpen < 1 {
		cfg.ErrorCountToOpen = d.ErrorCountToOpen
	}
	if cfg.DegradedAttemptsFactor <= 0 || cfg.DegradedAttemptsFactor > 1 {
		cfg.DegradedAttemptsFactor = d.DegradedAttemptsFactor
	}
}

// ConfigureFromSettings rebuilds Global from setting.ResilienceSetting.
func ConfigureFromSettings() {
	cfg := DefaultConfig()
	if s := setting.ResilienceSetting; s != nil {
		if s.DBProbeIntervalSec > 0 {
			cfg.DBProbeIntervalSec = s.DBProbeIntervalSec
		}
		if s.DBFailuresToOpen > 0 {
			cfg.DBFailuresToOpen = s.DBFailuresToOpen
		}
		if s.MaxInFlight > 0 {
			cfg.MaxInFlight = s.MaxInFlight
		}
		if s.HalfOpenProbeQuota > 0 {
			cfg.HalfOpenProbeQuota = s.HalfOpenProbeQuota
		}
		if s.OpenCooldownSec > 0 {
			cfg.OpenCooldownSec = s.OpenCooldownSec
		}
		if s.ErrorWindowSec > 0 {
			cfg.ErrorWindowSec = s.ErrorWindowSec
		}
		if s.ErrorCountToOpen > 0 {
			cfg.ErrorCountToOpen = s.ErrorCountToOpen
		}
		if s.DegradedAttemptsFactor > 0 && s.DegradedAttemptsFactor <= 1 {
			cfg.DegradedAttemptsFactor = s.DegradedAttemptsFactor
		}
	}
	Global = NewCircuitBreaker(cfg)
}

// ForceOpenForTest opens the circuit immediately (testing only).
func ForceOpenForTest(cb *CircuitBreaker) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.openLocked("test")
}

// Allow reports whether a new request/CLI operation may proceed.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.syncFromRedisLocked()

	switch cb.state {
	case stateClosed:
		return true
	case stateOpen:
		if time.Now().Before(cb.openUntil) {
			return false
		}
		cb.transitionHalfOpenLocked()
		return cb.consumeHalfOpenLocked()
	case stateHalfOpen:
		return cb.consumeHalfOpenLocked()
	default:
		return true
	}
}

func (cb *CircuitBreaker) consumeHalfOpenLocked() bool {
	if cb.halfOpenRemaining <= 0 {
		return false
	}
	cb.halfOpenRemaining--
	return true
}

func (cb *CircuitBreaker) transitionHalfOpenLocked() {
	cb.state = stateHalfOpen
	cb.halfOpenRemaining = cb.cfg.HalfOpenProbeQuota
}

// AttemptsFactor returns 1.0 when healthy, or a reduced factor when half-open (stricter rate limits).
func (cb *CircuitBreaker) AttemptsFactor() float64 {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	if cb.state == stateHalfOpen {
		return cb.cfg.DegradedAttemptsFactor
	}
	return 1.0
}

// RecordRequestStart increments in-flight counter; call the returned func when the request ends.
func (cb *CircuitBreaker) RecordRequestStart() func(success bool) {
	atomic.AddInt64(&cb.inFlight, 1)
	cb.checkLoadOpen()
	return func(success bool) {
		atomic.AddInt64(&cb.inFlight, -1)
		if !success {
			cb.recordHTTPErr()
		} else {
			cb.recordHTTPSuccess()
		}
	}
}

func (cb *CircuitBreaker) checkLoadOpen() {
	inFlight := atomic.LoadInt64(&cb.inFlight)
	if int(inFlight) <= cb.cfg.MaxInFlight {
		return
	}
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.state == stateClosed {
		cb.openLocked("max in-flight exceeded")
	}
}

func (cb *CircuitBreaker) recordHTTPErr() {
	now := time.Now()
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.errorWindow = append(cb.errorWindow, now)
	cutoff := now.Add(-time.Duration(cb.cfg.ErrorWindowSec) * time.Second)
	kept := cb.errorWindow[:0]
	for _, t := range cb.errorWindow {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	cb.errorWindow = kept
	if cb.state == stateClosed && len(cb.errorWindow) >= cb.cfg.ErrorCountToOpen {
		cb.openLocked("HTTP error rate exceeded")
		return
	}
	if cb.state == stateHalfOpen {
		cb.openLocked("half-open probe failed")
	}
}

func (cb *CircuitBreaker) recordHTTPSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.state == stateHalfOpen {
		cb.closeLocked()
	}
}

func (cb *CircuitBreaker) recordDBSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.consecutiveDBFail = 0
	if cb.state == stateHalfOpen {
		cb.closeLocked()
	}
	cb.persistLocked()
}

func (cb *CircuitBreaker) recordDBFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.consecutiveDBFail++
	if cb.state == stateClosed && cb.consecutiveDBFail >= cb.cfg.DBFailuresToOpen {
		cb.openLocked("database probe failed")
		return
	}
	if cb.state == stateHalfOpen {
		cb.openLocked("half-open DB probe failed")
	}
	cb.persistLocked()
}

func (cb *CircuitBreaker) openLocked(reason string) {
	cb.state = stateOpen
	cb.openUntil = time.Now().Add(time.Duration(cb.cfg.OpenCooldownSec) * time.Second)
	cb.halfOpenRemaining = 0
	zap.L().Warn("circuit breaker opened", zap.String("reason", reason))
	cb.persistLocked()
}

func (cb *CircuitBreaker) closeLocked() {
	cb.state = stateClosed
	cb.consecutiveDBFail = 0
	cb.halfOpenRemaining = cb.cfg.HalfOpenProbeQuota
	cb.errorWindow = nil
	cb.persistLocked()
}

type redisSnapshot struct {
	State             int       `json:"state"`
	OpenUntil         time.Time `json:"openUntil"`
	ConsecutiveDBFail int       `json:"consecutiveDBFail"`
}

func (cb *CircuitBreaker) persistLocked() {
	if !cache.RedisAvailable() {
		return
	}
	snap := redisSnapshot{
		State:             int(cb.state),
		OpenUntil:         cb.openUntil,
		ConsecutiveDBFail: cb.consecutiveDBFail,
	}
	data, err := json.Marshal(snap)
	if err != nil {
		return
	}
	_ = cache.Redis.Set(context.Background(), redisKey, data, time.Duration(cb.cfg.OpenCooldownSec*2)*time.Second).Err()
}

func (cb *CircuitBreaker) syncFromRedisLocked() {
	if !cache.RedisAvailable() {
		return
	}
	data, err := cache.Redis.Get(context.Background(), redisKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return
		}
		return
	}
	var snap redisSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return
	}
	if snap.State == int(stateOpen) && time.Now().Before(snap.OpenUntil) {
		cb.state = stateOpen
		cb.openUntil = snap.OpenUntil
		cb.consecutiveDBFail = snap.ConsecutiveDBFail
	}
}

// StartDBProbe runs periodic PingContext on db until ctx is cancelled.
func StartDBProbe(ctx context.Context, db *sql.DB) {
	if db == nil {
		return
	}
	interval := time.Duration(Global.cfg.DBProbeIntervalSec) * time.Second
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
				err := db.PingContext(pingCtx)
				cancel()
				if err != nil {
					Global.recordDBFailure()
				} else {
					Global.recordDBSuccess()
				}
			}
		}
	}()
}
