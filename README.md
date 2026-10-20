# KogniKwery (`kq`)

**KogniKwery (`kq`)** is a fast, local-first cognitive data engine written in Go. It translates plain-English questions into deterministic, read-only SQL queries executed against local files (CSV, TSV, JSON, Parquet via DuckDB) or relational databases, powered by any OpenAI-compatible local LLM server (`llama.cpp`, Ollama, LM Studio, vLLM).

## Status
🚧 **Active Rewrite (v0.1.0-dev):** Migrated from the original Python hackathon prototype (`v1.0-python-hackathon`) to a standalone Go binary.

## Quick Start (Development)
```bash
go build -o kq ./cmd/kq
./kq version
```

## Architecture
- `cmd/kq/` — CLI entry point
- `internal/ai/` — Zero-dependency HTTP client for OpenAI-compatible local LLM endpoints
- `internal/db/` — Tabular-to-Parquet normalizer, schema discovery, and read-only SQL execution engine
- `internal/tui/` — Interactive terminal UI
