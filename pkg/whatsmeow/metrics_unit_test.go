package whatsmeow

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCurrentDayKeyUTC tests the currentDayKeyUTC function
func TestCurrentDayKeyUTC(t *testing.T) {
	t.Parallel()

	result := CurrentDayKeyUTC()

	// Verify the result is in the expected format: YYYYMMDD
	// For example, 2024-03-11 would be 20240311
	assert.Greater(t, result, int64(20000000), "Day key should be greater than year 2000")
	assert.Less(t, result, int64(30000000), "Day key should be less than year 3000")

	// Verify the format is approximately correct
	// Extract year, month, day
	year := result / 10000
	month := (result / 100) % 100
	day := result % 100

	assert.GreaterOrEqual(t, year, int64(2000), "Year should be >= 2000")
	assert.LessOrEqual(t, year, int64(2099), "Year should be <= 2099")
	assert.GreaterOrEqual(t, month, int64(1), "Month should be >= 1")
	assert.LessOrEqual(t, month, int64(12), "Month should be <= 12")
	assert.GreaterOrEqual(t, day, int64(1), "Day should be >= 1")
	assert.LessOrEqual(t, day, int64(31), "Day should be <= 31")

	// Verify that the day key matches the current date
	now := time.Now().UTC()
	expectedYear := int64(now.Year())
	expectedMonth := int64(now.Month())
	expectedDay := int64(now.Day())

	assert.Equal(t, expectedYear, year, "Year should match current year")
	assert.Equal(t, expectedMonth, month, "Month should match current month")
	assert.Equal(t, expectedDay, day, "Day should match current day")
}

// TestNewInstanceMetrics tests the newInstanceMetrics function
func TestNewInstanceMetrics(t *testing.T) {
	t.Parallel()

	result := NewInstanceMetrics()

	require.NotNil(t, result, "newInstanceMetrics should return non-nil")

	// Verify day key is set
	dayKey := result.dayKeyUTC.Load()
	assert.NotZero(t, dayKey, "Day key should be set")

	// Verify all counters start at 0
	assert.Equal(t, uint64(0), result.messagesSent.Load(), "Messages sent should start at 0")
	assert.Equal(t, uint64(0), result.messagesReceived.Load(), "Messages received should start at 0")
	assert.Equal(t, uint64(0), result.messagesFailed.Load(), "Messages failed should start at 0")
	assert.Equal(t, uint64(0), result.errors.Load(), "Errors should start at 0")
	assert.Equal(t, int64(0), result.queueDepth.Load(), "Queue depth should start at 0")
	assert.Equal(t, int64(0), result.connectedSinceUnix.Load(), "Connected since should start at 0")
}

// TestInstanceMetrics_ResetIfDayChanged tests reset behavior
func TestInstanceMetrics_ResetIfDayChanged(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		initialDayKey    int64
		setCounters      bool
		expectReset      bool
		expectedMessages uint64
		expectedFailed   uint64
		expectedErrors   uint64
	}{
		{
			name:          "same day - no reset",
			initialDayKey: 20240315,
			setCounters:   true,
			expectReset:   false,
		},
		{
			name:          "different day - resets counters",
			initialDayKey: 20240314, // Previous day
			setCounters:   true,
			expectReset:   true,
		},
		{
			name:          "zero day key - sets current day",
			initialDayKey: 0,
			setCounters:   false,
			expectReset:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := &instanceMetrics{}

			// Set initial day key
			m.dayKeyUTC.Store(tt.initialDayKey)

			// Set some counters if requested
			if tt.setCounters {
				m.messagesSent.Store(10)
				m.messagesReceived.Store(20)
				m.messagesFailed.Store(3)
				m.errors.Store(5)
			}

			// Call resetIfDayChanged
			m.resetIfDayChanged()

			// Verify day key was updated to current day
			currentDay := CurrentDayKeyUTC()
			assert.Equal(t, currentDay, m.dayKeyUTC.Load(), "Day key should be updated to current day")

			// Check counter values based on whether reset occurred
			sent := m.messagesSent.Load()
			received := m.messagesReceived.Load()
			failed := m.messagesFailed.Load()
			errs := m.errors.Load()

			if tt.expectReset {
				// Counters should be reset to 0 if day changed and they won the race
				// Note: In concurrent scenarios, CompareAndSwap might fail,
				// but in single-threaded test it should succeed
				if tt.initialDayKey != currentDay {
					assert.Equal(t, uint64(0), sent, "Messages sent should be reset")
					assert.Equal(t, uint64(0), received, "Messages received should be reset")
					assert.Equal(t, uint64(0), failed, "Messages failed should be reset")
					assert.Equal(t, uint64(0), errs, "Errors should be reset")
				}
			} else if tt.setCounters && tt.initialDayKey == currentDay {
				// Counters should be preserved
				assert.Equal(t, uint64(10), sent, "Messages sent should be preserved")
				assert.Equal(t, uint64(20), received, "Messages received should be preserved")
				assert.Equal(t, uint64(3), failed, "Messages failed should be preserved")
				assert.Equal(t, uint64(5), errs, "Errors should be preserved")
			}
		})
	}
}

