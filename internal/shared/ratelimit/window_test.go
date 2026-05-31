package ratelimit

import (
	"sync"
	"testing"
)

func TestAllowFixedWindowWithinQuota(t *testing.T) {
	buckets := make(map[string]*bucket)
	var mu sync.Mutex
	key := "test"
	windowSec := int64(60)
	windowStart := int64(0)
	attempts := 3

	for i := 0; i < attempts; i++ {
		if !AllowFixedWindow(buckets, &mu, key, windowSec, windowStart, attempts, false) {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
	}
	if AllowFixedWindow(buckets, &mu, key, windowSec, windowStart, attempts, false) {
		t.Fatal("fourth attempt should be denied")
	}
}

func TestAllowFixedWindowRollover(t *testing.T) {
	buckets := make(map[string]*bucket)
	var mu sync.Mutex
	key := "test"
	windowSec := int64(60)
	firstWindow := int64(0)
	nextWindow := int64(60)
	attempts := 2

	for i := 0; i < attempts; i++ {
		if !AllowFixedWindow(buckets, &mu, key, windowSec, firstWindow, attempts, false) {
			t.Fatalf("first window attempt %d denied", i+1)
		}
	}
	if AllowFixedWindow(buckets, &mu, key, windowSec, firstWindow, attempts, false) {
		t.Fatal("third attempt in first window should be denied")
	}
	if !AllowFixedWindow(buckets, &mu, key, windowSec, nextWindow, attempts, false) {
		t.Fatal("first attempt in next window should be allowed")
	}
}

func TestInMemoryStoreIndependentKeys(t *testing.T) {
	store := NewInMemoryStore(false)
	windowSec := int64(60)
	windowStart := int64(0)
	if !store.Allow("a", windowSec, windowStart, 1) {
		t.Fatal("first key should be allowed")
	}
	if store.Allow("a", windowSec, windowStart, 1) {
		t.Fatal("second attempt on same key should be denied")
	}
	if !store.Allow("b", windowSec, windowStart, 1) {
		t.Fatal("different key should be allowed")
	}
}

func TestFileStorePersistsAcrossInstances(t *testing.T) {
	t.Setenv("MYCOURSE_CLI_RATE_LIMIT_PATH", t.TempDir()+"/cli_rate_limit.json")
	windowSec := int64(180)
	windowStart := int64(0)
	key := "CLI_SYSTEM_LOGIN|abc"
	attempts := 2

	store1, err := DefaultCLIFileStore()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < attempts; i++ {
		ok, err := store1.Allow(key, windowSec, windowStart, attempts)
		if err != nil || !ok {
			t.Fatalf("attempt %d: ok=%v err=%v", i+1, ok, err)
		}
	}
	store2, err := DefaultCLIFileStore()
	if err != nil {
		t.Fatal(err)
	}
	ok, err := store2.Allow(key, windowSec, windowStart, attempts)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("new process instance should see persisted count and deny")
	}
}
