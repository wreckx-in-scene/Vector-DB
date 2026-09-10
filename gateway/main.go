package main

import (
	"log"
	"net/http"
	"os"
)

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	enginePort := getenv("ENGINE_ADDR", "http://127.0.0.1:9090")
	ollamaAddr := getenv("OLLAMA_ADDR", "http://127.0.0.1:11434")
	publicPort := getenv("PORT", "8080")
	staticDir := getenv("STATIC_DIR", ".")

	engine := NewEngineClient(enginePort)
	ollama := NewOllamaClient(ollamaAddr)
	gw := &Gateway{engine: engine, ollama: ollama}

	mux := http.NewServeMux()

	// ── DEMO VECTOR ENDPOINTS ── pure proxies to the C++ engine.
	mux.HandleFunc("GET /search", func(w http.ResponseWriter, r *http.Request) {
		engine.proxy(w, r, "/search")
	})
	mux.HandleFunc("POST /insert", func(w http.ResponseWriter, r *http.Request) {
		engine.proxy(w, r, "/insert")
	})
	mux.HandleFunc("DELETE /delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		engine.proxy(w, r, "/delete/"+r.PathValue("id"))
	})
	mux.HandleFunc("GET /items", func(w http.ResponseWriter, r *http.Request) {
		engine.proxy(w, r, "/items")
	})
	mux.HandleFunc("GET /benchmark", func(w http.ResponseWriter, r *http.Request) {
		engine.proxy(w, r, "/benchmark")
	})
	mux.HandleFunc("GET /hnsw-info", func(w http.ResponseWriter, r *http.Request) {
		engine.proxy(w, r, "/hnsw-info")
	})
	mux.HandleFunc("GET /stats", func(w http.ResponseWriter, r *http.Request) {
		engine.proxy(w, r, "/stats")
	})

	// ── DOCUMENT + RAG ENDPOINTS ── Go owns chunking, Ollama, and prompting;
	// the C++ engine is only ever asked to index/search embeddings it's given.
	mux.HandleFunc("POST /doc/insert", gw.handleDocInsert)
	mux.HandleFunc("DELETE /doc/delete/{id}", gw.handleDocDelete)
	mux.HandleFunc("GET /doc/list", func(w http.ResponseWriter, r *http.Request) {
		engine.proxy(w, r, "/docengine/list")
	})
	mux.HandleFunc("POST /doc/search", gw.handleDocSearch)
	mux.HandleFunc("POST /doc/ask", gw.handleDocAsk)

	// ── STATUS ── composed from Ollama + engine.
	mux.HandleFunc("GET /status", gw.handleStatus)

	// ── STATIC FRONTEND ──
	mux.Handle("GET /", http.FileServer(http.Dir(staticDir)))

	log.Println("=== VectorDB Gateway (Go) ===")
	log.Println("http://localhost:" + publicPort)
	log.Println("engine:", enginePort, "| ollama:", ollamaAddr)

	if err := http.ListenAndServe(":"+publicPort, withCORS(mux)); err != nil {
		log.Fatal(err)
	}
}
