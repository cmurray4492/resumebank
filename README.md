# resumebank.biz

A recruiting site connecting candidates and employers, built in Go. See `@SPEC.md` for the original
project brief. This repo is being built in phases. **Phase 1** (accounts, profiles, job postings,
file uploads, and search) and **Phase 2** (employer &lt;-&gt; candidate messaging and candidate-only
job voting) are done. The sitemap/search-index background jobs and the embeddings/RAG "matching"
features are planned for a later phase (see "What's not built yet" below).

## Stack

- Go (standard library `net/http` + `html/template`), Bootstrap 5, Quill.js rich-text editor
- PostgreSQL with the `pgvector` extension (enabled now for a future embeddings phase)
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
   ```

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
- Search is plain PostgreSQL full-text search (`tsvector`/`tsquery`), which needs no separate index
  build step — this covers the spec's search requirement without the hourly rebuild job (planned
  for a later phase alongside the sitemap job, using the same corpus differently).

## Notable Phase 2 decisions

- **Messaging is candidate &lt;-&gt; employer only.** Sending a message where both parties have the
  same role (candidate-to-candidate or employer-to-employer) is rejected with a 400, matching the
  spec's framing of messaging as a way for "employers and candidates" to reach each other.
- Messaging has no separate `conversations` table — a "conversation" is derived at query time from
  `(sender_id, recipient_id)` pairs in `messages`, since a two-party direct-message model doesn't
  need one.
- Voting is a toggle: clicking the same direction (up or down) you already voted clears your vote,
  rather than requiring a separate "remove vote" control.

## What's not built yet (planned for later phases)

- Sitemap generation (daily) and a separate hourly search-index rebuild job
- The two embeddings/RAG "matching" features (employer pastes a JD to find candidates; candidate
  pastes a resume to find jobs), which will use Ollama running locally with an embedding model
  (e.g. `nomic-embed-text`) against the `pgvector` column already enabled in the schema
- Deployment configuration and novice-friendly deployment instructions (target: Railway)
