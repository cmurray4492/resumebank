# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

We're building the app described in @SPEC.MD. Read that file for general architectural tasks or to double check the exact database structure, tech stack, or application architecture.

Whenever you are working with any third-party library or something similar, you MUST look up the official documentation to ensure that you are working with up-to-date information.

Use the DocsExplorer subagent for efficient documentaton lookup.

Keep your replies extremely concise and focus on conveying key information. N o unnecessary fluff or long code snipets.

## Repository state

This is a Go web app implementing `@SPEC.md` (a recruiting website, "resumebank.biz"). Every feature in the spec is implemented, built in phases: **Phase 1** (accounts, candidate/employer/job CRUD with SEO pages, file uploads, Postgres full-text search), **Phase 2** (employer&lt;-&gt;candidate messaging, candidate-only job voting), **Phase 3a** (sitemap.xml + hourly search-index rebuild), **Phase 3b** (embeddings/RAG candidate&lt;-&gt;job matching via a local Ollama server), and deployment (Dockerfile + `DEPLOYMENT.md`, a from-scratch Railway walkthrough). `README.md` documents local dev; `DEPLOYMENT.md` documents deploying to Railway. Both have "Notable decisions" sections per phase worth reading before changing related code.

Confirmed technology choices (do not re-litigate; see `README.md` "Stack"): Go stdlib `net/http` + `html/template`, Bootstrap 5, Quill.js, PostgreSQL + `pgvector` (enabled now, used starting with the embeddings phase), `jackc/pgx/v5`, `bluemonday`, `bcrypt`, Go's standard `testing` package. Deployment target: Railway (not yet configured).

Build/lint/test commands, local run instructions, and the DB-gated test caveat (`go test -p 1 ./...`) are documented in `README.md` — read that before running tests or the server.

## Project brief summary

Per `@SPEC.md`, the intended stack and scope:

- **Backend**: Go, with a Go template engine, PostgreSQL.
- **Frontend**: Bootstrap, a WYSIWYG/rich text editor for resumes and job descriptions.
- **Embeddings/RAG**: a vector-capable database plus a local LLM RAG pipeline for two matching features — employers matching a pasted job description against the candidate resume corpus, and candidates matching a pasted resume against the active jobs corpus. Both are explicitly "in development" features to be surfaced as such in the UI.
- **Core entities**: Candidates (profile + required rich-text resume, required PDF resume upload + 3 additional public files), Employers (company profile + jobs), Job Postings.
- **Cross-cutting requirements**: SEO-formatted candidate/company/job pages, a sitemap regenerated daily, a search index regenerated hourly, combined candidate/job search with a checkbox toggle, candidate-only profile editing permissions, employer↔candidate messaging, and thumbs up/down voting on jobs (candidates only).

When starting implementation, re-read `@SPEC.md` in full for exact field lists and requirements (e.g. required vs. optional fields per entity) rather than relying on this summary.

## Working on this repo

Follow the existing patterns rather than introducing new ones: `internal/repo` for SQL access (one file per aggregate), `internal/web` for handlers (one file per feature, ownership checks inline per handler rather than generic middleware), `internal/db/migrations` for schema changes (new numbered `.sql` files, never edit an already-applied migration), and `web/templates/pages` for one template per page defining both a `content` and a `scripts` block. New third-party libraries or schema/architecture decisions with real trade-offs should still be confirmed with the user first.
