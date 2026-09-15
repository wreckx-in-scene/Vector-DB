# VectorDB — Go Gateway + C++ Engine

A personal systems project exploring vector search, approximate nearest-neighbor algorithms, Retrieval-Augmented Generation (RAG), and service-based architecture.

The project uses a **C++ vector engine** for high-performance search algorithms and a **Go gateway** for HTTP APIs, document processing, and RAG orchestration.

---

## Features

- **Approximate Nearest-Neighbor (ANN) Search:** HNSW and KD-Tree implementations
- **Exact Vector Search:** Brute-force baseline comparison
- **Distance Metrics:** Cosine, Euclidean, and Manhattan distance algorithms
- **Vector Operations:** Insertion, deletion, and item listing
- **Inspection & Benchmarking:** Built-in algorithm benchmark runner and HNSW graph inspection
- **Document Processing:** Automatic text chunking and metadata management
- **Local AI Integration:** Local embeddings and LLM text generation powered by Ollama
- **RAG Pipeline:** Full Retrieval-Augmented Generation pipeline end-to-end
- **Interfaces:** Clean REST API alongside a browser-based user interface

---

## Architecture

```text
                    Browser
                       │
                       ▼
              Go Gateway :8080
             ┌─────────┴─────────┐
             │                   │
       Vector API            RAG Pipeline
             │                   │
             ▼                   ▼
       C++ Engine            Ollama :11434
        :9090
             │
             ▼
   HNSW · KD-Tree · Brute Force
             │
             ▼
       In-memory vectors
```

### C++ Engine
The C++ service handles all performance-sensitive vector operations:
- HNSW search
- KD-Tree search
- Brute-force search
- Distance calculations
- In-memory vector storage
- Document-vector retrieval

*Runs internally on:* `127.0.0.1:9090`

### Go Gateway
The Go service acts as the public API layer and handles:
- HTTP routing and frontend delivery
- Gateway-to-Engine inter-process communication
- Ollama API requests
- Text chunking and embedding generation
- RAG prompt construction

*Runs publicly on:* `localhost:8080`

### Why C++ + Go?
Separating the backend into two distinct services allows each language to do what it does best:
- **C++** provides low-level memory control and maximum execution speed for complex data structures and numerical distance calculations.
- **Go** offers clean concurrency, effortless HTTP routing, standard JSON handling, and clean service orchestration.

Beyond technical fit, this separation serves as a practical exercise in backend system boundaries, process isolation, HTTP IPC, and systems API design.

---

## Project Structure

```text
.
├── engine/
│   ├── main.cpp         # Vector database engine and internal API
│   └── httplib.h        # C++ header-only HTTP server library
│
├── gateway/
│   ├── main.go          # Application entry point & route setup
│   ├── engine.go        # HTTP client for C++ engine communication
│   ├── ollama.go        # Client for Ollama API
│   ├── chunk.go         # Document text chunking logic
│   ├── handlers.go      # Document & RAG API endpoints
│   └── go.mod           # Go module file
│
├── index.html           # Frontend browser UI
└── README.md
```

---

## Prerequisites & Requirements

- **C++ Compiler:** Supporting C++17 (`g++`, `clang++`, or MSVC)
- **Go:** Version 1.22+
- **Ollama:** Installed and running locally

Before running the gateway, pull the required local AI models:

```bash
ollama pull nomic-embed-text
ollama pull llama3.2
```

---

## Build Instructions

### 1. Build the C++ Engine

**On Linux / macOS:**
```bash
cd engine
g++ -std=c++17 -O2 main.cpp -o engine -lpthread
```

**On Windows (MSYS2 / MinGW):**
```bash
cd engine
g++ -std=c++17 -O2 main.cpp -o engine -lws2_32
```

### 2. Build the Go Gateway

```bash
cd ../gateway
go build -o gateway .
```

---

## Running the System

Start the services in three separate terminal windows:

### 1. Start Ollama
```bash
ollama serve
```

### 2. Start the C++ Engine
```bash
cd engine
./engine
# Engine listens on http://127.0.0.1:9090
```

### 3. Start the Go Gateway
```bash
cd gateway
./gateway
# Gateway listens on http://localhost:8080
```

Once running, access the web interface at **`http://localhost:8080`**.

