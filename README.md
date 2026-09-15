from pathlib import Path

content = r'''# VectorDB — Go Gateway + C++ Engine

A personal systems project exploring **vector search, approximate nearest-neighbor algorithms, RAG, and service-based architecture**.

The project uses a **C++ vector engine** for the core search algorithms and a **Go gateway** for HTTP APIs, document processing, and RAG orchestration.

## Features

- HNSW approximate nearest-neighbor search
- KD-Tree search
- Brute-force vector search
- Cosine, Euclidean, and Manhattan distance metrics
- Vector insertion and deletion
- Search benchmarking
- HNSW graph inspection
- Document chunking
- Local embeddings and LLM generation using Ollama
- Retrieval-Augmented Generation (RAG)
- REST API
- Browser-based UI

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