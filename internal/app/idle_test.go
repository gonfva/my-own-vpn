package app

import (
	"sync"
	"testing"
	"time"
)

func TestNewIdleMonitor(t *testing.T) {
	monitor := NewIdleMonitor(30*time.Minute, 5*time.Minute)
	if monitor == nil {
		t.Fatal("NewIdleMonitor returned nil")
	}
	if monitor.IsEnabled() {
		t.Error("new monitor should not be enabled")
	}
	if monitor.Timeout() != 30*time.Minute {
		t.Errorf("expected timeout 30m, got %v", monitor.Timeout())
	}
}

func TestIdleMonitorStartStop(t *testing.T) {
	monitor := NewIdleMonitor(100*time.Millisecond, 50*time.Millisecond)

	if monitor.IsEnabled() {
		t.Error("monitor should not be enabled before Start")
	}

	monitor.Start()
	if !monitor.IsEnabled() {
		t.Error("monitor should be enabled after Start")
	}

	monitor.Stop()
	// Give the goroutine time to stop
	time.Sleep(10 * time.Millisecond)
	if monitor.IsEnabled() {
		t.Error("monitor should not be enabled after Stop")
	}
}

func TestIdleMonitorDoubleStart(t *testing.T) {
	monitor := NewIdleMonitor(100*time.Millisecond, 50*time.Millisecond)

	// Starting twice should not create multiple goroutines
	monitor.Start()
	monitor.Start()

	if !monitor.IsEnabled() {
		t.Error("monitor should be enabled")
	}

	monitor.Stop()
}

func TestIdleMonitorWarningCallback(t *testing.T) {
	var mu sync.Mutex
	warningCalled := false
	warningMinutes := 0

	monitor := NewIdleMonitor(150*time.Millisecond, 100*time.Millisecond)
	monitor.SetCallbacks(
		func(remainingMinutes int) {
			mu.Lock()
			warningCalled = true
			warningMinutes = remainingMinutes
			mu.Unlock()
		},
		nil,
	)
	monitor.Start()
	defer monitor.Stop()

	// Warning should trigger after 50ms (150ms - 100ms)
	// But monitor checks every 30 seconds, so we need to manually trigger
	// Let's use a shorter check by directly calling checkIdle

	// Wait past the warning threshold
	time.Sleep(60 * time.Millisecond)
	monitor.checkIdle()

	mu.Lock()
	if !warningCalled {
		mu.Unlock()
		t.Error("warning callback should have been called")
		return
	}
	if warningMinutes < 1 {
		mu.Unlock()
		t.Error("warning minutes should be at least 1")
		return
	}
	mu.Unlock()
}

func TestIdleMonitorTimeoutCallback(t *testing.T) {
	var mu sync.Mutex
	timeoutCalled := false

	monitor := NewIdleMonitor(100*time.Millisecond, 50*time.Millisecond)
	monitor.SetCallbacks(
		nil,
		func() {
			mu.Lock()
			timeoutCalled = true
			mu.Unlock()
		},
	)
	monitor.Start()

	// Wait past the timeout threshold
	time.Sleep(110 * time.Millisecond)
	monitor.checkIdle()

	mu.Lock()
	if !timeoutCalled {
		mu.Unlock()
		t.Error("timeout callback should have been called")
		return
	}
	mu.Unlock()

	// Monitor should be disabled after timeout
	if monitor.IsEnabled() {
		t.Error("monitor should be disabled after timeout")
	}
}

func TestIdleMonitorUpdateActivity(t *testing.T) {
	var mu sync.Mutex
	timeoutCalled := false

	monitor := NewIdleMonitor(100*time.Millisecond, 50*time.Millisecond)
	monitor.SetCallbacks(
		nil,
		func() {
			mu.Lock()
			timeoutCalled = true
			mu.Unlock()
		},
	)
	monitor.Start()
	defer monitor.Stop()

	// Simulate activity every 40ms by updating traffic stats
	for i := 0; i < 5; i++ {
		time.Sleep(40 * time.Millisecond)
		monitor.UpdateActivity(uint64(i*100), uint64(i*50))
		monitor.checkIdle()
	}

	mu.Lock()
	if timeoutCalled {
		mu.Unlock()
		t.Error("timeout should not be called when there's activity")
		return
	}
	mu.Unlock()
}

