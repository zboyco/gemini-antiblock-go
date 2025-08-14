package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"gemini-antiblock/config"
	"gemini-antiblock/logger"
	"gemini-antiblock/streaming"
)

// ProxyHandler handles proxy requests to Gemini API
type ProxyHandler struct {
	Config *config.Config
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(cfg *config.Config) *ProxyHandler {
	return &ProxyHandler{Config: cfg}
}

// BuildUpstreamHeaders builds headers for upstream requests
func (h *ProxyHandler) BuildUpstreamHeaders(reqHeaders http.Header) http.Header {
	headers := make(http.Header)
	
	// Copy specific headers
	if auth := reqHeaders.Get("Authorization"); auth != "" {
		headers.Set("Authorization", auth)
	}
	if apiKey := reqHeaders.Get("X-Goog-Api-Key"); apiKey != "" {
		headers.Set("X-Goog-Api-Key", apiKey)
	}
	if contentType := reqHeaders.Get("Content-Type"); contentType != "" {
		headers.Set("Content-Type", contentType)
	}
	if accept := reqHeaders.Get("Accept"); accept != "" {
		headers.Set("Accept", accept)
	}
	
	return headers
}

// InjectSystemPrompt injects system prompt to ensure [done] token
func (h *ProxyHandler) InjectSystemPrompt(body map[string]interface{}) {
	newSystemPromptPart := map[string]interface{}{
		"text": "Your message must end with [done] to signify the end of your output.",
	}
	
	// Case 1: systemInstruction field is missing or null
	if _, exists := body["systemInstruction"]; !exists {
		body["systemInstruction"] = map[string]interface{}{
			"parts": []interface{}{newSystemPromptPart},
		}
		return
	}
	
	systemInstruction, ok := body["systemInstruction"].(map[string]interface{})
	if !ok {
		body["systemInstruction"] = map[string]interface{}{
			"parts": []interface{}{newSystemPromptPart},
		}
		return
	}
	
	// Case 2: systemInstruction exists, but parts array is missing, null, or not an array
	parts, ok := systemInstruction["parts"].([]interface{})
	if !ok {
		systemInstruction["parts"] = []interface{}{newSystemPromptPart}
		return
	}
	
	// Case 3: systemInstruction and parts array both exist - append to existing array
	systemInstruction["parts"] = append(parts, newSystemPromptPart)
}

// HandleStreamingPost handles streaming POST requests
func (h *ProxyHandler) HandleStreamingPost(w http.ResponseWriter, r *http.Request) {
	urlObj, _ := url.Parse(r.URL.String())
	upstreamURL := h.Config.UpstreamURLBase + urlObj.Path
	if urlObj.RawQuery != "" {
		upstreamURL += "?" + urlObj.RawQuery
	}
	
	logger.LogInfo("=== NEW STREAMING REQUEST ===")
	logger.LogInfo("Upstream URL:", upstreamURL)
	logger.LogInfo("Request method:", r.Method)
	logger.LogInfo("Content-Type:", r.Header.Get("Content-Type"))
	
	// Read and parse request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.LogError("Failed to read request body:", err)
		JSONError(w, 400, "Failed to read request body", err.Error())
		return
	}
	
	var requestBody map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &requestBody); err != nil {
		logger.LogError("Failed to parse request body:", err)
		JSONError(w, 400, "Invalid JSON in request body", err.Error())
		return
	}
	
	logger.LogDebug(fmt.Sprintf("Request body size: %d bytes", len(bodyBytes)))
	
	if contents, ok := requestBody["contents"].([]interface{}); ok {
		logger.LogDebug(fmt.Sprintf("Parsed request body with %d messages", len(contents)))
	}
	
	// Inject system prompt
	h.InjectSystemPrompt(requestBody)
	
	// Create upstream request
	modifiedBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		logger.LogError("Failed to marshal modified request body:", err)
		JSONError(w, 500, "Internal server error", "Failed to process request body")
		return
	}
	
	logger.LogInfo("=== MAKING INITIAL REQUEST ===")
	upstreamHeaders := h.BuildUpstreamHeaders(r.Header)
	
	upstreamReq, err := http.NewRequest("POST", upstreamURL, bytes.NewReader(modifiedBodyBytes))
	if err != nil {
		logger.LogError("Failed to create upstream request:", err)
		JSONError(w, 500, "Internal server error", "Failed to create upstream request")
		return
	}
	
	upstreamReq.Header = upstreamHeaders
	
	client := &http.Client{}
	initialResponse, err := client.Do(upstreamReq)
	if err != nil {
		logger.LogError("Failed to make initial request:", err)
		JSONError(w, 502, "Bad Gateway", "Failed to connect to upstream server")
		return
	}
	
	logger.LogInfo(fmt.Sprintf("Initial response status: %d %s", initialResponse.StatusCode, initialResponse.Status))
	
	// Initial failure: return standardized error
	if initialResponse.StatusCode != http.StatusOK {
		logger.LogError("=== INITIAL REQUEST FAILED ===")
		logger.LogError("Status:", initialResponse.StatusCode)
		logger.LogError("Status Text:", initialResponse.Status)
		
		// Read error response
		errorBody, _ := io.ReadAll(initialResponse.Body)
		initialResponse.Body.Close()
		
		// Try to parse as JSON error
		var errorResp map[string]interface{}
		if json.Unmarshal(errorBody, &errorResp) == nil {
			if errorObj, ok := errorResp["error"].(map[string]interface{}); ok {
				if _, hasStatus := errorObj["status"]; !hasStatus {
					if code, ok := errorObj["code"].(float64); ok {
						errorObj["status"] = StatusToGoogleStatus(int(code))
					}
				}
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.WriteHeader(initialResponse.StatusCode)
			json.NewEncoder(w).Encode(errorResp)
			return
		}
		
		// Fallback to standard error
		message := "Request failed"
		if initialResponse.StatusCode == 429 {
			message = "Resource has been exhausted (e.g. check quota)."
		}
		JSONError(w, initialResponse.StatusCode, message, string(errorBody))
		return
	}
	
	logger.LogInfo("=== INITIAL REQUEST SUCCESSFUL - STARTING STREAM PROCESSING ===")
	
	// Set up streaming response
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	
	// Process stream with retry logic
	err = streaming.ProcessStreamAndRetryInternally(
		h.Config,
		initialResponse.Body,
		w,
		requestBody,
		upstreamURL,
		r.Header,
	)
	
	if err != nil {
		logger.LogError("=== UNHANDLED EXCEPTION IN STREAM PROCESSOR ===")
		logger.LogError("Exception:", err)
	}
	
	initialResponse.Body.Close()
	logger.LogInfo("Streaming response completed")
}

