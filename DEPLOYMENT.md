# Deploying resumebank.biz to Railway

This walks through deploying the app to [Railway](https://railway.app) from scratch, assuming you've
never used Railway before. It takes about 20-30 minutes the first time.

## What you'll end up with

Three services running in one Railway project:

1. **Your app** — built from this repo's `Dockerfile`.
2. **Postgres with pgvector** — a database, deployed from Railway's template marketplace.
3. **Ollama** (optional) — runs the local AI model that powers the "in development" candidate/job
   matching feature. Skip this and the app still works fine — the matching pages just show a
   friendly "temporarily unavailable" message instead of results.

## What it costs

Railway isn't free, but it's cheap for a small project like this:

- New accounts get a one-time **$5 trial credit** with no time limit on using it up.
- After that, the **Hobby plan is $5/month**, and that $5 is itself usage credit — a small app like
  this one (one web service, one small database, optionally one small Ollama service) typically
  costs at or near that $5/month minimum.
- You only pay for what you use (CPU/RAM/network/storage), billed by the second.

## Prerequisites

- This code pushed to a GitHub repository you control (if you're reading this in the repo, that's
  already done).
- A [Railway](https://railway.app) account (sign up with GitHub — it's the easiest option since
  you'll be deploying from a GitHub repo).

## Step 1 — Create the project and deploy the app

1. Go to [railway.app/new](https://railway.app/new) and click **"Deploy from GitHub repo"**.
2. Authorize Railway to access your GitHub account if prompted, then pick this repository.
3. Railway creates a project with one service for your app and starts a build automatically. It'll
   likely fail at first — that's expected, because it doesn't have the environment variables it
   needs yet. Continue to the next steps.
4. Railway detects the `Dockerfile` in this repo and uses it to build automatically — you don't need
   to configure anything for the build itself.

## Step 2 — Add Postgres with pgvector

The app needs the `pgvector` Postgres extension, which Railway's plain "Postgres" template does
*not* include. Use the dedicated template instead:

1. In your project, click **"+ New"** → **"Database"** → search for **"pgvector"** (or go to
   [railway.com/template](https://railway.com/template) and search "pgvector" there) and deploy it.
   This gives you a Postgres database with the `vector` extension already available.
2. Wait for it to finish deploying (a minute or two).

## Step 3 — Configure your app's environment variables

Click on your app's service (not the database), go to the **Variables** tab, and add these one at a
time. For any variable value below written as `${{...}}`, type the `${{` and Railway will show an
autocomplete of other services' variables to reference — don't type the service name blind, since it
depends on what Railway named your database service.

| Variable | Value | Notes |
|---|---|---|
| `DATABASE_URL` | reference your Postgres service's `DATABASE_URL` | e.g. `${{Postgres.DATABASE_URL}}` — pick from the autocomplete |
| `SESSION_SECRET` | a long random string | Generate one locally: `openssl rand -hex 32` (or any password generator — it just needs to be long and secret) |
| `ENV` | `production` | |
| `COOKIE_SECURE` | `true` | Required in production — Railway serves your app over HTTPS |
| `AUTO_MIGRATE` | `true` | Runs pending database migrations automatically on each deploy, so you never need a separate migration step |
| `BASE_URL` | `https://${{RAILWAY_PUBLIC_DOMAIN}}` | Used to build `sitemap.xml`/`robots.txt` links and password reset links correctly |
| `UPLOAD_DIR` | `/app/data` | See Step 4 — this must match the volume mount path |

Railway automatically provides `PORT` — you don't need to set it yourself.

**Password reset email.** Without any `SMTP_*` variables set, "forgot password" still works, but the
reset link only gets written to your app's deploy logs (Railway dashboard → your app service →
Deployments → View Logs) instead of being emailed — fine for testing, not for real users. To send
real emails, add:

| Variable | Value |
|---|---|
| `SMTP_HOST` | your provider's SMTP hostname |
| `SMTP_PORT` | usually `587` (the default if you don't set this) |
| `SMTP_USERNAME` | your SMTP username |
| `SMTP_PASSWORD` | your SMTP password (or app-specific password) |
| `SMTP_FROM` | e.g. `resumebank.biz <no-reply@yourdomain.com>` |

Any provider with an SMTP endpoint works — SendGrid, Mailgun, Postmark, AWS SES, or even a Gmail
account with an [App Password](https://myaccount.google.com/apppasswords) (`smtp.gmail.com`, port
`587`, your Gmail address as both the username and the "from" address) if you just want something
that works quickly for a small project.

**Confirmed in production: Railway blocks outbound SMTP (ports 25/465/587/2525) on the Free/Trial/
Hobby plans** to prevent spam abuse — it's only unblocked on **Pro** ($20/month) and above. If
you're on Hobby, SMTP will silently hang and time out no matter how correct your credentials are.
Either upgrade to Pro, or use an email provider with an HTTPS API instead of SMTP (e.g.
[Resend](https://resend.com), free tier 3,000 emails/month) — that needs a small code change
(`internal/mailer` would need a second `Mailer` implementation), not just new environment variables.

## Step 4 — Add a persistent volume for uploaded files

Without this, uploaded PDF resumes and other files would be lost every time you redeploy (Railway's
container filesystem doesn't persist by default).

1. Open the Command Palette (⌘K / Ctrl+K) and search for **"volume"**, or right-click your app
   service on the project canvas and look for the volume option.
2. Attach a new volume to your app service.
3. In the volume's settings, set its **mount path** to `/app/data` — the same path you set
   `UPLOAD_DIR` to in Step 3.

A single volume gives you 0.5GB on the trial plan or 5GB on Hobby, which is plenty for a while (each
resume/file upload is capped at 10MB by default — see `MAX_UPLOAD_MB` if you want to change that).

## Step 5 — Deploy and verify

1. Trigger a new deploy (Railway usually does this automatically after you finish changing
   variables; if not, use the **"Deploy"** button on your app service).
2. Watch the deploy logs. You should see a line like `listening on :8080 (env=production)` with no
   errors above it. If `AUTO_MIGRATE=true` is set, you'll also see migrations apply on first boot.
3. Click your app service's generated URL (Settings → Networking → the `*.up.railway.app` domain,
   or click **"Generate Domain"** if you don't have one yet).
4. Visit the site. Sign up as a candidate and as an employer, post a job, upload a file, and search
   — this exercises the database, file storage, and search all at once. See `README.md`'s "Manual
   verification walkthrough" for the full checklist.
5. Create your admin account. Install the [Railway CLI](https://docs.railway.com/guides/cli) (`npm
   i -g @railway/cli`, or see that page for other install methods), run `railway login`, then from
   this repo's directory:

   ```
   railway link                          # pick this project and your app service when prompted
   railway ssh
   ```

   `railway ssh` opens a shell *inside your already-deployed container* (not your local machine),
   where the `createadmin` binary from the Dockerfile already exists. Once connected, run:

   ```
   /app/createadmin -email=you@example.com -password=some-long-password
   ```

   then `exit` the SSH session. Log in at `https://<your-domain>/admin/login`. Run the same command
   again any time (via `railway ssh`) to reset that admin's password. (Don't confuse this with
   `railway run <command>`, which runs a command on *your own machine* with Railway's environment
   variables injected — useful for other things, but `/app/createadmin` only exists inside the
   deployed container, not on your laptop.)

If something's wrong, check "Troubleshooting" below before anything else.

## Step 6 (optional) — Enable candidate/job matching with Ollama

Skip this section entirely if you're fine with the matching pages showing "temporarily unavailable."
Nothing else in the app depends on this. `nomic-embed-text` is a small model (137M parameters), so
this doesn't need much CPU/RAM — it should run fine on Railway's default resources.

1. In your project, click **"+ New"** → **"Empty Service"**, then in that service's Settings set its
   **source** to a Docker Image: `ollama/ollama`.
2. Rename the service to `ollama` (Settings → General → service name). This matters because
   Railway's internal networking addresses services as `<service-name>.railway.internal`, and you'll
   reference this exact name from your app.
3. The Ollama image doesn't automatically download any model on its own — you need to tell it to
   pull `nomic-embed-text` when the container starts. In this service's Settings → Deploy, set a
   **Custom Start Command**:

   ```
   sh -c "ollama serve & until ollama list >/dev/null 2>&1; do sleep 1; done; ollama pull nomic-embed-text; wait"
   ```

   This starts the Ollama server, waits until it's responding, pulls the embedding model, and then
   keeps running. The first deploy will take a few minutes while the model downloads; watch this
   service's logs to see progress.
4. Add a volume to this service (same process as Step 4) mounted at **`/root/.ollama`** — that's
   where Ollama stores downloaded models, so without this you'd re-download the model on every
   redeploy.
5. **Don't** generate a public domain for this service — it only needs to be reachable from your app
   over Railway's private network, not from the internet.
6. Back on your **app** service's Variables tab, add:

   | Variable | Value |
   |---|---|
   | `OLLAMA_URL` | `http://ollama.railway.internal:11434` |
   | `OLLAMA_EMBED_MODEL` | `nomic-embed-text` |

7. Redeploy your app service so it picks up the new variables. Once both services are up and the
   model has finished downloading, log in as an employer and try "Match Candidates" (or as a
   candidate, "Match Jobs") to confirm it returns results instead of "temporarily unavailable."

Anyone who signed up (or posted a job) before the Ollama service finished pulling the model isn't
lost — the app retries computing their embedding automatically every 5 minutes in the background, so
they'll show up in match results shortly after Ollama comes online, with no action needed.

## Step 7 (optional) — Custom domain

In your app service's Settings → Networking, click **"Custom Domain"** and follow Railway's
instructions to point your domain's DNS at Railway. Once it's active, update `BASE_URL` to your real
domain instead of the `RAILWAY_PUBLIC_DOMAIN` reference.

## Updating the app later

Railway redeploys automatically every time you push to your GitHub repo's default branch. There's
nothing extra to run — `AUTO_MIGRATE=true` handles any new database migrations for you.

## Troubleshooting

- **Deploy fails with a database connection error.** Double-check `DATABASE_URL` is a *reference* to
  your Postgres service (it should show as a linked variable, not a literal string you typed in).
- **`CREATE EXTENSION vector` fails / migrations fail on first deploy.** You likely added Railway's
  plain "Postgres" template instead of the "pgvector" template — see Step 2. You'd need to delete
  that database and re-add the correct template.
- **Login doesn't stick / you get logged out immediately.** Make sure `COOKIE_SECURE=true` is set —
  browsers refuse "secure" cookies over plain HTTP, but they will work fine over Railway's HTTPS
  domain once this is set correctly.
- **Uploaded files disappear after a redeploy.** You're missing the volume from Step 4, or
  `UPLOAD_DIR` doesn't match the mount path you set.
- **"Forgot password" says it sent an email, but nothing arrives.** If you haven't set the `SMTP_*`
  variables (see Step 3), this is expected — check your deploy logs for the reset link instead. If
  you *have* set them, check your deploy logs for the actual SMTP error (the user themselves never
  sees this — it fails silently on their end, by design, so a broken email address doesn't leak
  which emails have accounts) and look for one of these two confirmed-in-production gotchas:
  - **The send hangs and eventually fails with a timeout/context-deadline error.** Railway blocks
    outbound SMTP (ports 25, 465, 587, 2525) on the **Free/Trial/Hobby plans** to prevent spam abuse
    — it's only unblocked on **Pro and above**. Either upgrade to Pro, or switch to an email
    provider with an HTTPS API instead of SMTP (e.g. [Resend](https://resend.com), free tier 3,000
    emails/month) — HTTPS on port 443 is never blocked, but that needs a small code change
    (`internal/mailer` would need a second `Mailer` implementation calling that provider's API
    instead of `net/smtp`).
  - **The send fails immediately with something like `555 5.5.2 Syntax error, cannot decode
    response`.** This means `SMTP_FROM` is set to a "Display Name &lt;addr&gt;" form (e.g. `resumebank.biz
    <no-reply@resumebank.biz>`) — which is correct for the email's own `From:` header, but some
    providers (Gmail included) also require the *separate* SMTP envelope sender to be a bare
    address, and older code here didn't make that distinction. If you're running a version from
    before this was fixed, update to the latest `master`.
- **The site works but matching always says "temporarily unavailable."** Either you skipped Step 6
  on purpose (fine), or your `OLLAMA_URL` doesn't match your Ollama service's internal address, or
  the `nomic-embed-text` model hasn't finished downloading yet — check the Ollama service's logs.
- **The Ollama service keeps restarting or the model never finishes downloading.** The startup
  command in Step 6 is a widely-used community pattern, not an officially documented Ollama feature
  — if a future Ollama image version changes its CLI behavior, that command may need adjusting.
  Check the service's logs for the actual error; `ollama serve` failing to start is usually a
  resource (RAM) limit, while `ollama pull` failing is usually a network issue.
- **Build fails immediately with a Dockerfile-related error.** Make sure you didn't accidentally
  delete or rename the `Dockerfile` at the repo root — Railway needs it to build this app.
