// Package cost provides cost tracking for VPN sessions.
package cost

import (
	"fmt"
	"sync"
	"time"
)

// Tracker tracks the cost of a VPN session based on hourly rate and duration.
type Tracker struct {
	mu         sync.RWMutex
	started    time.Time
	hourlyRate float64
	currency   string
	isTracking bool
}

// NewTracker creates a new Tracker instance.
func NewTracker() *Tracker {
	return &Tracker{}
}

// Start begins tracking cost with the given hourly rate and currency.
func (t *Tracker) Start(hourlyRate float64, currency string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.started = time.Now()
	t.hourlyRate = hourlyRate
	t.currency = currency
	t.isTracking = true
}

// Stop stops tracking cost.
func (t *Tracker) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.isTracking = false
}

// IsTracking returns true if the tracker is currently tracking a session.
func (t *Tracker) IsTracking() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.isTracking
}

// CurrentCost returns the current accumulated cost and currency.
// Returns (0, currency) if not tracking.
func (t *Tracker) CurrentCost() (cost float64, currency string) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.isTracking {
		return 0, t.currency
	}

	duration := time.Since(t.started)
	hours := duration.Hours()
	return t.hourlyRate * hours, t.currency
}

// Duration returns the current session duration.
// Returns 0 if not tracking.
func (t *Tracker) Duration() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if !t.isTracking {
		return 0
	}
	return time.Since(t.started)
}

// HourlyRate returns the configured hourly rate.
func (t *Tracker) HourlyRate() float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.hourlyRate
}

// Currency returns the configured currency.
func (t *Tracker) Currency() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.currency
}

// FormatCost returns a formatted string of the current cost (e.g., "$0.0052").
func (t *Tracker) FormatCost() string {
	cost, currency := t.CurrentCost()
	return FormatCostValue(cost, currency)
}

// FormatDuration returns a formatted string of the session duration (e.g., "00:30:15").
func (t *Tracker) FormatDuration() string {
	duration := t.Duration()
	return FormatDurationValue(duration)
}

// FormatHourlyRate returns a formatted string of the hourly rate (e.g., "$0.0104/hr").
func (t *Tracker) FormatHourlyRate() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return formatRateValue(t.hourlyRate, t.currency)
}

// FormatCostValue formats a cost value with the appropriate currency symbol.
func FormatCostValue(cost float64, currency string) string {
	symbol := "$"
	if currency == "EUR" {
		symbol = "€"
	}
	return fmt.Sprintf("%s%.4f", symbol, cost)
}

// FormatDurationValue formats a duration as HH:MM:SS.
func FormatDurationValue(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

// formatRateValue formats a rate with the appropriate currency symbol and /hr suffix.
func formatRateValue(rate float64, currency string) string {
	symbol := "$"
	if currency == "EUR" {
		symbol = "€"
	}
	return fmt.Sprintf("%s%.4f/hr", symbol, rate)
}
