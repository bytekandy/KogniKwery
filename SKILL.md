---
name: kognikwery
description: Query local tabular files (CSV, TSV, Parquet) or SQL databases in plain English using the KogniKwery (kq) Go binary and a local open-source LLM.
---

# KogniKwery (`kq`) Agent Skill

Use this skill when the user wants to inspect schemas, run analytical queries, or ask plain-English questions against local data files or databases using `kq`.

## Build & Verify Binary
```bash
go build -o kq ./cmd/kq
./kq version
```

## Ask a Question Against Data
```bash
./kq ask "<PLAIN_ENGLISH_QUESTION>" --data <PATH_TO_FILE_OR_DB>
```
