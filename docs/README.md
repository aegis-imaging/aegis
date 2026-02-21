# AEGIS — Shared Research & Documentation

This `docs/` folder is the shared knowledge base for AEGIS agents, developers, and collaborators.

## Purpose

Research, analysis, and findings are saved here so they can be reused across agent sessions, shared with human collaborators, and built upon incrementally. This avoids redundant web searches and ensures citations are consistent across the codebase.

## Conventions

- **Check here first** before doing web searches — the answer may already be documented
- Save research findings here before incorporating them into product documents
- Include full citations (author, journal, year, DOI/URL) — not just a URL
- Note source type: peer-reviewed journal article, preprint, government primary source, commercial report
- **Never save PHI (Protected Health Information) or CBI (Confidential Business Information)** in this folder or anywhere in the repository

## For Agents

When given a research task, save your findings to `docs/research/<topic>.md` before using them in product documents. Future agents will check here first before doing redundant searches.

## Current Research Files

| File | Contents |
|------|----------|
| `research/medical-imaging-deidentification.md` | De-identification failures, face reconstruction from MRI, burned-in PHI, NIH DMS policy, HIPAA Safe Harbor, DICOM PS3.15, MIDI-B challenge, market sizing |
| `research/mri-protocol-compliance.md` | MRI acquisition parameter ranges, consortia protocols (ADNI4, HCP, ABCD, UK Biobank, ENIGMA), tolerance recommendations, mrQA tool, Enhanced vs Classic DICOM |
| `research/mri-defacing-tools-comparison.md` | Tool comparison (afni_refacer, DeepDefacer, PyDeface, mri_deface, Quickshear), success rates, speed benchmarks, Docker size, licensing |
| `research/encog-competitive-analysis.md` | Competitive landscape analysis, Encog and adjacent competitors |
| `research/dual-market-strategy.md` | Dual-market strategy analysis (academic + commercial) |

## Directory Structure

| Directory | Contents |
|-----------|----------|
| `research/` | Peer-reviewed papers, policy documents, market reports, competitive analysis |
