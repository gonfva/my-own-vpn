package cost

import (
	"strings"
	"testing"
	"time"
)

func TestNewTracker(t *testing.T) {
	tracker := NewTracker()
	if tracker == nil {
		t.Fatal("NewTracker() returned nil")
	}
	if tracker.IsTracking() {
		t.Error("new tracker should not be tracking")
	}
}

func TestTrackerStartStop(t *testing.T) {
	tracker := NewTracker()

	tracker.Start(0.01, "USD")
	if !tracker.IsTracking() {
		t.Error("tracker should be tracking after Start()")
	}
	if tracker.HourlyRate() != 0.01 {
		t.Errorf("expected hourly rate 0.01, got %f", tracker.HourlyRate())
	}
	if tracker.Currency() != "USD" {
		t.Errorf("expected currency USD, got %s", tracker.Currency())
	}

	tracker.Stop()
	if tracker.IsTracking() {
		t.Error("tracker should not be tracking after Stop()")
	}
}

func TestTrackerCurrentCost(t *testing.T) {
	tracker := NewTracker()

	// Before starting, cost should be 0
	cost, _ := tracker.CurrentCost()
	if cost != 0 {
		t.Errorf("expected cost 0 before start, got %f", cost)
	}

	// Start tracking
	tracker.Start(0.01, "USD") // $0.01/hour

	// Wait a small amount of time
	time.Sleep(100 * time.Millisecond)

	cost, currency := tracker.CurrentCost()
	if cost <= 0 {
		t.Error("expected positive cost after tracking")
	}
	if currency != "USD" {
		t.Errorf("expected USD, got %s", currency)
	}

	// Stop and verify cost is 0
	tracker.Stop()
	cost, _ = tracker.CurrentCost()
	if cost != 0 {
		t.Errorf("expected cost 0 after stop, got %f", cost)
	}
}

func TestTrackerDuration(t *testing.T) {
	tracker := NewTracker()

	// Before starting, duration should be 0
	if tracker.Duration() != 0 {
		t.Error("expected 0 duration before start")
	}

	tracker.Start(0.01, "USD")
	time.Sleep(50 * time.Millisecond)

	duration := tracker.Duration()
	if duration < 50*time.Millisecond {
		t.Errorf("expected duration >= 50ms, got %v", duration)
	}

	tracker.Stop()
	if tracker.Duration() != 0 {
		t.Error("expected 0 duration after stop")
	}
}

func TestTrackerFormatCost(t *testing.T) {
	tracker := NewTracker()
	tracker.Start(36.0, "USD") // $36/hour = $0.01/second

	time.Sleep(100 * time.Millisecond)

	formatted := tracker.FormatCost()
	if formatted == "" {
		t.Error("FormatCost() returned empty string")
	}
	// Should start with $ for USD
	if formatted[0] != '$' {
		t.Errorf("expected USD format to start with $, got %s", formatted)
	}

	// Test EUR formatting
	tracker.Stop()
	tracker.Start(36.0, "EUR")
	time.Sleep(50 * time.Millisecond)
	formatted = tracker.FormatCost()
	if !strings.HasPrefix(formatted, "€") {
		t.Errorf("expected EUR format to start with €, got %s", formatted)
	}
}

func TestTrackerFormatDuration(t *testing.T) {
	tracker := NewTracker()
	tracker.Start(0.01, "USD")
	time.Sleep(50 * time.Millisecond)

	formatted := tracker.FormatDuration()
	// Should be in HH:MM:SS format
	if len(formatted) != 8 || formatted[2] != ':' || formatted[5] != ':' {
		t.Errorf("expected HH:MM:SS format, got %s", formatted)
	}
}

func TestTrackerFormatHourlyRate(t *testing.T) {
	tracker := NewTracker()
	tracker.Start(0.0104, "USD")

	formatted := tracker.FormatHourlyRate()
	expected := "$0.0104/hr"
	if formatted != expected {
		t.Errorf("expected %s, got %s", expected, formatted)
	}

	tracker.Stop()
	tracker.Start(0.005, "EUR")
	formatted = tracker.FormatHourlyRate()
	expected = "€0.0050/hr"
	if formatted != expected {
		t.Errorf("expected %s, got %s", expected, formatted)
	}
}

func TestFormatCostValue(t *testing.T) {
	tests := []struct {
		cost     float64
		currency string
		expected string
	}{
		{0.0052, "USD", "$0.0052"},
		{1.2345, "USD", "$1.2345"},
		{0.0050, "EUR", "€0.0050"},
		{10.0000, "EUR", "€10.0000"},
	}

	for _, tc := range tests {
		result := FormatCostValue(tc.cost, tc.currency)
		if result != tc.expected {
			t.Errorf("FormatCostValue(%f, %s) = %s, expected %s",
				tc.cost, tc.currency, result, tc.expected)
		}
	}
}

func TestFormatDurationValue(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{0, "00:00:00"},
		{30 * time.Second, "00:00:30"},
		{5 * time.Minute, "00:05:00"},
		{1 * time.Hour, "01:00:00"},
		{1*time.Hour + 30*time.Minute + 15*time.Second, "01:30:15"},
		{25 * time.Hour, "25:00:00"},
	}

	for _, tc := range tests {
		result := FormatDurationValue(tc.duration)
		if result != tc.expected {
			t.Errorf("FormatDurationValue(%v) = %s, expected %s",
				tc.duration, result, tc.expected)
		}
	}
}

func TestTrackerConcurrency(t *testing.T) {
	tracker := NewTracker()

	// Start multiple goroutines accessing the tracker
	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			tracker.Start(0.01, "USD")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			tracker.Stop()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			tracker.CurrentCost()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			tracker.Duration()
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		<-done
	}
}