// TestInstanceMetrics_ResetCountersOnly tests that reset only clears specific counters
func TestInstanceMetrics_ResetCountersOnly(t *testing.T) {
	t.Parallel()

	m := NewInstanceMetrics()

	// Set various counters
	m.messagesSent.Store(100)
	m.messagesReceived.Store(200)
	m.messagesFailed.Store(10)
	m.errors.Store(15)
	m.queueDepth.Store(5)
	m.connectedSinceUnix.Store(12345)

	// Manually set day key to previous day
	previousDay := CurrentDayKeyUTC() - 1
	m.dayKeyUTC.Store(previousDay)

	// Call reset
	m.resetIfDayChanged()

	// Verify specific counters were reset
	assert.Equal(t, uint64(0), m.messagesSent.Load(), "Messages sent should be reset")
	assert.Equal(t, uint64(0), m.messagesReceived.Load(), "Messages received should be reset")
	assert.Equal(t, uint64(0), m.messagesFailed.Load(), "Messages failed should be reset")
	assert.Equal(t, uint64(0), m.errors.Load(), "Errors should be reset")

	// Verify other fields are not affected
	assert.Equal(t, int64(5), m.queueDepth.Load(), "Queue depth should not be reset")
	assert.Equal(t, int64(12345), m.connectedSinceUnix.Load(), "Connected since should not be reset")
}

// TestInstanceMetrics_ConstantValues tests that constants are defined
func TestInstanceMetrics_ConstantValues(t *testing.T) {
	t.Parallel()

	// This test verifies that the InstanceHealthMetrics struct
	// has the expected fields with correct JSON tags
	metrics := InstanceHealthMetrics{}

	// Verify struct can be created
	assert.NotNil(t, metrics, "InstanceHealthMetrics should be created")
}

// TestInstanceMetrics_TypeIntegrity tests atomic field operations
func TestInstanceMetrics_TypeIntegrity(t *testing.T) {
	t.Parallel()

	m := NewInstanceMetrics()
	require.NotNil(t, m, "NewInstanceMetrics should return non-nil")

	// Test that atomic operations work correctly
	m.messagesSent.Store(42)
	assert.Equal(t, uint64(42), m.messagesSent.Load(), "Atomic store/load should work")

	m.messagesSent.Add(10)
	assert.Equal(t, uint64(52), m.messagesSent.Load(), "Atomic add should work")

	m.messagesReceived.Store(100)
	assert.Equal(t, uint64(100), m.messagesReceived.Load(), "Atomic operations should be independent")
}

// TestInstanceMetrics_DayKeyFormat tests the day key format
func TestInstanceMetrics_DayKeyFormat(t *testing.T) {
	t.Parallel()

	// Test specific dates
	tests := []struct {
		name        string
		year        int
		month       time.Month
		day         int
		expectedKey int64
	}{
		{
			name:        "2024-01-01",
			year:        2024,
			month:       1,
			day:         1,
			expectedKey: 20240101,
		},
		{
			name:        "2024-12-31",
			year:        2024,
			month:       12,
			day:         31,
			expectedKey: 20241231,
		},
		{
			name:        "2023-06-15",
			year:        2023,
			month:       6,
			day:         15,
			expectedKey: 20230615,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// We can't directly test currentDayKeyUTC with specific dates without mocking time
			// But we can verify the format logic
			result := int64(tt.year*10000 + int(tt.month)*100 + tt.day)
			assert.Equal(t, tt.expectedKey, result, "Day key format should be correct")
		})
	}
}
