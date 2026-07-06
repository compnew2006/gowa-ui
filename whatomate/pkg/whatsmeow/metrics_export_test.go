package whatsmeow

// Exported functions for testing metrics.go

// CurrentDayKeyUTC exports currentDayKeyUTC for testing
func CurrentDayKeyUTC() int64 {
	return currentDayKeyUTC()
}

// NewInstanceMetrics exports newInstanceMetrics for testing
func NewInstanceMetrics() *instanceMetrics {
	return newInstanceMetrics()
}