func TestIdleMonitorResetActivity(t *testing.T) {
	monitor := NewIdleMonitor(100*time.Millisecond, 50*time.Millisecond)
	monitor.Start()
	defer monitor.Stop()

	// Wait a bit
	time.Sleep(30 * time.Millisecond)

	idleTimeBefore := monitor.IdleTime()

	// Reset activity
	monitor.ResetActivity()

	// Idle time should be reset
	idleTimeAfter := monitor.IdleTime()
	if idleTimeAfter >= idleTimeBefore {
		t.Error("idle time should be reset after ResetActivity")
	}
}

func TestIdleMonitorIdleTime(t *testing.T) {
	monitor := NewIdleMonitor(1*time.Minute, 30*time.Second)

	// Before start, idle time should be 0
	if monitor.IdleTime() != 0 {
		t.Error("idle time should be 0 before start")
	}

	monitor.Start()
	defer monitor.Stop()

	time.Sleep(50 * time.Millisecond)

	idleTime := monitor.IdleTime()
	if idleTime < 50*time.Millisecond {
		t.Errorf("expected idle time >= 50ms, got %v", idleTime)
	}
}

func TestIdleMonitorSameTrafficNoReset(t *testing.T) {
	monitor := NewIdleMonitor(100*time.Millisecond, 50*time.Millisecond)
	monitor.Start()
	defer monitor.Stop()

	// Set initial traffic
	monitor.UpdateActivity(100, 50)

	time.Sleep(30 * time.Millisecond)
	idleTime1 := monitor.IdleTime()

	// Update with same values - should NOT reset timer
	time.Sleep(20 * time.Millisecond)
	monitor.UpdateActivity(100, 50)

	idleTime2 := monitor.IdleTime()

	// Idle time should have continued (not reset)
	if idleTime2 < idleTime1 {
		t.Error("idle time should not reset when traffic hasn't changed")
	}
}

func TestIdleMonitorConcurrency(t *testing.T) {
	monitor := NewIdleMonitor(1*time.Second, 500*time.Millisecond)
	monitor.Start()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				monitor.UpdateActivity(uint64(id*100+j), uint64(j))
				monitor.IdleTime()
				monitor.IsEnabled()
			}
		}(i)
	}

	wg.Wait()
	monitor.Stop()
}

func TestIdleMonitorWarningOnlyOnce(t *testing.T) {
	var mu sync.Mutex
	warningCount := 0

	monitor := NewIdleMonitor(200*time.Millisecond, 100*time.Millisecond)
	monitor.SetCallbacks(
		func(remainingMinutes int) {
			mu.Lock()
			warningCount++
			mu.Unlock()
		},
		nil,
	)
	monitor.Start()
	defer monitor.Stop()

	// Wait past warning threshold
	time.Sleep(110 * time.Millisecond)

	// Trigger check multiple times
	for i := 0; i < 5; i++ {
		monitor.checkIdle()
	}

	mu.Lock()
	if warningCount != 1 {
		mu.Unlock()
		t.Errorf("warning should be called exactly once, got %d", warningCount)
		return
	}
	mu.Unlock()
}

func TestIdleMonitorWarningResetAfterActivity(t *testing.T) {
	var mu sync.Mutex
	warningCount := 0

	monitor := NewIdleMonitor(200*time.Millisecond, 100*time.Millisecond)
	monitor.SetCallbacks(
		func(remainingMinutes int) {
			mu.Lock()
			warningCount++
			mu.Unlock()
		},
		nil,
	)
	monitor.Start()
	defer monitor.Stop()

	// Wait past warning threshold and trigger warning
	time.Sleep(110 * time.Millisecond)
	monitor.checkIdle()

	mu.Lock()
	if warningCount != 1 {
		mu.Unlock()
		t.Fatalf("expected 1 warning, got %d", warningCount)
	}
	mu.Unlock()

	// Simulate activity (reset the warning flag)
	monitor.UpdateActivity(1000, 500)

	// Wait past warning threshold again
	time.Sleep(110 * time.Millisecond)
	monitor.checkIdle()

	mu.Lock()
	if warningCount != 2 {
		mu.Unlock()
		t.Errorf("warning should be called again after activity, got %d", warningCount)
		return
	}
	mu.Unlock()
}
