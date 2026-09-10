package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type Gateway struct {
	engine *EngineClient
	ollama *OllamaClient
}

// ---- POST /doc/insert  {"title":"...","text":"..."} ----
// Chunks the text (in Go), embeds each chunk via Ollama, stores each chunk
// in the C++ engine's DocumentDB. Same response shape as the original.

type docInsertReq struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

func (gw *Gateway) handleDocInsert(w http.ResponseWriter, r *http.Request) {
	var req docInsertReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" || req.Text == "" {
		writeErr(w, http.StatusBadRequest, "need title and text")
		return
	}

	chunks := chunkText(req.Text, 250, 30)
	ids := make([]int, 0, len(chunks))
	var dims int

	for i, chunk := range chunks {
		emb := gw.ollama.Embed(chunk)
		if emb == nil {
			writeErr(w, http.StatusOK,
				"Ollama unavailable. Install from https://ollama.com then run: "+
					"ollama pull nomic-embed-text && ollama pull llama3.2")
			return
		}
		title := req.Title
		if len(chunks) > 1 {
			title = fmt.Sprintf("%s [%d/%d]", req.Title, i+1, len(chunks))
		}
		id, d, err := gw.engine.DocInsert(title, chunk, emb)
		if err != nil {
			writeErr(w, http.StatusBadGateway, "engine unreachable: "+err.Error())
			return
		}
		ids = append(ids, id)
		dims = d
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"ids": ids, "chunks": len(chunks), "dims": dims,
	})
}

// ---- POST /doc/search  {"question":"...","k":3} ----
// Fast retrieval for the UI visualizer — id/title/distance only, no text.

type docSearchReq struct {
	Question string `json:"question"`
	K        int    `json:"k"`
}

func (gw *Gateway) handleDocSearch(w http.ResponseWriter, r *http.Request) {
	var req docSearchReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Question == "" {
		writeErr(w, http.StatusBadRequest, "need question")
		return
	}
	if req.K == 0 {
		req.K = 3
	}

	qEmb := gw.ollama.Embed(req.Question)
	if qEmb == nil {
		writeErr(w, http.StatusOK, "Ollama unavailable")
		return
	}

	hits, err := gw.engine.DocSearch(qEmb, req.K)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "engine unreachable: "+err.Error())
		return
	}

	type lite struct {
		ID       int     `json:"id"`
		Title    string  `json:"title"`
		Distance float64 `json:"distance"`
	}
	out := make([]lite, len(hits))
	for i, h := range hits {
		out[i] = lite{h.ID, h.Title, h.Distance}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"contexts": out})
}

// ---- POST /doc/ask  {"question":"...","k":3} ----
// Full RAG pipeline: embed -> retrieve -> generate.

type docAskReq struct {
	Question string `json:"question"`
	K        int    `json:"k"`
}

func (gw *Gateway) handleDocAsk(w http.ResponseWriter, r *http.Request) {
	var req docAskReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Question == "" {
		writeErr(w, http.StatusBadRequest, "need question")
		return
	}
	if req.K == 0 {
		req.K = 3
	}

	// Step 1: embed the question
	qEmb := gw.ollama.Embed(req.Question)
	if qEmb == nil {
		writeErr(w, http.StatusOK, "Ollama unavailable")
		return
	}

	// Step 2: retrieve top-k relevant chunks
	hits, err := gw.engine.DocSearch(qEmb, req.K)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "engine unreachable: "+err.Error())
		return
	}

	// Step 3: build prompt
	var ctx strings.Builder
	for i, h := range hits {
		fmt.Fprintf(&ctx, "[%d] %s:\n%s\n\n", i+1, h.Title, h.Text)
	}
	prompt := "You are a helpful assistant. Answer the user's question directly. " +
		"Use the provided context if it contains relevant information. " +
		"If it doesn't, just use your own general knowledge. " +
		"IMPORTANT: Do NOT mention the 'context', 'provided text', or say things like 'the context doesn't mention'. " +
		"Just answer the question naturally.\n\n" +
		"Context:\n" + ctx.String() +
		"Question: " + req.Question + "\n\n" +
		"Answer:"

	// Step 4: generate answer
	answer := gw.ollama.Generate(prompt)

	// Step 5: docCount for the response
	stats, _ := gw.engine.DocStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"answer":   answer,
		"model":    gw.ollama.GenModel,
		"contexts": hits,
		"docCount": stats.DocCount,
	})
}

// ---- GET /status ----
// Combines Ollama availability with stats pulled from the C++ engine.

func (gw *Gateway) handleStatus(w http.ResponseWriter, r *http.Request) {
	up := gw.ollama.IsAvailable()
	docStats, _ := gw.engine.DocStats()
	demoStats, _ := gw.engine.DemoStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"ollamaAvailable": up,
		"embedModel":      gw.ollama.EmbedModel,
		"genModel":        gw.ollama.GenModel,
		"docCount":        docStats.DocCount,
		"docDims":         docStats.DocDims,
		"demoDims":        demoStats.Dims,
		"demoCount":       demoStats.Count,
	})
}

// ---- DELETE /doc/delete/{id} -> proxy to engine /docengine/delete/{id} ----

func (gw *Gateway) handleDocDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := strconv.Atoi(id); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	gw.engine.proxy(w, r, "/docengine/delete/"+id)
}
