package metrics

import (
	"sync"
	"sync/atomic"
	"time"

	"gemini-antiblock/logger"
)

// Metrics holds various performance metrics
type Metrics struct {
	// Request metrics
	TotalRequests       int64
	StreamingRequests   int64
	NonStreamingRequests int64
	
	// Response metrics
	SuccessfulRequests  int64
	FailedRequests      int64
	
	// Retry metrics
	TotalRetries        int64
	RetrySuccesses      int64
	RetryFailures       int64
	
	// Performance metrics
	AverageResponseTime time.Duration
	MaxResponseTime     time.Duration
	MinResponseTime     time.Duration
	
	// Memory metrics
	AccumulatedTextBytes int64
	MaxAccumulatedText   int64
	
	// Error metrics by type
	ErrorsByType        map[string]int64
	errorsMutex         sync.RWMutex
	
	// Response time tracking
	responseTimes       []time.Duration
	responseTimesMutex  sync.RWMutex
	maxResponseTimes    int // Maximum number of response times to keep
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	logger.LogInfo("Initializing metrics collection")
	
	return &Metrics{
		ErrorsByType:     make(map[string]int64),
		responseTimes:    make([]time.Duration, 0, 1000),
		maxResponseTimes: 1000,
		MinResponseTime:  time.Hour, // Initialize with a large value
	}
}

// IncrementTotalRequests increments the total request counter
func (m *Metrics) IncrementTotalRequests() {
	atomic.AddInt64(&m.TotalRequests, 1)
}

// IncrementStreamingRequests increments the streaming request counter
func (m *Metrics) IncrementStreamingRequests() {
	atomic.AddInt64(&m.StreamingRequests, 1)
}

// IncrementNonStreamingRequests increments the non-streaming request counter
func (m *Metrics) IncrementNonStreamingRequests() {
	atomic.AddInt64(&m.NonStreamingRequests, 1)
}

// IncrementSuccessfulRequests increments the successful request counter
func (m *Metrics) IncrementSuccessfulRequests() {
	atomic.AddInt64(&m.SuccessfulRequests, 1)
}

// IncrementFailedRequests increments the failed request counter
func (m *Metrics) IncrementFailedRequests() {
	atomic.AddInt64(&m.FailedRequests, 1)
}

// IncrementRetries increments the retry counter
func (m *Metrics) IncrementRetries() {
	atomic.AddInt64(&m.TotalRetries, 1)
}

// IncrementRetrySuccesses increments the retry success counter
func (m *Metrics) IncrementRetrySuccesses() {
	atomic.AddInt64(&m.RetrySuccesses, 1)
}

// IncrementRetryFailures increments the retry failure counter
func (m *Metrics) IncrementRetryFailures() {
	atomic.AddInt64(&m.RetryFailures, 1)
}

// RecordResponseTime records a response time and updates statistics
func (m *Metrics) RecordResponseTime(duration time.Duration) {
	m.responseTimesMutex.Lock()
	defer m.responseTimesMutex.Unlock()
	
	// Add to response times slice
	if len(m.responseTimes) >= m.maxResponseTimes {
		// Remove oldest entry if at capacity
		m.responseTimes = m.responseTimes[1:]
	}
	m.responseTimes = append(m.responseTimes, duration)
	
	// Update min/max
	if duration > m.MaxResponseTime {
		m.MaxResponseTime = duration
	}
	if duration < m.MinResponseTime {
		m.MinResponseTime = duration
	}
	
	// Calculate average
	var total time.Duration
	for _, rt := range m.responseTimes {
		total += rt
	}
	m.AverageResponseTime = total / time.Duration(len(m.responseTimes))
	
	logger.LogDebug("Recorded response time:", duration)
}

// RecordAccumulatedText records accumulated text size
func (m *Metrics) RecordAccumulatedText(size int64) {
	atomic.StoreInt64(&m.AccumulatedTextBytes, size)
	
	// Update max if necessary
	current := atomic.LoadInt64(&m.MaxAccumulatedText)
	if size > current {
		atomic.CompareAndSwapInt64(&m.MaxAccumulatedText, current, size)
	}
}

// IncrementErrorByType increments error counter for a specific type
func (m *Metrics) IncrementErrorByType(errorType string) {
	m.errorsMutex.Lock()
	defer m.errorsMutex.Unlock()
	
	m.ErrorsByType[errorType]++
	logger.LogDebug("Incremented error count for type:", errorType)
}

// GetSnapshot returns a snapshot of current metrics
func (m *Metrics) GetSnapshot() MetricsSnapshot {
	m.errorsMutex.RLock()
	m.responseTimesMutex.RLock()
	defer m.errorsMutex.RUnlock()
	defer m.responseTimesMutex.RUnlock()
	
	errorsCopy := make(map[string]int64)
	for k, v := range m.ErrorsByType {
		errorsCopy[k] = v
	}
	
	return MetricsSnapshot{
		TotalRequests:        atomic.LoadInt64(&m.TotalRequests),
		StreamingRequests:    atomic.LoadInt64(&m.StreamingRequests),
		NonStreamingRequests: atomic.LoadInt64(&m.NonStreamingRequests),
		SuccessfulRequests:   atomic.LoadInt64(&m.SuccessfulRequests),
		FailedRequests:       atomic.LoadInt64(&m.FailedRequests),
		TotalRetries:         atomic.LoadInt64(&m.TotalRetries),
		RetrySuccesses:       atomic.LoadInt64(&m.RetrySuccesses),
		RetryFailures:        atomic.LoadInt64(&m.RetryFailures),
		AverageResponseTime:  m.AverageResponseTime,
		MaxResponseTime:      m.MaxResponseTime,
		MinResponseTime:      m.MinResponseTime,
		AccumulatedTextBytes: atomic.LoadInt64(&m.AccumulatedTextBytes),
		MaxAccumulatedText:   atomic.LoadInt64(&m.MaxAccumulatedText),
		ErrorsByType:         errorsCopy,
		Timestamp:            time.Now(),
	}
}

// MetricsSnapshot represents a point-in-time snapshot of metrics
type MetricsSnapshot struct {
	TotalRequests        int64             `json:"total_requests"`
	StreamingRequests    int64             `json:"streaming_requests"`
	NonStreamingRequests int64             `json:"non_streaming_requests"`
	SuccessfulRequests   int64             `json:"successful_requests"`
	FailedRequests       int64             `json:"failed_requests"`
	TotalRetries         int64             `json:"total_retries"`
	RetrySuccesses       int64             `json:"retry_successes"`
	RetryFailures        int64             `json:"retry_failures"`
	AverageResponseTime  time.Duration     `json:"average_response_time"`
	MaxResponseTime      time.Duration     `json:"max_response_time"`
	MinResponseTime      time.Duration     `json:"min_response_time"`
	AccumulatedTextBytes int64             `json:"accumulated_text_bytes"`
	MaxAccumulatedText   int64             `json:"max_accumulated_text"`
	ErrorsByType         map[string]int64  `json:"errors_by_type"`
	Timestamp            time.Time         `json:"timestamp"`
}

// Global metrics instance
var globalMetrics *Metrics
var once sync.Once

// GetGlobalMetrics returns the global metrics instance
func GetGlobalMetrics() *Metrics {
	once.Do(func() {
		globalMetrics = NewMetrics()
	})
	return globalMetrics
}
