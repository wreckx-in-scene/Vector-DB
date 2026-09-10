package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

// OllamaClient wraps the local Ollama REST API.
// Install: https://ollama.com
// Models:  ollama pull nomic-embed-text
//          ollama pull llama3.2
type OllamaClient struct {
	baseURL    string
	EmbedModel string
	GenModel   string
	httpShort  *http.Client // short timeout, for embeddings + availability checks
	httpLong   *http.Client // long timeout, for generation (LLMs can be slow)
}

func NewOllamaClient(baseURL string) *OllamaClient {
	return &OllamaClient{
		baseURL:    baseURL,
		EmbedModel: "nomic-embed-text",
		GenModel:   "llama3.2",
		httpShort:  &http.Client{Timeout: 30 * time.Second},
		httpLong:   &http.Client{Timeout: 180 * time.Second},
	}
}

func (o *OllamaClient) IsAvailable() bool {
	cli := &http.Client{Timeout: 2 * time.Second}
	resp, err := cli.Get(o.baseURL + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

type embedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type embedResponse struct {
	Embedding []float32 `json:"embedding"`
}

// Embed returns nil if Ollama is not running or the model isn't pulled.
func (o *OllamaClient) Embed(text string) []float32 {
	body, _ := json.Marshal(embedRequest{Model: o.EmbedModel, Prompt: text})
	resp, err := o.httpShort.Post(o.baseURL+"/api/embeddings", "application/json", bytes.NewReader(body))
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}
	defer resp.Body.Close()

	var out embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil
	}
	return out.Embedding
}

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type generateResponse struct {
	Response string `json:"response"`
}

// Generate returns an error string (matching the original C++ behavior) if
// Ollama is unavailable, rather than a Go error — callers pass this straight
// through to the "answer" field like the original did.
func (o *OllamaClient) Generate(prompt string) string {
	body, _ := json.Marshal(generateRequest{Model: o.GenModel, Prompt: prompt, Stream: false})
	resp, err := o.httpLong.Post(o.baseURL+"/api/generate", "application/json", bytes.NewReader(body))
	if err != nil || resp.StatusCode != http.StatusOK {
		return "ERROR: Ollama unavailable. Run: ollama serve"
	}
	defer resp.Body.Close()

	var out generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "ERROR: Ollama unavailable. Run: ollama serve"
	}
	return out.Response
}
