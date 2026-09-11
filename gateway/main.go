package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// resolveStaticDir finds index.html without requiring the caller to run the
// binary from a specific directory. If STATIC_DIR is set explicitly, it's
// trusted as-is (and a missing index.html there is a loud, fatal error —
// the user asked for that exact path). Otherwise we probe the handful of
// places people actually run this from: the current directory (repo root),
// one level up (running from gateway/), and the same two relative to the
// compiled binary's own location (running the binary from elsewhere, e.g.
// after `go build -o /usr/local/bin/gateway`).
//
// This exists because Go's http.FileServer does NOT 404 on a missing
// index.html — it silently serves a directory listing instead, which looks
// like "the UI is broken" rather than "wrong working directory."
func resolveStaticDir() string {
	if explicit := os.Getenv("STATIC_DIR"); explicit != "" {
		if _, err := os.Stat(filepath.Join(explicit, "index.html")); err != nil {
			log.Fatalf("STATIC_DIR=%s was set explicitly but has no index.html: %v", explicit, err)
		}
		return explicit
	}

	candidates := []string{"."}
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates, exeDir)
	}
	// also check one directory up from cwd and from the exe — covers
	// `cd gateway && ./gateway` and `cd gateway && go run .`
	more := make([]string, 0, len(candidates))
	for _, c := range candidates {
		more = append(more, filepath.Join(c, ".."))
	}
	candidates = append(candidates, more...)

	for _, dir := range candidates {
		if _, err := os.Stat(filepath.Join(dir, "index.html")); err == nil {
			return dir
		}
	}

	log.Println("WARNING: couldn't find index.html in the current directory, its parent, " +
		"the binary's directory, or the binary's parent directory. Serving \"" +
		"." + "\" — GET / will show a directory listing instead of the UI. " +
		"Set STATIC_DIR to the folder containing index.html to fix this.")
	return "."
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
	staticDir := resolveStaticDir()

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
	log.Println("engine:", enginePort, "| ollama:", ollamaAddr, "| serving UI from:", staticDir)

	if err := http.ListenAndServe(":"+publicPort, withCORS(mux)); err != nil {
		log.Fatal(err)
	}
}
