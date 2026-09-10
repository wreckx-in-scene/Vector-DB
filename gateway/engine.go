package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// EngineClient talks to the C++ vector engine (main.cpp), which is expected
// to be reachable only from this gateway — it binds to 127.0.0.1 by default
// and is never exposed to the internet directly.
type EngineClient struct {
	baseURL string
	http    *http.Client
}

func NewEngineClient(baseURL string) *EngineClient {
	return &EngineClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// proxy forwards an incoming request to the engine unchanged (same method,
// query string, and body) and copies the engine's response straight back to
// the client. Used for the demo-vector endpoints, which don't need any
// gateway-side logic — they're pure C++ engine calls.
func (e *EngineClient) proxy(w http.ResponseWriter, r *http.Request, enginePath string) {
	url := e.baseURL + enginePath
	if r.URL.RawQuery != "" {
		url += "?" + r.URL.RawQuery
	}

	var body io.Reader
	if r.Body != nil {
		b, _ := io.ReadAll(r.Body)
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequest(r.Method, url, body)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "engine request build failed")
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.http.Do(req)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "engine unreachable: "+err.Error())
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// ---- typed doc-engine calls (used by the RAG/doc handlers) ----

type docInsertResp struct {
	ID   int `json:"id"`
	Dims int `json:"dims"`
}

func (e *EngineClient) DocInsert(title, text string, emb []float32) (int, int, error) {
	payload := map[string]any{"title": title, "text": text, "embedding": emb}
	b, _ := json.Marshal(payload)

	resp, err := e.http.Post(e.baseURL+"/docengine/insert", "application/json", bytes.NewReader(b))
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	var out docInsertResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, 0, err
	}
	return out.ID, out.Dims, nil
}

type DocContext struct {
	ID       int     `json:"id"`
	Title    string  `json:"title"`
	Text     string  `json:"text"`
	Distance float64 `json:"distance"`
}

type docSearchResp struct {
	Contexts []DocContext `json:"contexts"`
}

func (e *EngineClient) DocSearch(emb []float32, k int) ([]DocContext, error) {
	payload := map[string]any{"embedding": emb, "k": k}
	b, _ := json.Marshal(payload)

	resp, err := e.http.Post(e.baseURL+"/docengine/search", "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out docSearchResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Contexts, nil
}

type docStatsResp struct {
	DocCount int `json:"docCount"`
	DocDims  int `json:"docDims"`
}

func (e *EngineClient) DocStats() (docStatsResp, error) {
	resp, err := e.http.Get(e.baseURL + "/docengine/stats")
	if err != nil {
		return docStatsResp{}, err
	}
	defer resp.Body.Close()

	var out docStatsResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return docStatsResp{}, err
	}
	return out, nil
}

type demoStatsResp struct {
	Count int `json:"count"`
	Dims  int `json:"dims"`
}

func (e *EngineClient) DemoStats() (demoStatsResp, error) {
	resp, err := e.http.Get(e.baseURL + "/stats")
	if err != nil {
		return demoStatsResp{}, err
	}
	defer resp.Body.Close()

	var out demoStatsResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return demoStatsResp{}, err
	}
	return out, nil
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":%q}`, msg)
}