// HandleNonStreaming handles non-streaming requests
func (h *ProxyHandler) HandleNonStreaming(w http.ResponseWriter, r *http.Request) {
	urlObj, _ := url.Parse(r.URL.String())
	upstreamURL := h.Config.UpstreamURLBase + urlObj.Path
	if urlObj.RawQuery != "" {
		upstreamURL += "?" + urlObj.RawQuery
	}
	
	upstreamHeaders := h.BuildUpstreamHeaders(r.Header)
	
	var body io.Reader
	if r.Method != "GET" && r.Method != "HEAD" {
		body = r.Body
	}
	
	upstreamReq, err := http.NewRequest(r.Method, upstreamURL, body)
	if err != nil {
		JSONError(w, 500, "Internal server error", "Failed to create upstream request")
		return
	}
	
	upstreamReq.Header = upstreamHeaders
	
	client := &http.Client{}
	resp, err := client.Do(upstreamReq)
	if err != nil {
		JSONError(w, 502, "Bad Gateway", "Failed to connect to upstream server")
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		// Handle error response
		errorBody, _ := io.ReadAll(resp.Body)
		
		var errorResp map[string]interface{}
		if json.Unmarshal(errorBody, &errorResp) == nil {
			if errorObj, ok := errorResp["error"].(map[string]interface{}); ok {
				if _, hasStatus := errorObj["status"]; !hasStatus {
					if code, ok := errorObj["code"].(float64); ok {
						errorObj["status"] = StatusToGoogleStatus(int(code))
					}
				}
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.WriteHeader(resp.StatusCode)
			json.NewEncoder(w).Encode(errorResp)
			return
		}
		
		JSONError(w, resp.StatusCode, resp.Status, string(errorBody))
		return
	}
	
	// Copy response headers
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// ServeHTTP implements the http.Handler interface
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger.LogInfo("=== WORKER REQUEST ===")
	logger.LogInfo("Method:", r.Method)
	logger.LogInfo("URL:", r.URL.String())
	logger.LogInfo("User-Agent:", r.Header.Get("User-Agent"))
	logger.LogInfo("X-Forwarded-For:", r.Header.Get("X-Forwarded-For"))
	
	if r.Method == "OPTIONS" {
		logger.LogDebug("Handling CORS preflight request")
		HandleCORS(w, r)
		return
	}
	
	// Determine if this is a streaming request
	isStream := strings.Contains(strings.ToLower(r.URL.Path), "stream") ||
		strings.Contains(strings.ToLower(r.URL.Path), "sse") ||
		r.URL.Query().Get("alt") == "sse"
	
	logger.LogInfo("Detected streaming request:", isStream)
	
	if r.Method == "POST" && isStream {
		h.HandleStreamingPost(w, r)
		return
	}
	
	h.HandleNonStreaming(w, r)
}
