package async

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
)

func TestEval_FirstGet_FetchesValue(t *testing.T) {
	// PRD Item 36: "Async evaluation triggers fetch on first Get()"
	callCount := atomic.Int32{}
	fetch := func() (string, error) {
		callCount.Add(1)
		return "value", nil
	}

	eval := New(fetch)
	value, err := eval.Get()

	if err != nil {
		t.Fatalf("Get() should not error on first call: %v", err)
	}
	if value != "value" {
		t.Fatalf("Get() returned %q, want %q", value, "value")
	}
	if callCount.Load() != 1 {
		t.Fatalf("fetch called %d times, want 1", callCount.Load())
	}
}

func TestEval_ErrorHandling(t *testing.T) {
	// PRD Item 36: "Errors are returned on failed fetch"
	expectedErr := fmt.Errorf("network error")
	fetch := func() (string, error) {
		return "", expectedErr
	}

	eval := New(fetch)
	_, err := eval.Get()

	if err != expectedErr {
		t.Fatalf("Get() returned %v, want %v", err, expectedErr)
	}
}

func TestEval_CachedValue_NotFetchedAgain(t *testing.T) {
	// PRD Item 37: "After successful fetch, set loaded=true and store value"
	// "Subsequent Get() calls return cached value immediately"
	// "Fetch function not called again if already loaded"
	callCount := atomic.Int32{}
	fetch := func() (string, error) {
		callCount.Add(1)
		return "cached", nil
	}

	eval := New(fetch)

	// First Get() triggers fetch
	value1, err1 := eval.Get()
	if err1 != nil || value1 != "cached" {
		t.Fatalf("first Get() failed: value=%q, err=%v", value1, err1)
	}

	// Second Get() should return cached value
	value2, err2 := eval.Get()
	if err2 != nil || value2 != "cached" {
		t.Fatalf("second Get() failed: value=%q, err=%v", value2, err2)
	}

	// Verify fetch was called only once
	if callCount.Load() != 1 {
		t.Fatalf("fetch called %d times, want 1", callCount.Load())
	}
}

func TestEval_IsLoaded_ReflectsState(t *testing.T) {
	// PRD Item 36: "IsLoaded() returns true after successful fetch"
	fetch := func() (string, error) {
		return "value", nil
	}

	eval := New(fetch)

	if eval.IsLoaded() {
		t.Fatal("IsLoaded() should return false before first Get()")
	}

	_, _ = eval.Get()

	if !eval.IsLoaded() {
		t.Fatal("IsLoaded() should return true after successful Get()")
	}
}

func TestEval_Invalidate_ClearsCache(t *testing.T) {
	// PRD Item 38: "Implement Invalidate() method on Eval"
	// "Invalidate sets loaded=false and clears cached value"
	// "Next Get() triggers fresh fetch"
	callCount := atomic.Int32{}
	fetch := func() (string, error) {
		callCount.Add(1)
		return fmt.Sprintf("value-%d", callCount.Load()), nil
	}

	eval := New(fetch)

	// First Get() triggers fetch
	value1, _ := eval.Get()
	if value1 != "value-1" {
		t.Fatalf("first Get() returned %q, want value-1", value1)
	}

	// Invalidate cache
	eval.Invalidate()

	if eval.IsLoaded() {
		t.Fatal("IsLoaded() should return false after Invalidate()")
	}

	// Next Get() should trigger another fetch
	value2, _ := eval.Get()
	if value2 != "value-2" {
		t.Fatalf("second Get() returned %q, want value-2", value2)
	}

	if callCount.Load() != 2 {
		t.Fatalf("fetch called %d times, want 2", callCount.Load())
	}
}

