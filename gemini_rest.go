package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// GeminiRestClient handles direct REST API calls to Gemini with search capabilities
type GeminiRestClient struct {
	apiKey string
	model  string
	logger *logrus.Logger
}

// GeminiRequest represents the request structure for Gemini REST API
type GeminiRequest struct {
	Contents         []GeminiContent        `json:"contents"`
	Tools            []GeminiTool           `json:"tools,omitempty"`
	GenerationConfig GeminiGenerationConfig `json:"generationConfig,omitempty"`
	SafetySettings   []GeminiSafetySetting  `json:"safetySettings,omitempty"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiTool struct {
	GoogleSearch map[string]interface{} `json:"google_search"`
}

type GeminiGenerationConfig struct {
	Temperature     float32 `json:"temperature,omitempty"`
	TopP            float32 `json:"topP,omitempty"`
	TopK            int32   `json:"topK,omitempty"`
	MaxOutputTokens int32   `json:"maxOutputTokens,omitempty"`
}

type GeminiSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// GeminiResponse represents the response from Gemini REST API
type GeminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
}

type GeminiCandidate struct {
	Content           GeminiContent            `json:"content"`
	FinishReason      string                   `json:"finishReason"`
	Index             int                      `json:"index"`
	SafetyRatings     []GeminiSafetyRating     `json:"safetyRatings"`
	GroundingMetadata *GeminiGroundingMetadata `json:"groundingMetadata,omitempty"`
}

type GeminiSafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

type GeminiGroundingMetadata struct {
	SearchEntryPoint *GeminiSearchEntryPoint `json:"searchEntryPoint,omitempty"`
	GroundingChunks  []GeminiGroundingChunk  `json:"groundingChunks,omitempty"`
}

type GeminiSearchEntryPoint struct {
	RenderedContent string `json:"renderedContent"`
}

type GeminiGroundingChunk struct {
	Web *GeminiWebChunk `json:"web,omitempty"`
}

type GeminiWebChunk struct {
	URI   string `json:"uri"`
	Title string `json:"title"`
}

func NewGeminiRestClient(apiKey string, model string, logger *logrus.Logger) *GeminiRestClient {
	return &GeminiRestClient{
		apiKey: apiKey,
		model:  model,
		logger: logger,
	}
}

func (g *GeminiRestClient) GenerateContentWithSearch(prompt string) (CheckResult, error) {
	result := CheckResult{
		ConditionMet: false,
		ResultData:   make(map[string]interface{}),
	}

	// Prepare the request with Google Search enabled
	request := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: prompt},
				},
			},
		},
		Tools: []GeminiTool{
			{
				GoogleSearch: map[string]interface{}{},
			},
		},
		GenerationConfig: GeminiGenerationConfig{
			Temperature:     0.1,
			TopP:            0.8,
			TopK:            40,
			MaxOutputTokens: 4000,
		},
		SafetySettings: []GeminiSafetySetting{
			{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_NONE"},
			{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_NONE"},
			{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_NONE"},
			{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_NONE"},
		},
	}

	// Convert to JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return result, fmt.Errorf("failed to marshal request: %w", err)
	}

	g.logger.WithFields(logrus.Fields{
		"request_size": len(requestBody),
		"has_search":   true,
		"model":        g.model,
		"tools":        "google_search",
		"request_body": string(requestBody),
	}).Info("Sending Gemini REST API request with Google Search")

	// Make the REST API call (API key via header per docs)
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", g.model)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return result, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", g.apiKey)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return result, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return result, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, fmt.Errorf("failed to read response: %w", err)
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return result, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	g.logger.WithFields(logrus.Fields{
		"candidates_count": len(geminiResp.Candidates),
		"response_size":    len(body),
		"raw_response":     string(body),
	}).Info("Received Gemini REST API response")

	if len(geminiResp.Candidates) == 0 {
		return result, fmt.Errorf("no candidates in Gemini response")
	}

	if len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return result, fmt.Errorf("no content parts in Gemini response")
	}

	// Extract the response text
	responseText := geminiResp.Candidates[0].Content.Parts[0].Text

	// Log grounding metadata if available (shows search results used)
	if geminiResp.Candidates[0].GroundingMetadata != nil {
		g.logger.WithFields(logrus.Fields{
			"search_entry_point": geminiResp.Candidates[0].GroundingMetadata.SearchEntryPoint,
			"grounding_chunks":   len(geminiResp.Candidates[0].GroundingMetadata.GroundingChunks),
		}).Info("Gemini used Google Search data")

		// Add search metadata to result
		result.ResultData["grounding_metadata"] = geminiResp.Candidates[0].GroundingMetadata
		result.ResultData["used_search"] = true
	} else {
		g.logger.Warn("No grounding metadata found - Google Search may not have been used")
		result.ResultData["used_search"] = false
	}

	g.logger.WithFields(logrus.Fields{
		"response_text": responseText,
	}).Debug("Raw Gemini response text")

	// Parse the response (try JSON first, then text analysis)
	var geminiResult map[string]interface{}
	if err := json.Unmarshal([]byte(responseText), &geminiResult); err != nil {
		// If JSON parsing fails, try to extract from markdown JSON block
		if jsonStart := bytes.Index([]byte(responseText), []byte("```json")); jsonStart != -1 {
			jsonStart += 7 // Skip "```json"
			if jsonEnd := bytes.Index([]byte(responseText)[jsonStart:], []byte("```")); jsonEnd != -1 {
				jsonContent := responseText[jsonStart : jsonStart+jsonEnd]
				if err := json.Unmarshal([]byte(jsonContent), &geminiResult); err == nil {
					if conditionMet, ok := geminiResult["condition_met"].(bool); ok {
						result.ConditionMet = conditionMet
					}
					result.ResultData = mergeMaps(result.ResultData, geminiResult)
					result.ResultData["parsing_method"] = "json_from_markdown"
					return result, nil
				}
			}
		}

		// Fallback to text analysis
		result.ConditionMet = containsConditionMet(responseText)
		result.ResultData["raw_response"] = responseText
		result.ResultData["parsing_method"] = "text_analysis"
	} else {
		if conditionMet, ok := geminiResult["condition_met"].(bool); ok {
			result.ConditionMet = conditionMet
		}
		result.ResultData = mergeMaps(result.ResultData, geminiResult)
		result.ResultData["parsing_method"] = "json_parsing"
	}

	return result, nil
}

func containsConditionMet(text string) bool {
	return bytes.Contains([]byte(text), []byte("\"condition_met\": true")) ||
		bytes.Contains([]byte(text), []byte("condition met")) ||
		bytes.Contains([]byte(text), []byte("true")) && bytes.Contains([]byte(text), []byte("condition"))
}

func mergeMaps(map1, map2 map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range map1 {
		result[k] = v
	}
	for k, v := range map2 {
		result[k] = v
	}
	return result
}
