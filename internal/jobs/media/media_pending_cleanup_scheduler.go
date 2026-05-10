package jobmedia

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"mycourse-io-be/constants"
	"mycourse-io-be/repository"
)

var (
	mediaCleanupMu     sync.Mutex
	mediaCleanupCancel context.CancelFunc
	mediaCleanupWG     sync.WaitGroup
)

func mediaPendingCleanupIntervalFromEnv() time.Duration {
	s := strings.TrimSpace(os.Getenv("MEDIA_CLEANUP_INTERVAL_SEC"))
	if s == "0" {
		return 0
	}
	if s == "" {
		return time.Duration(constants.MediaCleanupDefaultIntervalSec) * time.Second
	}
	sec, err := strconv.Atoi(s)
	if err != nil || sec <= 0 {
		return time.Duration(constants.MediaCleanupDefaultIntervalSec) * time.Second
	}
	return time.Duration(sec) * time.Second
}

// StopMediaPendingCleanupJob stops the background pending-cleanup worker.
func StopMediaPendingCleanupJob() {
	mediaCleanupMu.Lock()
	c := mediaCleanupCancel
	mediaCleanupCancel = nil
	mediaCleanupMu.Unlock()
	if c == nil {
		return
	}
	c()
	mediaCleanupWG.Wait()
}

// StartMediaPendingCleanupJob starts (or replaces) the media pending cloud cleanup loop.
func StartMediaPendingCleanupJob(db *gorm.DB) {
	if db == nil {
		log.Println("media-pending-cleanup-job: skipped (nil database)")
		return
	}
	interval := mediaPendingCleanupIntervalFromEnv()
	if interval <= 0 {
		log.Println("media-pending-cleanup-job: skipped (MEDIA_CLEANUP_INTERVAL_SEC=0)")
		return
	}
	StopMediaPendingCleanupJob()

	ctx, cancel := context.WithCancel(context.Background())
	mediaCleanupMu.Lock()
	mediaCleanupCancel = cancel
	mediaCleanupMu.Unlock()

	mediaCleanupWG.Add(1)
	go func() {
		defer mediaCleanupWG.Done()
		runMediaPendingCleanupLoop(ctx, db, interval)
	}()

	log.Printf("media-pending-cleanup-job: started (interval=%s)", interval)
}

func runMediaPendingCleanupLoop(ctx context.Context, db *gorm.DB, interval time.Duration) {
	runOnce := func() {
		repo := repository.New(db).Media
		ProcessPendingCleanupBatch(context.Background(), repo)
	}

	runOnce()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("media-pending-cleanup-job: stopped")
			return
		case <-ticker.C:
			runOnce()
		}
	}
}
