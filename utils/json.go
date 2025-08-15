package utils

import (
	"bytes"
	"encoding/json"
	"sync"

	"gemini-antiblock/logger"
)

// JSONProcessor provides optimized JSON processing with buffer pooling
type JSONProcessor struct {
	bufferPool sync.Pool
	bufferSize int
}

// NewJSONProcessor creates a new JSON processor with buffer pooling
func NewJSONProcessor(bufferSize int) *JSONProcessor {
	logger.LogInfo("Initializing JSON processor with buffer pooling")
	logger.LogDebug("JSON Buffer Size:", bufferSize)

	processor := &JSONProcessor{
		bufferSize: bufferSize,
		bufferPool: sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, bufferSize))
			},
		},
	}

	logger.LogInfo("JSON processor initialized successfully")
	return processor
}

// Marshal serializes the given value to JSON using buffer pooling
func (jp *JSONProcessor) Marshal(v interface{}) ([]byte, error) {
	buf := jp.bufferPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		jp.bufferPool.Put(buf)
	}()

	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(v); err != nil {
		logger.LogDebug("JSON marshal error:", err)
		return nil, err
	}

	// Remove trailing newline added by encoder.Encode
	data := buf.Bytes()
	if len(data) > 0 && data[len(data)-1] == '\n' {
		data = data[:len(data)-1]
	}

	// Create a copy to return since we're reusing the buffer
	result := make([]byte, len(data))
	copy(result, data)

	logger.LogDebug("JSON marshaled successfully, size:", len(result))
	return result, nil
}

// Unmarshal deserializes JSON data into the given value
func (jp *JSONProcessor) Unmarshal(data []byte, v interface{}) error {
	err := json.Unmarshal(data, v)
	if err != nil {
		logger.LogDebug("JSON unmarshal error:", err)
		return err
	}

	logger.LogDebug("JSON unmarshaled successfully, data size:", len(data))
	return nil
}

// MarshalToBuffer marshals JSON directly to a buffer (for streaming)
func (jp *JSONProcessor) MarshalToBuffer(v interface{}, buf *bytes.Buffer) error {
	encoder := json.NewEncoder(buf)
	err := encoder.Encode(v)
	if err != nil {
		logger.LogDebug("JSON marshal to buffer error:", err)
		return err
	}

	logger.LogDebug("JSON marshaled to buffer successfully")
	return nil
}

// GetBufferSize returns the configured buffer size
func (jp *JSONProcessor) GetBufferSize() int {
	return jp.bufferSize
}