---

## Configuration

All configuration variables are optional and default to local system settings:

| Variable | Default | Purpose |
| :--- | :--- | :--- |
| `ENGINE_HOST` | `127.0.0.1` | C++ engine bind host |
| `ENGINE_PORT` | `9090` | C++ engine bind port |
| `ENGINE_ADDR` | `http://127.0.0.1:9090` | Gateway → Engine target address |
| `OLLAMA_ADDR` | `http://127.0.0.1:11434` | Gateway → Ollama target address |
| `PORT` | `8080` | Go Gateway HTTP port |
| `STATIC_DIR` | *Auto-detected* | Directory serving `index.html` |

---

## API Reference

### Vector Search API

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `GET` | `/search` | K-nearest-neighbor search (`?v=f1,f2&k=5&metric=cosine&algo=hnsw`) |
| `POST` | `/insert` | Insert raw vector data |
| `DELETE` | `/delete/:id` | Delete vector by ID |
| `GET` | `/items` | List stored raw vectors |
| `GET` | `/benchmark` | Execute search speed comparisons across algorithms |
| `GET` | `/hnsw-info` | Retrieve internal HNSW graph metadata |
| `GET` | `/stats` | View general vector database metrics |

**Supported Search Algorithms:** `hnsw`, `kd-tree`, `brute-force`  
**Supported Distance Metrics:** `cosine`, `euclidean`, `manhattan`

---

### Document & RAG API

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `POST` | `/doc/insert` | Chunk, generate embeddings, and store a raw document |
| `GET` | `/doc/list` | List all stored documents and chunks |
| `DELETE` | `/doc/delete/:id` | Delete a specific document chunk |
| `POST` | `/doc/search` | Execute vector similarity search on document chunks |
| `POST` | `/doc/ask` | Retrieve relevant chunk context and synthesize LLM answer |
| `GET` | `/status` | View gateway system health and database status |

#### Workflow Examples

**Document Insertion Pipeline:**
```text
Document Text ──> Text Chunking ──> Embedding Generation (Ollama) ──> C++ Engine Storage
```
```json
// POST /doc/insert
{
  "title": "Operating Systems",
  "text": "An operating system manages hardware resources..."
}
```

**RAG Pipeline:**
```text
Question ──> Generate Embedding ──> Vector Search ──> Fetch Chunks ──> Synthesize Prompt ──> LLM Output
```
```json
// POST /doc/ask
{
  "question": "What does an operating system do?",
  "k": 3
}
```

---

### Internal Engine API (C++ Engine)

These endpoints are used internally by the Go Gateway and are not meant for public UI consumption:

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `POST` | `/docengine/insert` | Store a pre-computed vector embedding with text context |
| `POST` | `/docengine/search` | Search vector space using query embedding |
| `GET` | `/docengine/list` | Fetch raw document chunks |
| `DELETE` | `/docengine/delete/:id` | Remove chunk entry from memory |
| `GET` | `/docengine/stats` | View internal engine memory stats |

---

## What I Used This Project To Learn

This repository served as a hands-on exploration of core systems and search fundamentals:

- **Graph & Spatial Indexing:** Implementing HNSW graphs and KD-Trees from scratch.
- **RAG Architecture:** Building an end-to-end local retrieval pipeline without external libraries (LangChain, LlamaIndex, etc.).
- **Inter-Process Systems:** Managing dual-process HTTP communication and clean API contracts.
- **Performance Profiling:** Measuring trade-offs between exact nearest-neighbor search (Brute-force) and approximate graph methods (HNSW).
- **Local AI Execution:** Interfacing directly with locally running LLMs and embedding providers via REST APIs.

---

## Scope & Future Improvements

This project stores all vectors **in-memory** and is built primarily for architectural experimentation, learning, and benchmarking.

### Roadmap Features
- [ ] Persistent disk storage (write-ahead log or memory-mapped files)
- [ ] Concurrency safety and lock-free graph updates in C++
- [ ] Fine-grained HNSW hyperparameter tuning (efConstruction, efSearch)
- [ ] Automated RAG context relevance evaluations
- [ ] Vector sharding & multi-node distribution experimentation
- [ ] Containerized deployment with `docker-compose`
