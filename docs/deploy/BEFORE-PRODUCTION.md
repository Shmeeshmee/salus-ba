# Before production

Everything that has to be filled in, decided or fixed before Salus takes its first real patient, in one place: the
values left for the owner, the placeholders, every setting and what refuses to start without it, and the dependency
audit. Written 2026-11-25 by the operations drill (docs/clinic-cms/WAVE7.md, RD4) and updated 2026-11-26 when the
production wiring it found missing was fixed; what it says refuses or does not refuse was read from `internal/config`,
`cmd/*/main.go` and `deployments/production/docker-compose.prod.yml`, not assumed. The legal and owner questions of the clinic system are
in docs/clinic-cms/PROGRESS.md "Before production"; the server's own setup is docs/deploy/README.md.

## 1. What is left for the owner: values

The four deploy blockers the drill found on 2026-11-25 (the notifier not given `SALUS_OPERATOR_EMAIL` and
`SALUS_ENV`; the `SELLER_*`, analytics, retention, push, SMS, fiscal and chat-summary settings reaching no container;
the landing image built without `VITE_APP_URL`; no mobile release build) were fixed on 2026-11-26 (readiness wiring):
the production compose file passes every setting each binary reads, `SALUS_ENV` is fixed to `production` there for
every Go service (an env file cannot make production run as dev), and `deployments/production/wiring_test.go` (run by
`make test` and CI's `make cover`) fails when a variable internal/config reads for a binary is not passed to its
service, or a compose variable is not named in `.env.production.example`. A placeholder that would otherwise ship now
stops the start-up or the build (§2). What remains is filling in values; nothing in the repository holds them.

**The `SALUS_ENV_FILE` secret** (the `production` environment; `deployments/production/.env.production.example`
names every key). New since the drill, and refused while empty or still in brackets:

| Key | What | Refused by |
| --- | --- | --- |
| `SALUS_OPERATOR_EMAIL` | who answers the data requests to the Salus account (their deadline warnings) | the deploy (empty or `[…]`) and the notifier |
| `SELLER_NAME`, `SELLER_TAX_ID`, `SELLER_ADDRESS`, `SELLER_ACCOUNT` | the owner's company on the membership invoices: legal name, JIB/IDB, address on one line, IBAN (clinic-cms Q3) | the deploy (empty or `[…]`) and the api and the notifier |

Optional, with the decided default when left out: `SALUS_OPERATOR_LANGUAGE` (bs), `ANALYTICS_PLATFORM_ENABLED`
(false) and `ANALYTICS_MIN_COUNT` (10) until the legal answers L1 to L8, `RETENTION_CLINICAL_DATA` (keep) and
`RETENTION_OPERATIONAL_LOG_DAYS` (90) until lawyer L1 to L3, `SMS_ENGINE`, `FISCAL_ENGINE` and `CHAT_SUMMARY_ENGINE`
(fake: the only value each takes today), and the push keys `FCM_CREDENTIALS_FILE`, `APNS_KEY_FILE`, `APNS_KEY_ID`,
`APNS_TEAM_ID`, `APNS_BUNDLE_ID`, `APNS_SANDBOX` (off; the two key files also need their compose secrets, which land
with the app's token registration: docs/deploy/push.md). The keys that existed before (domain, passwords, LiveKit,
Garage, SMTP, `PATIENT_APP_URL`) are unchanged: docs/deploy/README.md.

**Repository variables** (GitHub, Settings, Variables; none is secret, each ends up in a published page or app):

| Variable | Used by | Default | Unset |
| --- | --- | --- | --- |
| `SALUS_DOMAIN` | the deploy (already required), and the defaults below | none | the deploy refuses |
| `VITE_APP_URL` | the Images workflow: the landing image's build argument, the sign-up link | `https://app.<SALUS_DOMAIN>` | with no `SALUS_DOMAIN` either: a warning, and the page links to `[APP URL]/signup` |
| `MOBILE_API_BASE_URL` | the mobile workflow's release job: `--dart-define=API_BASE_URL` | `https://api.<SALUS_DOMAIN>` | the release build refuses |
| `MOBILE_PRIVACY_NOTICE_URL` | the same: `PRIVACY_NOTICE_URL`, the published privacy notice (https) | none | the release build refuses |
| `MOBILE_SUPPORT_EMAIL` | the same: `SUPPORT_EMAIL`, where a patient asks to reopen a closed account | none | the release build refuses |

The release job (`scripts/mobile-release.sh`) runs on a manual run of the mobile workflow or a `mobile-v*` tag. Its
APK is signed with the debug key, as `mobile/android/app/build.gradle.kts` is set up: for the owner's test phones, not
for Google Play. A Play release needs an upload keystore (created by the owner, kept in Bitwarden, given to CI as
secrets) and its signing config; an App Store build needs a macOS runner and Apple signing. Neither exists yet.

## 2. Placeholders

Outside `SALUS_ENV=dev` (production sets `production`) a placeholder of a server setting stops the binary at start, and
one of the patient app stops its release build and the release app's start; the landing page's are still printed
as-is (it has no start to stop: the Images workflow warns about `[APP URL]`, the others are hand edits).

| Placeholder | Where | Filled by | In production while unfilled |
| --- | --- | --- | --- |
| `[OPERATOR EMAIL]` | `SALUS_OPERATOR_EMAIL` (configs/.env.example, .env.production.example; any value in brackets counts as unset) | the env file | **The deploy and the notifier refuse** |
| `[SELLER]`, `[SELLER TAX ID]`, `[SELLER ADDRESS]`, `[SELLER ACCOUNT]` | `SELLER_NAME`, `SELLER_TAX_ID`, `SELLER_ADDRESS`, `SELLER_ACCOUNT` (internal/config/config_seller.go; .env.production.example) | the env file; the owner's company (clinic-cms Q3) | **The deploy, the api and the notifier refuse** (empty or any value in brackets); in dev they are printed on the invoice |
| `[PRICE]` | landing/src/content/pricing.ts (two plans); web/src/i18n/reminders/*.json `smsHint` (the SMS option's price, five languages) | a hand edit (and every language of the web catalog) | Shown as-is on the landing page's pricing and in the clinic's reminder settings |
| `[APP URL]` | landing/src/lib/appHref.ts, the sign-up link of landing/src/content/nav.ts | build argument `VITE_APP_URL` of the landing image: the repository variable `VITE_APP_URL`, else `https://app.<SALUS_DOMAIN>` | Only with neither variable: the link is `[APP URL]/signup`, and the Images workflow warns |
| `[YOUR EMAIL]` | landing/src/content/demo.ts (`mailto:[YOUR EMAIL]`) | a hand edit | "Book a demo" opens a broken mail link |
| `[Hosting]` / `[Where data is stored]`, `[Certification]` / `[Audit or standard]` | landing/src/content/standards.ts | a hand edit | Shown as-is on the landing page's standards tiles |
| `[SUPPORT EMAIL]` | mobile/lib/close_account_screen.dart | the repository variable `MOBILE_SUPPORT_EMAIL`, `--dart-define=SUPPORT_EMAIL` of the release build | **The release build refuses** (scripts/mobile-release.sh), and a release app built without it does not start (mobile/lib/release_settings.dart); debug builds show it |
| `https://salus.invalid/privacy` | mobile/lib/main.dart `PRIVACY_NOTICE_URL` | the repository variable `MOBILE_PRIVACY_NOTICE_URL` | The same: refused by the release build and the release app |
| `http://localhost:8080` | mobile/lib/main.dart `API_BASE_URL` | the repository variable `MOBILE_API_BASE_URL`, else `https://api.<SALUS_DOMAIN>` | The same |
| Plan prices (no bracket) | no plan is seeded | `salusctl subscription plan` (docs/plans/clinic-cms.md Q2) | No membership can be invoiced |

Not placeholders, though they look like one: `[DEPLOY_HOST]` in docs/deploy/README.md (known_hosts syntax) and the
usage arguments of the scripts (`[FILE]`, `[SERVICE]`, …).

## 3. Every setting, and what refuses without it

Read from `internal/config` (the only package that reads the environment) and the binaries' `main.go`. "Refuses"
means the binary exits at start with a message naming the variable (never its value). Production's values come from
`deployments/production/docker-compose.prod.yml`, the `SALUS_ENV_FILE` secret
(`deployments/production/.env.production.example`) and the key files on the encrypted volume (ADR 0011). Every
variable below that a binary reads is passed to its service (`deployments/production/wiring_test.go` fails
otherwise). `SALUS_ENV` is fixed to `production` by the compose file for every Go service, never read from the env
file; anything but `dev` counts as production.

### Every Go binary (`config.Load`: api, worker, notifier, scheduler, agent, migrate, salusctl)

| Setting | Default | Production | Unset or wrong |
| --- | --- | --- | --- |
| `DATABASE_URL` | none | compose (salus_app; migrate: the owner) | Every binary refuses |
| `REDIS_URL` | none | compose | api, worker and agent refuse (scheduler and notifier do not use Redis) |
| `HTTP_ADDR` | `:8080` | compose | Falls back to `:8080` (api only) |
| `FHIR_BASE_URL` | none | compose, `https://app.${SALUS_DOMAIN}/fhir/R4` | Every binary refuses a value not ending in `/fhir/R4`; the api refuses it empty or not absolute |
| `WEB_BASE_URL`, `PATIENT_APP_URL`, `APPOINTMENT_LINK_URL` | none | compose (`PATIENT_APP_URL` from the env file, `:?`) | The notifier refuses without any of the three |
| `TOTP_KEY` / `TOTP_KEY_FILE` | none | key file `/run/secrets/totp_key` | Invalid base64, both set, or an unreadable file: every binary refuses; unset: the api refuses (not 32 bytes) |
| `RATE_LIMIT_KEY` | none | env file (`:?`) | Invalid base64: every binary refuses; under 32 bytes: the api refuses |
| `TRUSTED_PROXY_CIDRS` | empty (no proxy trusted) | compose (the private ranges) | A non-CIDR entry is refused; empty is silent |
| `ANALYTICS_PLATFORM_ENABLED` | `false` | api, worker, scheduler, migrate (salusctl): env file, `:-false` | A value other than true, false or empty is refused |
| `ANALYTICS_MIN_COUNT` | 10 | the same, `:-10` | Below 10 or not a number is refused |
| `SALUS_ENV` | none (production) | compose: `production`, every Go service | `dev` only on the compose stack of deployments/docker-compose.yml |
| `APP_DATABASE_PASSWORD` | none | env file (`:?`) | migrate only; empty: migrate leaves salus_app's password alone |

### api, worker, agent

| Setting | Read by | Default | Unset or wrong |
| --- | --- | --- | --- |
| `LIVEKIT_URL`, `LIVEKIT_PUBLIC_URL`, `LIVEKIT_API_KEY`, `LIVEKIT_API_SECRET` | api, worker, agent | none | Refused; production: compose and the env file (`:?`) |
| `S3_ENDPOINT`, `S3_REGION`, `S3_BUCKET`, `S3_ACCESS_KEY_ID`, `S3_SECRET_ACCESS_KEY`, `BINARY_KEK_ID` | api, worker | none (compose: bucket `salus-binaries`) | Refused; an endpoint that is not an absolute http(s) URL too |
| `BINARY_KEK` / `BINARY_KEK_FILE` | api, worker | none | Refused when missing or not base64 of 32 bytes; production: the key file |
| `WEBHOOK_KEY` / `WEBHOOK_KEY_FILE` | api, worker | none (dev: a random key per process) | **Refused outside `SALUS_ENV=dev`**; production: the key file |
| `LLM_ENGINE`, `ANTHROPIC_API_KEY`, `LLM_MODEL` | api, worker, agent | `fake`, none, `claude-opus-5-5` | An engine other than fake or anthropic is refused; anthropic without a key is refused (stays fake until its DPA, ADR 0011 point 6) |
| `CHAT_SUMMARY_ENGINE` | api | `fake` | Anything else is refused; production `:-fake` |
| `SELLER_NAME`, `SELLER_TAX_ID`, `SELLER_ADDRESS`, `SELLER_ACCOUNT` | api (and the notifier) | dev: the `[SELLER…]` placeholders | **Refused outside `SALUS_ENV=dev`** when empty or in brackets; production `:?` |
| `FAKE_ENGINE_SCRIPT` | api, worker, agent | empty | Refused beside a real engine; set only by docker-compose.e2e.yml |
| `STT_ENGINE`, `AZURE_SPEECH_REGION`, `AZURE_SPEECH_KEY` | agent | `fake` | An engine other than fake or azure is refused; azure without both is refused |

### worker

| Setting | Default | Unset or wrong |
| --- | --- | --- |
| `RETENTION_DATABASE_URL`, `STATS_DATABASE_URL` | none | Refused (compose builds both from the env file's passwords) |
| `CLAMD_ADDR` | none | Refused when not host:port |
| `FISCAL_ENGINE` | `fake` | `eset` and anything else are refused (package G writes the adapter); production `:-fake` |
| `RETENTION_CLINICAL_DATA` | `keep` | Anything but keep is refused; production `:-keep` |
| `RETENTION_OPERATIONAL_LOG_DAYS` | 90 | Outside 1 to 3653 is refused; production `:-90` |
| `GOMEMLIMIT` | Go's | Read by the Go runtime; compose sets it from `SALUS_WORKER_GOMEMLIMIT` |

### notifier

| Setting | Default | Unset or wrong |
| --- | --- | --- |
| `SALUS_OPERATOR_EMAIL` | none (any value in brackets counts as none) | **Refused outside `SALUS_ENV=dev`**; production `:?` |
| `SALUS_OPERATOR_LANGUAGE` | `bs` | A value that is not one of the ten languages is refused; production `:-bs` |
| `SELLER_NAME`, `SELLER_TAX_ID`, `SELLER_ADDRESS`, `SELLER_ACCOUNT` | dev: the placeholders | As the api's |
| `SMTP_ADDR` | none | Refused when not host:port; production `:?` |
| `SMTP_TLS` | none in Go; production compose `starttls` | Anything but starttls, tls or none is refused; none with a login is refused |
| `SMTP_USERNAME`, `SMTP_PASSWORD` | none | Optional, but both or neither; production `:?` both |
| `SMTP_FROM` | none | Refused when not one RFC 5322 address; production `:?` |
| `SMS_ENGINE` | `fake` | `infobip` and anything else are refused until its DPA; production `:-fake` |
| `FCM_CREDENTIALS_FILE` | off | An unreadable or blank file is refused; production `:-` (the key file's compose secret lands with push, docs/deploy/push.md) |
| `APNS_KEY_FILE`, `APNS_KEY_ID`, `APNS_TEAM_ID`, `APNS_BUNDLE_ID`, `APNS_SANDBOX` | off | All four or none (a partial set is refused); `APNS_SANDBOX` without them, or not a boolean, is refused; production `:-` |

### migrate

`DATABASE_URL` (the owner), `APP_DATABASE_PASSWORD`, `RETENTION_DATABASE_PASSWORD`, `STATS_DATABASE_PASSWORD`:
each password is set on its role after the migrations, and an empty one leaves the role's password as it is. The
production compose file requires all three (`:?`).

### Production compose only

Compose itself refuses to start without (`${VAR:?}`): `SALUS_TAG`, `SALUS_DOMAIN`, `SALUS_ACME_EMAIL`,
`POSTGRES_PASSWORD`, `APP_DATABASE_PASSWORD`, `RETENTION_DATABASE_PASSWORD`, `STATS_DATABASE_PASSWORD`,
`RATE_LIMIT_KEY`, `LIVEKIT_API_KEY`, `LIVEKIT_API_SECRET`, `LIVEKIT_NODE_IP`, `S3_ACCESS_KEY_ID`,
`S3_SECRET_ACCESS_KEY`, `BINARY_KEK_ID`, `GARAGE_RPC_SECRET`, `SMTP_ADDR`, `SMTP_USERNAME`, `SMTP_PASSWORD`,
`SMTP_FROM`, `PATIENT_APP_URL`, `SALUS_OPERATOR_EMAIL`, `SELLER_NAME`, `SELLER_TAX_ID`, `SELLER_ADDRESS`,
`SELLER_ACCOUNT`. With a default: `GHCR_OWNER`, `SALUS_ACME_CA`, `S3_BUCKET`, `LLM_ENGINE`, `ANTHROPIC_API_KEY`,
`LLM_MODEL`, `CHAT_SUMMARY_ENGINE`, `SMTP_TLS`, `STT_ENGINE`, `AZURE_SPEECH_REGION`, `AZURE_SPEECH_KEY`,
`SALUS_OPERATOR_LANGUAGE`, `ANALYTICS_PLATFORM_ENABLED`, `ANALYTICS_MIN_COUNT`, `RETENTION_CLINICAL_DATA`,
`RETENTION_OPERATIONAL_LOG_DAYS`, `FISCAL_ENGINE`, `SMS_ENGINE`, `FCM_CREDENTIALS_FILE`, the `APNS_*` keys,
`SALUS_DATA_DIR`, `SALUS_WORKER_GOMEMLIMIT`, `SALUS_REDIS_MAXMEMORY` and the `SALUS_MEM_*` limits. The deploy
workflow refuses an env file without a value for each required key, or with a value still in brackets. The server's own
settings (`bootstrap.conf`: the deploy key, SSH, the firewall, upgrades, the backup's `BACKUP_*` and `ALERT_EMAIL`)
are documented in `deployments/production/bootstrap/bootstrap.conf.example` and docs/deploy/backup-restore.md.

### Builds

| Setting | Where | Default |
| --- | --- | --- |
| `VITE_APP_URL` | landing image build argument (Images workflow: the variable `VITE_APP_URL`, else `https://app.<SALUS_DOMAIN>`) | empty: `[APP URL]`, with a warning |
| `API_URL` | the web dev server's proxy only | `http://localhost:8080` |
| `API_BASE_URL`, `PRIVACY_NOTICE_URL`, `SUPPORT_EMAIL` | the patient app's `--dart-define`s; the release job takes them from `MOBILE_API_BASE_URL` (else `https://api.<SALUS_DOMAIN>`), `MOBILE_PRIVACY_NOTICE_URL`, `MOBILE_SUPPORT_EMAIL` | debug: localhost, `salus.invalid`, `[SUPPORT EMAIL]`; release: refused by scripts/mobile-release.sh and at the app's start |

### Tests and local stacks

`TEST_*` (the test helpers and the compose `tools` service), `E2E_*` (the end-to-end suite), `RESTORE_DRILL_STATE`
(the restore drill), `COMPOSE_PROJECT_NAME`, `SALUS_IMAGE_TAG` and the `SALUS_*_PORT` host ports: local only,
documented in configs/.env.example and deployments/docker-compose.yml.

## 4. Dependencies (audit of 2026-11-25)

`scripts/audit.sh` runs the whole audit in Docker with pinned scanners: govulncheck v1.9.0 over the module and
`tools/`, `npm audit` over web/, landing/ and test/e2e/web/, osv-scanner v2.6.0 over mobile/pubspec.lock, and Trivy
(the Images workflow's pinned image) over the four production images built from the checkout and the five
third-party images the production compose file pins. The CI gate stays the Images workflow (a CRITICAL fails a
digest); third-party images are not scanned in CI, only by this script.

**Fixed by RD4**:

| What | From → to | Finding | Why the update is safe |
| --- | --- | --- | --- |
| Go toolchain (all three Go images; `toolchain` in go.mod and tools/go.mod) | 1.27.3 → 1.27.4 | GO-2026-6859, an HTTP/1 server crash (MaxBytesReader header race); govulncheck found it reachable from `cmd/api`'s `http.Server` | A Go patch release: security and bug fixes only, no language or API change (Go's release policy); the whole `go` check set ran on it |
| postgres (production and the backup test) | 18.6 → 18.7 | 55 of the image's 129 HIGH (Debian packages rebuilt) | A PostgreSQL minor release: fixes only, same on-disk format, no dump or `pg_upgrade` (the next deploy restarts Postgres on the same data); the dev stack and every test already ran on 18.7 |
| clamav (production and dev) | 1.5.4 → 1.5.5 | libxml2 CVE-2026-86144 (HIGH, fixed in 2.13.9-r3) | A ClamAV patch release; clamd's protocol and the `CLAMD_CONF_*` settings are unchanged; pkg/clamd and internal/binary's scan tests ran against it |
| redis (production) | 8.10.3, rebuilt digest | one Debian package (zlib) fixed | The same Redis version, the official image rebuilt on a newer Debian base |

**Left, with their reach**:

| Where | Finding | Fix | Reach |
| --- | --- | --- | --- |
| salus-caddy (Caddy 2.11.9) | go-jose/v3 v3.0.5, CVE-2026-116587 (HIGH, denial of service parsing a JWE/JWS) | none upstream | Not reached: the Caddyfile uses no JWT or JOSE module; go-jose is in the binary through Caddy's ACME client, which parses only Let's Encrypt's answers |
| livekit-server v1.13.11 | golang.org/x/net v0.59.0, CVE-2026-78661 and CVE-2026-78669 (HIGH, x/net/html parsing) | x/net 0.62.0; no LiveKit release with it yet | Not reached: LiveKit parses no HTML from clients; move to the next LiveKit release that carries it |
| postgres 18.7 (Debian 13) | libxml2 CVE-2026-6653 (CRITICAL) and about 70 HIGH in Debian packages (util-linux, systemd, perl, gnupg, openssl, ncurses), most "affected" with no Debian fix, a few fixed in Debian and not yet in the image | Debian and the image | Low: Postgres is on the compose network only, no host port, and the app sends it no XML (no `xml` column, no `xmlparse`); the gnupg, perl and util-linux tools are not run by the server. Repin the digest when the image is rebuilt |
| postgres 18.7 (gosu) | Go stdlib of gosu 1.24.6: CVE-2025-68121 (CRITICAL, crypto/tls) and 24 HIGH | a gosu rebuild | Not reached: gosu only drops privileges at the entrypoint; it opens no connection |
| redis 8.10.3 (Debian 13) | 51 HIGH in Debian packages, none fixed upstream | Debian | Low: Redis is on the compose network only and holds temporary state; the vulnerable tools are not run by it |

govulncheck reported nothing else (one more vulnerability in an imported package that no code calls); npm audit
found 0 in each of the three lock files; osv-scanner found nothing in the mobile lock file; garage v2.5.0 and the
salus, salus-web and salus-landing images are clean of HIGH and CRITICAL (the nginx images take their Alpine fixes
from the runtime stage's `apk upgrade`, which the Images workflow rebuilds without the cache).

## 5. The drills

The backup and restore drill and the migration drill of 2026-11-25 are recorded in docs/deploy/restore-drills.md;
the production drill (the real Storage Box, before the first real patient data) is still to run, by
docs/deploy/backup-restore.md "Restore drill".
