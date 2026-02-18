# AEGIS — Shared Research & Documentation

This `docs/` folder is the shared knowledge base for AEGIS agents, developers, and collaborators.

## Purpose

Research, analysis, and findings are saved here so they can be reused across agent sessions, shared with human collaborators, and built upon incrementally. This avoids redundant web searches and ensures citations are consistent across the codebase.

## Conventions

- Save research findings here before incorporating them into product documents
- Organize by topic in subdirectories (e.g., `research/`, `compliance/`, `market/`)
- Include full citations (author, journal, year, DOI/URL) — not just a URL
- Note when a source is a preprint, commercial report, or peer-reviewed journal article
- **Never save PHI (Protected Health Information) or CBI (Confidential Business Information)** in this folder or anywhere in the repository

## For Agents

When given a research task, save your findings to `docs/research/<topic>.md` before using them in product documents. Future agents will check here first before doing redundant searches.

## Subdirectories

| Directory | Contents |
|-----------|----------|
| `docs/research/` | Peer-reviewed papers, policy documents, market reports |
| `docs/compliance/` | Regulatory analysis, HIPAA/HITECH notes |
| `docs/market/` | Competitive analysis, market sizing |
