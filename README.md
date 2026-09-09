# resumebank.biz

A recruiting site connecting candidates and employers, built in Go. See `@SPEC.md` for the original
project brief. Every feature in the spec is implemented: accounts and SEO-formatted profiles, job
postings, file uploads, search, employer &lt;-&gt; candidate messaging, candidate-only job voting,
sitemap.xml + hourly search-index rebuild, and the embeddings/RAG candidate&lt;-&gt;job matching
features. Beyond the original spec, there's also a blog (`/blog`) and an admin panel (`/admin`) for
managing blog posts and editing any candidate/employer/job.

This file covers running the app **locally**. To deploy it to production, see
**[DEPLOYMENT.md](DEPLOYMENT.md)** — a from-scratch walkthrough for deploying to
[Railway](https://railway.app), written for someone who's never deployed anything before.

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
   `OLLAMA_EMBED_MODEL` both have the defaults shown above if unset. There's also `AUTO_MIGRATE`
   (default `false`), which makes `cmd/server` apply pending migrations on startup instead of you
   running `cmd/migrate` separately — leave it off locally (step 2 below covers migrations), it's
   meant for deployment (see `DEPLOYMENT.md`).

2. Apply database migrations:

   ```
   go run ./cmd/migrate
   ```

3. Start the server:

   ```
   go run ./cmd/server
   ```

4. Visit http://localhost:8080.

5. To use the admin panel, create an admin account (there's no public admin signup):

   ```
   go run ./cmd/createadmin -email=admin@example.com -password=some-long-password
   ```

   Then log in at http://localhost:8080/admin/login. Running this again with the same email resets
   that admin's password. It refuses to touch an email that already belongs to a candidate/employer
   account, to prevent an email typo from accidentally granting admin access to an existing user.

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
- Create an admin account (see step 5 above), log in at `/admin`, and confirm the dashboard shows
  correct candidate/employer/job/blog-post counts. Create a blog post with "Published" checked and
  confirm it shows up at `/blog` and its own `/blog/{slug}` page; uncheck "Published" and confirm it
  404s publicly again; delete it and confirm it's gone.
- From the admin panel, edit a candidate, a company, and a job you didn't create as that account —
  confirm the change persists on the public page (ownership checks don't apply to admin edits by
  design). Log out of admin and confirm `/admin` redirects to `/admin/login`. While logged in as a
  regular candidate or employer, confirm visiting `/admin` also redirects to `/admin/login` — a
  public-site session must never grant admin access.

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

## Notable blog/admin decisions

These two features were added on top of the original spec at the user's request, not from `@SPEC.md`.

- **Admins are `users` rows with `role='admin'`**, reusing the existing password-hashing, session,
  and CSRF infrastructure rather than a parallel account system — see
  `internal/db/migrations/0012_admin_role.sql`. There is no public admin signup route; the only way
  to create one is `cmd/createadmin`, which refuses to convert an existing candidate/employer email
  to admin (to prevent a typo from granting admin access to the wrong account) and otherwise creates
  the account or resets its password if it already exists.
- **The admin session is a separate cookie** (`resumebank_admin_session`, scoped to `Path=/admin`)
  from the public site's session cookie, and admin identity is loaded into a completely separate
  request-context key (`AdminUserFromContext`, never `CurrentUser`). This means a public
  candidate/employer session can never grant admin access even if someone reused the same browser,
  and the public site's navbar/templates can never accidentally show admin-only state. See
  `internal/auth/admin.go`.
- **Admin panel uses its own layout** (`admin_base.html.tmpl`, no public navbar/footer) via a second
  `Renderer.RenderAdmin` method — see `internal/render/render.go`. It carries a `noindex, nofollow`
  meta tag and is deliberately left out of `sitemap.xml`.
- **Admin can edit, but not delete, candidates/employers/jobs** (only their own blog posts support
  delete) — matching exactly what was asked for; edits bypass ownership checks entirely (by design;
  that's the point of an admin panel) but reuse the same repo `Update` methods and validation as the
  owner-facing forms, and still trigger re-embedding when a resume/job description changes.
- **Blog posts have no `conversations`-style extra tooling** (no categories, tags, or comments) —
  title, rich-text body (via the same Quill/sanitize pipeline as resumes and job descriptions),
  author name, and a published/draft flag. `published_at` is set the first time a post is published
  and left alone on later edits, so it reflects the original publish date, not the last-edited date.

## Notable deployment decisions

- **The app is deployed as a Docker image** (see `Dockerfile`) rather than relying on Railway's
  zero-config builder, since it needs two separate binaries (`cmd/server`, `cmd/migrate`) and a
  Dockerfile is the more predictable, explicit option. Templates and static assets are compiled into
  the binary via `go:embed`, so the runtime image needs nothing but the compiled binaries.
- **`AUTO_MIGRATE=true`** (an opt-in env var, off by default — see `internal/config`) makes
  `cmd/server` apply pending migrations on startup before serving traffic, so a Railway deploy needs
  no separate migration step. It's safe to leave on permanently: the migration runner tracks what's
  already applied and is a no-op when there's nothing pending.
- File storage and Ollama both need Railway **Volumes** to persist across redeploys (uploaded
  resumes, and downloaded Ollama models respectively) — see `DEPLOYMENT.md` Steps 4 and 6.