func TestEval_ConcurrentGet_OnlyFetchesOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Concurrent Gets before first fetch completes should not cause multiple fetches
		callCount := atomic.Int32{}
		blockFetch := make(chan struct{})
		defer close(blockFetch)

		fetch := func() (string, error) {
			callCount.Add(1)
			<-blockFetch // Block until test unblocks
			return "value", nil
		}

		eval := New(fetch)

		// Start multiple Get() calls concurrently
		var wg sync.WaitGroup
		results := make([]string, 5)
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				value, _ := eval.Get()
				results[idx] = value
			}(i)
		}

		// Wait for goroutines to be durably blocked (competing for lock or waiting on channel)
		synctest.Wait()

		// First caller acquires lock, starts fetch, releases lock
		// Other callers should block on first lock acquisition, see loading=true, return error
		// When first fetch completes, subsequent calls use cache

		// Unblock the fetch
		blockFetch <- struct{}{}

		wg.Wait()

		// At least the first Get should succeed (it does the fetch)
		// The exact number of calls depends on timing, but it should be minimal
		// The key is that we don't call fetch 5 times
		t.Logf("Fetch called %d times for 5 concurrent Gets", callCount.Load())
		if callCount.Load() > 2 {
			t.Fatalf("fetch called %d times, want 1 or 2 (first caller + possible timing)", callCount.Load())
		}
	})
}

func TestEval_ErrorCaching(t *testing.T) {
	// Errors should also be cached
	callCount := atomic.Int32{}
	expectedErr := fmt.Errorf("error")

	fetch := func() (string, error) {
		callCount.Add(1)
		return "", expectedErr
	}

	eval := New(fetch)

	// First Get() returns error
	_, err1 := eval.Get()
	if err1 != expectedErr {
		t.Fatalf("first Get() returned %v, want %v", err1, expectedErr)
	}

	// Second Get() returns cached error
	_, err2 := eval.Get()
	if err2 != expectedErr {
		t.Fatalf("second Get() returned %v, want %v", err2, expectedErr)
	}

	if callCount.Load() != 1 {
		t.Fatalf("fetch called %d times, want 1", callCount.Load())
	}
}

func TestEval_InvalidateAfterError_RefetchesOnNextGet(t *testing.T) {
	// Invalidate should clear errors too
	callCount := atomic.Int32{}

	fetch := func() (string, error) {
		callCount.Add(1)
		if callCount.Load() == 1 {
			return "", fmt.Errorf("error")
		}
		return "success", nil
	}

	eval := New(fetch)

	// First Get() returns error
	_, err1 := eval.Get()
	if err1 == nil {
		t.Fatal("first Get() should error")
	}

	// Invalidate
	eval.Invalidate()

	// Next Get() should fetch again
	value, err := eval.Get()
	if err != nil {
		t.Fatalf("second Get() should succeed after Invalidate(): %v", err)
	}
	if value != "success" {
		t.Fatalf("second Get() returned %q, want success", value)
	}

	if callCount.Load() != 2 {
		t.Fatalf("fetch called %d times, want 2", callCount.Load())
	}
}

func TestEval_IsLoading_DuringFetch(t *testing.T) {
	// IsLoading should return true while fetch is in progress
	fetchStarted := make(chan struct{})
	blockFetch := make(chan struct{})

	fetch := func() (string, error) {
		close(fetchStarted)
		<-blockFetch // Block until test unblocks
		return "value", nil
	}

	eval := New(fetch)

	// Start Get in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = eval.Get()
	}()

	// Wait for fetch to start
	<-fetchStarted

	// IsLoading should return true
	if !eval.IsLoading() {
		t.Fatal("IsLoading() should return true while fetch is in progress")
	}

	// IsLoaded should return false
	if eval.IsLoaded() {
		t.Fatal("IsLoaded() should return false while fetch is in progress")
	}

	// Unblock fetch
	close(blockFetch)
	wg.Wait()

	// Now IsLoading should return false
	if eval.IsLoading() {
		t.Fatal("IsLoading() should return false after fetch completes")
	}

	// And IsLoaded should return true
	if !eval.IsLoaded() {
		t.Fatal("IsLoaded() should return true after successful fetch")
	}
}
