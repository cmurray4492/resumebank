# resumebank.biz

A recruiting site connecting candidates and employers, built in Go. See `@SPEC.md` for the original
project brief. This repo is being built in phases. **Phase 1** (accounts, profiles, job postings,
file uploads, and search), **Phase 2** (employer &lt;-&gt; candidate messaging and candidate-only job
voting), **Phase 3a** (sitemap.xml and the hourly search-index rebuild), and **Phase 3b** (the
embeddings/RAG candidate&lt;-&gt;job matching features) are done. Novice-friendly deployment docs are
the only thing left (see "What's not built yet" below).

## Stack

- Go (standard library `net/http` + `html/template`), Bootstrap 5, Quill.js rich-text editor
- PostgreSQL with the `pgvector` extension, for candidate/job embeddings
- Ollama (local) running `nomic-embed-text` for the candidate/job matching features
- `jackc/pgx/v5` for database access, `bluemonday` for HTML sanitization, `bcrypt` for passwords

## Prerequisites

- Go 1.23+
- A PostgreSQL server with the `vector` extension available. The easiest way to get this locally is
  Docker:

  ```
  docker run --name resumebank-db -e POSTGRES_PASSWORD=dev -p 5432:5432 -d pgvector/pgvector:pg16
  ```

  (If you don't want to use Docker, install Postgres normally and additionally install the
  [pgvector](https://github.com/pgvector/pgvector) extension for your Postgres version.)

- [Ollama](https://ollama.com) running locally, with the `nomic-embed-text` model pulled
  (`ollama pull nomic-embed-text`), for the candidate/job matching features. **This is optional** —
  if Ollama isn't running, everything else in the app works normally; the matching pages just show a
  "temporarily unavailable" message instead of results (see "Notable Phase 3b decisions" below).

## Running locally

1. Set environment variables (a plain shell `export`/PowerShell `$env:` or a `.env`-loading tool of
   your choice both work — there's no framework-specific config format required):

   ```
   DATABASE_URL=postgres://postgres:dev@localhost:5432/postgres?sslmode=disable
   SESSION_SECRET=some-long-random-string
   PORT=8080
   ENV=development
   COOKIE_SECURE=false
   UPLOAD_DIR=./uploads
   BASE_URL=http://localhost:8080
   OLLAMA_URL=http://localhost:11434
   OLLAMA_EMBED_MODEL=nomic-embed-text
   ```

   `BASE_URL` is the absolute origin used to build `sitemap.xml`/`robots.txt` URLs; it defaults to
   `http://localhost:$PORT` if unset, but set it to your real domain in production. `OLLAMA_URL` and
   `OLLAMA_EMBED_MODEL` both have the defaults shown above if unset.

2. Apply database migrations:

   ```
   go run ./cmd/migrate
   ```

3. Start the server:

   ```
   go run ./cmd/server
   ```

4. Visit http://localhost:8080.

### Manual verification walkthrough

- Sign up as a candidate, fill in the required fields and resume (via the rich-text editor), save.
- On your profile's edit page, upload a PDF resume and up to 3 additional files; confirm they
  download correctly and are publicly visible on your profile page.
- Sign up as an employer, fill in your company profile, and post a job.
- Visit `/search`, toggle between "Candidates" and "Jobs", and confirm keyword search returns
  relevant results.
- While logged in as one candidate, try navigating directly to another candidate's `/edit` URL —
  you should get a 403 Forbidden. The same applies to employers editing another company's profile
  or jobs.
- As a candidate, thumbs up/down a job on its job page; click the same direction again to clear
  your vote. Confirm an employer account gets a 403 trying to vote.
- As an employer, click "Message" on a candidate's profile, send a message, then log in as that
  candidate and confirm it shows up in `/messages` with an unread badge in the nav; reply and
  confirm the employer sees it.
- Visit `/sitemap.xml` and `/robots.txt` and confirm every candidate/employer/job you created
  appears with a `<lastmod>` date. Sign up a new candidate, confirm it's immediately findable via
  `/search` (live), and confirm `SELECT * FROM search_index` in psql does *not* yet include it until
  the next hourly refresh (or run `REFRESH MATERIALIZED VIEW CONCURRENTLY search_index;` manually).
- With Ollama running, log in as an employer, go to "Match Candidates" in the nav, paste in a job
  description, and confirm it returns candidates ranked by relevance with a similarity percentage.
  Log in as a candidate, go to "Match Jobs", paste in a resume, and confirm the same in the other
  direction. Then stop Ollama and confirm both pages show "temporarily unavailable" instead of an
  error, and that signing up or editing a profile/job still works instantly either way.

## Running tests

Pure logic tests (validation, sanitization, slugs, local file storage, password hashing) run with no
database:

```
go test ./...
```

Repository, session, and HTTP-handler ownership/permission tests are gated on a real Postgres
database (they call `t.Skip` automatically if it isn't configured) to avoid needing mocks for SQL
behavior like full-text search ranking and unique constraints:

```
docker run --name resumebank-test-db -e POSTGRES_PASSWORD=dev -p 5433:5432 -d pgvector/pgvector:pg16
TEST_DATABASE_URL=postgres://postgres:dev@localhost:5433/postgres?sslmode=disable go test -p 1 ./...
```

(Use a separate database/port from your dev database — the test helper truncates all application
tables at the start of every test. The `-p 1` flag is required: `go test ./...` normally runs each
package's tests in a separate, concurrent process, and since every DB-gated test package shares the
one `TEST_DATABASE_URL`, concurrent truncations will otherwise wipe out another package's in-progress
test data and cause spurious failures. `-p 1` runs one package at a time against the shared database.)

Also useful:

```
go vet ./...
gofmt -l .   # should print nothing
```

## Notable Phase 1 decisions

- **Job `Title` is enforced as required** at the application validation layer, even though the spec
  doesn't explicitly mark it required — an untitled job listing isn't usable. This is a deliberate,
  narrow deviation, not silent scope creep.
- All candidate-uploaded files (PDF resume + up to 3 additional files) are public and downloadable
  without authentication, per spec.
- File storage is local disk (`UPLOAD_DIR`) for now. This works locally but will need to change to a
  persistent volume or object storage before a real deployment, since most PaaS platforms
  (Railway included) don't guarantee a persistent local filesystem across deploys — noted in
  `internal/storage` for the deployment phase.
- Search is plain PostgreSQL full-text search (`tsvector`/`tsquery`).

## Notable Phase 2 decisions

- **Messaging is candidate &lt;-&gt; employer only.** Sending a message where both parties have the
  same role (candidate-to-candidate or employer-to-employer) is rejected with a 400, matching the
  spec's framing of messaging as a way for "employers and candidates" to reach each other.
- Messaging has no separate `conversations` table — a "conversation" is derived at query time from
  `(sender_id, recipient_id)` pairs in `messages`, since a two-party direct-message model doesn't
  need one.
- Voting is a toggle: clicking the same direction (up or down) you already voted clears your vote,
  rather than requiring a separate "remove vote" control.

## Notable Phase 3a decisions

- **The hourly search index doesn't back live search.** `search_index` is a Postgres materialized
  view unioning candidates and jobs, refreshed hourly by a background goroutine
  (`internal/background.RunEvery`) via `REFRESH MATERIALIZED VIEW CONCURRENTLY`. It exists to satisfy
  the spec's literal "search index regenerated hourly" requirement, but `/search` intentionally
  queries the live `candidates`/`jobs` tables directly instead, so new signups and job postings are
  searchable immediately rather than lagging up to an hour behind. See
  `internal/db/migrations/0010_search_index.sql`.
- **The sitemap is generated once at server startup (blocking) and then every 24 hours** by the same
  background scheduler, cached in memory (`internal/sitemap.Cache`) so `GET /sitemap.xml` never hits
  the database. The startup generation means a fresh deploy never serves an empty sitemap while
  waiting for the first daily tick.
- `robots.txt` was added alongside the sitemap (not explicitly requested by the spec) since it's the
  standard way search engines discover a sitemap, and directly serves the spec's SEO requirement.

## Notable Phase 3b decisions

- **Embedding computation is async, best-effort, and self-healing.** After a candidate signs up (or
  changes their resume) or a job is posted (or its description changes), the server kicks off
  embedding computation in a background goroutine with its own timeout — the HTTP response is never
  delayed by it, and a failure (most likely Ollama not running) is just logged, never shown to the
  user. A background sweep (`internal/app.RefreshMissingEmbeddings`, every 5 minutes, plus once at
  startup) finds any candidate/job still missing an embedding and retries it. This means the
  matching feature degrades gracefully to "temporarily unavailable" instead of breaking anything
  else in the app when Ollama isn't running — appropriate for a feature the spec itself calls
  "in development."
- **`nomic-embed-text` is asymmetric**: text being indexed (a resume, a job description) is embedded
  with a `"search_document: "` prefix, and text being searched with (a pasted JD or resume at match
  time) gets a `"search_query: "` prefix instead — mixing these up measurably hurts retrieval
  quality. See `internal/embeddings/client.go`.
- **No ANN index (ivfflat/hnsw) on the embedding columns yet.** A brute-force
  `ORDER BY embedding <=> $1` scan is fast enough at the row counts this app has today, and those
  index types need representative data present to choose good parameters. Add one if the corpus
  grows large enough for it to matter.
- Match results show a similarity percentage (cosine similarity, since nomic-embed-text embeddings
  are normalized) but no rank-independent quality threshold; a corpus with only weakly-related
  entries will still return its "closest" results rather than an empty list, which is the intended
  interpretation of "will get better with time" as more profiles/jobs are added.

## What's not built yet

- Deployment configuration and novice-friendly deployment instructions (target: Railway)
