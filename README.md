# VectorDB — Go Gateway + C++ Engine

A vector database with three search algorithms (HNSW, KD-Tree, Brute Force) and
a local RAG pipeline, split across two processes:

| Process | Language | Role |
|---|---|---|
| **engine** | C++ | Owns the algorithms — HNSW / KD-Tree / Brute Force, distance metrics, the in-memory vector stores. Internal-only, binds to `127.0.0.1:9090`. |
| **gateway** | Go | Public REST API on `:8080`. Talks to Ollama (embeddings + generation), chunks documents, builds RAG prompts, serves the frontend, and proxies the demo-vector endpoints straight through to the engine. |

The public REST API is **unchanged** from the original single-binary version —
`index.html` was not modified at all. Every endpoint the frontend calls
(`/search`, `/insert`, `/delete/:id`, `/items`, `/benchmark`, `/hnsw-info`,
`/stats`, `/doc/insert`, `/doc/list`, `/doc/delete/:id`, `/doc/search`,
`/doc/ask`, `/status`) behaves exactly as before.

## Why split it this way

- The **HNSW/KD-Tree graph work is CPU-bound and pointer-heavy** — it stays in
  C++ so it's free of GC pauses and keeps the hand-written distance kernels.
- **Everything else is I/O-bound orchestration** — HTTP routing, calling
  Ollama, chunking text, building prompts — which is exactly where Go's
  stdlib (`net/http`, `encoding/json`) is more ergonomic than hand-rolled
  JSON string building in C++.
- The engine never talks to Ollama and never does text chunking anymore —
  it only ever indexes and searches embedding vectors it's handed. That logic
  moved to `gateway/ollama.go` and `gateway/chunk.go`.

```
Browser
   │
   ▼
Go Gateway  (:8080, public)
   │              │
   │ (demo/doc    │ (embed / generate)
   │  vector ops) ▼
   │           Ollama (:11434)
   ▼
C++ Engine  (127.0.0.1:9090, internal only)
   HNSW · KD-Tree · Brute Force · DocumentDB
```

## Project structure

```
.
├── engine/
│   ├── main.cpp       ← C++ engine: algorithms + internal REST API (unchanged core classes)
│   └── httplib.h       ← single-header HTTP server library (unchanged)
├── gateway/
│   ├── main.go         ← entrypoint, route wiring
│   ├── engine.go        ← HTTP client to the C++ engine
│   ├── ollama.go        ← Ollama client (ported from the old C++ OllamaClient)
│   ├── chunk.go          ← text chunker (ported from the old C++ chunkText)
│   ├── handlers.go       ← /doc/*, /status handlers (RAG orchestration)
│   └── go.mod
└── index.html            ← frontend (unmodified)
```

## Prerequisites

1. **g++** with C++17 support
2. **Go** 1.22+
3. **Ollama** — [ollama.com](https://ollama.com), then:
   ```
   ollama pull nomic-embed-text
   ollama pull llama3.2
   ```

## Build

```bash
# C++ engine
cd engine
g++ -std=c++17 -O2 main.cpp -o engine -lpthread    # add -lws2_32 instead of -lpthread on Windows/MSYS2

# Go gateway
cd ../gateway
go build -o gateway .
```

## Run

Three terminals:

```bash
# 1. Ollama
ollama serve

# 2. C++ engine (internal, port 9090)
cd engine && ./engine

# 3. Go gateway (public, port 8080) — run from the repo root so it finds index.html,
#    or set STATIC_DIR to point at it.
cd gateway && ./gateway
```

Then open `http://localhost:8080`.

### Config (env vars, all optional)

| Var | Default | Applies to |
|---|---|---|
| `ENGINE_HOST` / `ENGINE_PORT` | `127.0.0.1` / `9090` | engine bind address |
| `ENGINE_ADDR` | `http://127.0.0.1:9090` | gateway → engine target |
| `OLLAMA_ADDR` | `http://127.0.0.1:11434` | gateway → Ollama target |
| `PORT` | `8080` | gateway public port |
| `STATIC_DIR` | `.` | where the gateway looks for `index.html` |

## REST API reference

Identical to the original — see the table below. Everything under `/doc*`
is now handled in Go; everything else is proxied to the C++ engine unchanged.

### Demo vector endpoints (proxied to C++ engine)

| Method | Endpoint | Description |
|---|---|---|
| GET | `/search?v=f1,f2,...&k=5&metric=cosine&algo=hnsw` | K-NN search |
| POST | `/insert` | Insert a demo vector |
| DELETE | `/delete/:id` | Delete by ID |
| GET | `/items` | List all demo vectors |
| GET | `/benchmark?v=...&k=5&metric=cosine` | Compare all 3 algorithms |
| GET | `/hnsw-info` | HNSW graph structure and layer stats |
| GET | `/stats` | Database statistics |

### Document & RAG endpoints (handled in Go, retrieval delegated to C++ engine)

| Method | Endpoint | Body | Description |
|---|---|---|---|
| POST | `/doc/insert` | `{"title":"...","text":"..."}` | Chunk, embed, and store document |
| GET | `/doc/list` | — | List all stored documents |
| DELETE | `/doc/delete/:id` | — | Delete document chunk |
| POST | `/doc/search` | `{"question":"...","k":3}` | Retrieval only (no generation) |
| POST | `/doc/ask` | `{"question":"...","k":3}` | RAG: retrieve + generate |
| GET | `/status` | — | Ollama status + doc/demo DB stats |

### Internal-only engine endpoints (not for direct public use)

| Method | Endpoint | Body | Description |
|---|---|---|---|
| POST | `/docengine/insert` | `{"title","text","embedding"}` | Index a pre-computed embedding |
| POST | `/docengine/search` | `{"embedding","k"}` | Nearest-neighbor retrieval |
| GET | `/docengine/list` | — | List stored chunks |
| DELETE | `/docengine/delete/:id` | — | Delete a chunk |
| GET | `/docengine/stats` | — | Doc DB stats |

## License

MIT — use this however you want.
