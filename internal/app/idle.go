package app

import (
	"sync"
	"time"
)

// IdleMonitor tracks network activity and triggers callbacks when the VPN
// has been idle for a configurable period.
type IdleMonitor struct {
	mu            sync.Mutex
	enabled       bool
	timeout       time.Duration
	warningPeriod time.Duration
	lastActivity  time.Time
	lastBytesSent uint64
	lastBytesRecv uint64
	warningShown  bool

	// Callbacks
	onWarning func(remainingMinutes int)
	onTimeout func()

	// For stopping the monitor
	stopCh chan struct{}
}

// NewIdleMonitor creates a new IdleMonitor with the specified timeout and warning period.
// The warning callback is triggered when (timeout - warningPeriod) has elapsed.
// The timeout callback is triggered when the full timeout has elapsed.
func NewIdleMonitor(timeout, warningPeriod time.Duration) *IdleMonitor {
	return &IdleMonitor{
		timeout:       timeout,
		warningPeriod: warningPeriod,
		enabled:       false,
	}
}

// SetCallbacks sets the callback functions for warning and timeout events.
// onWarning receives the remaining minutes before timeout.
// onTimeout is called when the idle timeout is reached.
func (m *IdleMonitor) SetCallbacks(onWarning func(remainingMinutes int), onTimeout func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onWarning = onWarning
	m.onTimeout = onTimeout
}

// Start begins monitoring for idle activity.
// It should be called when a VPN connection is established.
func (m *IdleMonitor) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If already running, don't start another goroutine
	if m.enabled {
		return
	}

	m.lastActivity = time.Now()
	m.warningShown = false
	m.enabled = true
	m.stopCh = make(chan struct{})

	go m.monitor()
}

// Stop stops the idle monitoring.
// It should be called when the VPN connection is terminated.
func (m *IdleMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.enabled {
		return
	}

	m.enabled = false
	if m.stopCh != nil {
		close(m.stopCh)
		m.stopCh = nil
	}
}

// UpdateActivity updates the activity tracker with current traffic stats.
// If traffic has changed since the last update, the idle timer is reset.
func (m *IdleMonitor) UpdateActivity(bytesSent, bytesRecv uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if bytesSent != m.lastBytesSent || bytesRecv != m.lastBytesRecv {
		m.lastActivity = time.Now()
		m.lastBytesSent = bytesSent
		m.lastBytesRecv = bytesRecv
		m.warningShown = false
	}
}

// ResetActivity resets the idle timer without requiring traffic changes.
// This can be used when the user explicitly interacts with the VPN.
func (m *IdleMonitor) ResetActivity() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastActivity = time.Now()
	m.warningShown = false
}

// IdleTime returns the duration since the last activity.
func (m *IdleMonitor) IdleTime() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.enabled || m.lastActivity.IsZero() {
		return 0
	}
	return time.Since(m.lastActivity)
}

// IsEnabled returns whether the monitor is currently active.
func (m *IdleMonitor) IsEnabled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.enabled
}

// Timeout returns the configured timeout duration.
func (m *IdleMonitor) Timeout() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.timeout
}

// monitor is the main loop that checks for idle timeout.
func (m *IdleMonitor) monitor() {
	// Check every 30 seconds
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.checkIdle()
		}
	}
}

// checkIdle checks the current idle state and triggers callbacks as needed.
func (m *IdleMonitor) checkIdle() {
	m.mu.Lock()

	if !m.enabled {
		m.mu.Unlock()
		return
	}

	idleTime := time.Since(m.lastActivity)
	warningTime := m.timeout - m.warningPeriod
	onWarning := m.onWarning
	onTimeout := m.onTimeout
	warningShown := m.warningShown
	timeout := m.timeout
	warningPeriod := m.warningPeriod

	// Check for full timeout
	if idleTime >= timeout {
		m.enabled = false
		m.mu.Unlock()
		if onTimeout != nil {
			onTimeout()
		}
		return
	}

	// Check for warning threshold
	if idleTime >= warningTime && !warningShown {
		m.warningShown = true
		m.mu.Unlock()
		if onWarning != nil {
			remainingMinutes := int(warningPeriod.Minutes())
			if remainingMinutes < 1 {
				remainingMinutes = 1
			}
			onWarning(remainingMinutes)
		}
		return
	}

	m.mu.Unlock()
}
