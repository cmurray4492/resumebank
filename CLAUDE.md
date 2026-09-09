# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

We're building the app described in @SPEC.MD. Read that file for general architectural tasks or to double check the exact database structure, tech stack, or application architecture.

Whenever you are working with any third-party library or something similar, you MUST look up the official documentation to ensure that you are working with up-to-date information.

Use the DocsExplorer subagent for efficient documentaton lookup.

Keep your replies extremely concise and focus on conveying key information. N o unnecessary fluff or long code snipets.

## Repository state

This repository currently contains only `@SPEC.md` — a project brief for a Go-based recruiting website ("resumebank.biz"). No source code, build tooling, or tests exist yet. There are no build/lint/test commands to document until the project is scaffolded.

## Project brief summary

Per `@SPEC.md`, the intended stack and scope:

- **Backend**: Go, with a Go template engine, PostgreSQL.
- **Frontend**: Bootstrap, a WYSIWYG/rich text editor for resumes and job descriptions.
- **Embeddings/RAG**: a vector-capable database plus a local LLM RAG pipeline for two matching features — employers matching a pasted job description against the candidate resume corpus, and candidates matching a pasted resume against the active jobs corpus. Both are explicitly "in development" features to be surfaced as such in the UI.
- **Core entities**: Candidates (profile + required rich-text resume, required PDF resume upload + 3 additional public files), Employers (company profile + jobs), Job Postings.
- **Cross-cutting requirements**: SEO-formatted candidate/company/job pages, a sitemap regenerated daily, a search index regenerated hourly, combined candidate/job search with a checkbox toggle, candidate-only profile editing permissions, employer↔candidate messaging, and thumbs up/down voting on jobs (candidates only).

When starting implementation, re-read `@SPEC.md` in full for exact field lists and requirements (e.g. required vs. optional fields per entity) rather than relying on this summary.

## Working on this repo

Since there is no existing code, architecture, or conventions to follow yet, the first substantive task in this repo is almost certainly scaffolding the Go module, database schema, and template structure from scratch. Confirm technology choices left open by the spec (template engine, vector database, embedding model, WYSIWYG editor) with the user before committing to one, since the spec explicitly says to select "an appropriate" option rather than naming one.
