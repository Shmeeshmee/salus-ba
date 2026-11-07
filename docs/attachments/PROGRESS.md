# Questionnaire attachments progress

| Package | Status | Commit |
| --- | --- | --- |
| 0.1 Baseline | done, 2026-11-07 | `attachments(0.1)` |
| 1.1 pkg/envelope | not started | |
| 1.2 pkg/filetype | not started | |
| 1.3 pkg/clamd | not started | |
| 1.4 internal/binary: the object store | not started | |
| 1.5 internal/questionnaire: attachment items | not started | |
| 1.6 Web: builder options and the file viewer | not started | |
| 1.7 Mobile: pickers, conversion and upload | not started | |
| 2.1 internal/binary: service, Postgres, handlers and wiring | not started | |
| 3.1 Scan, discard and purge | not started | |
| 3.2 internal/questionnaireresponse: attachment answers | not started | |
| 4.1 End-to-end and privacy | not started | |

In a wave, packages write only under their own heading below (package 0.1 adds one per package); this table is
updated in the wave's integration step (`IMPLEMENTATION_PLAN.md` "Running a wave").

## Decisions

- 2026-11-07, owner (recorded in `SPEC.md` §2): photos, documents and short videos attached to questionnaire
  answers in the mobile app and viewed in the web app, the reserved `attachment` type and `valueAttachment`
  enabled; bytes in an S3-compatible object store (a pinned compose container locally, the hosting provider's EU
  store in production, not chosen); no MinIO image, ADR 0010 picks the local store (SeaweedFS and Garage compared
  at least) and the Go S3 client; a new domain `binary` (FHIR `Binary`) in `clinical`, owning the store, the
  `binary` table and the files' lifecycle.
- 2026-11-07, plan: the open points O1 to O12 (`SPEC.md` §3) are built to their defaults until the owner
  answers them. O8 (compute the R4 SHA-1 `hash`), O9 (submit refused while a file is pending), O10 (Garage and
  `aws-sdk-go-v2`, checked in 0.1), O11 (keep the file name as `title`) and O12 (no viewer of one's own files on
  mobile) were added by the plan to the owner's list.
- 2026-11-07, plan: `questionnaireresponse` → `binary`, not the reverse; `binary.Holder` is implemented by
  `questionnaireresponse`, as `retention.Purger` is by the domains, and `questionnaireresponse` reaches files only
  through the `binary_discard` job (`SPEC.md` §2 "Dependency direction"). Attachment answers live in their own
  column, outside `version_id`, so an upload never makes the patient's next save stale.
- 2026-11-07, plan: `COPY.md` is a draft. `RUN.tsv` holds 1.6 until the owner approves it; 1.7 needs the same
  approval (its permission text replaces the two iOS strings of the consultation feature) and does not start
  before it either.
- 2026-11-07, owner: the plan's changes to the brief (the `Holder` port, the separate `attachments` column,
  `DELETE /binaries/{id}`, the `keep` → `delete` retention rule, the route deadlines) approved; O1 to O12 approved
  as defaulted; `COPY.md` approved as written, the hold on 1.6 removed and 1.7 cleared to run. EXIF: the app
  strips location and all other metadata from photos and videos before upload and keeps the orientation, tested
  in 1.7 (SPEC §2 "Image and video metadata", O13). Production placement of ClamAV stays in SPEC §8. Folded in:
  1.6 the duplicate-key warning in `Availability.tsx` and the spacing under "Edit as new version" and "Retire";
  1.7 `compileSdk 37`, the questionnaire "Saved" state in the footer (background, top border, always visible) and
  no slot offered that starts within 5 minutes; 4.1 a CI job building the debug APK. The run goes on unattended:
  the session decides, records and continues (below).

## Decided while the owner was away

Each decision the session took during the unattended run, with its reason; the ones that touch the SPEC are also
rows of SPEC §3 marked "decided by the session, owner to review".

- 2026-11-07, session: the main checkout holds the owner's uncommitted `CLAUDE.md` edit, which would block
  merging package 0.1 (it edits `CLAUDE.md`). The main checkout was switched back to `main` (the edit travels with
  it, untouched; it was made on `main`), and every integration step runs in a worktree of `feat/attachments` at
  `../salus-attachments`, which also starts the shared stack. Wave packages run `docker compose … run --rm
  --no-deps tools …`, so that a run from one worktree never recreates a shared service whose bind-mounted config
  path differs between worktrees.
- 2026-11-07, session (0.1): Garage needs no init service. `garage server --single-node --default-bucket` creates
  the one-node layout, the bucket `salus-binaries` and the synthetic key pair from `GARAGE_DEFAULT_*` at start, and
  does nothing on a restart (checked). One service less, no shell in a scratch image to script. SPEC §3 O14.
- 2026-11-07, session (0.1): `binary` is a reserved word in Postgres (`CREATE TABLE binary` is a syntax error);
  the table keeps the SPEC's name, quoted `"binary"` in every statement. Renaming it would have reopened §2
  "Tables". SPEC §3 O15.
- 2026-11-07, session (0.1): no grant to `salus_retention` on `binary` yet. The existing test
  `TestRetention_RoleIsGrantedSelectAndDeleteOnExactlyTheTablesWhoseRuleIsDelete` requires the role's grants to be
  exactly the tables whose rule is `delete`, and the rule is `keep` until 3.1; the cascade from a purged
  `questionnaire_response` needs no privilege on `binary` (the referential action runs as the table's owner;
  tested). Least privilege until the purger exists. Package 3.1 adds the grant in a new migration
  (`00028_binary_retention.sql`) with its rule and purger. SPEC §3 O16.
- 2026-11-07, session (0.1): the JSON API names three codes the SPEC does not (`range_not_satisfiable` for a 416,
  `invalid_request` for a body shorter than `Content-Length`, `forbidden` for staff on upload and deletion), so
  that nobody after 0.1 has to change `api/openapi.yaml`. SPEC §3 O17.
- 2026-11-07, session (0.1): versions pinned no newer than about two weeks: Garage v2.4.1 (2026-09-08; v2.5.0 was
  14 days old), `clamav/clamav:1.5.4` (the current patch release), `aws-sdk-go-v2/service/s3` v1.114.5
  (2026-10-22; v1.114.9 was a day old).

## 0.1 Baseline

Status: done, 2026-11-07.

Built:
- ADR 0010 (`docs/adr/0010-object-store-and-file-scanning.md`): Garage over SeaweedFS for the local store,
  `aws-sdk-go-v2/service/s3` over minio-go and a hand-written SigV4 client (its transitive modules listed: AWS's
  own only), ClamAV's clamd for scanning; the four operations checked against the compose Garage and recorded.
- Compose: `objectstore` (`dxflrs/garage:v2.4.1`, `--single-node --default-bucket`, config inline through
  `configs:`, volume `objectstore`, health check `garage bucket info salus-binaries`) and `clamd`
  (`clamav/clamav:1.5.4`, the three 250 MB limits and `ConcurrentDatabaseReload no` through `CLAMD_CONF_*`, volume
  `clamav`, health check `clamdcheck.sh` = `PING`/`PONG`, start period 5 min). Neither publishes a host port and
  neither bind-mounts a repository file, so every worktree resolves the same two services. The api gets `S3_*`
  and `BINARY_KEK*` and waits for the store; the worker also gets `CLAMD_ADDR` and waits for clamd; `tools` gets
  `TEST_S3_*` and `TEST_CLAMD_ADDR` and waits for both. `configs/.env.example` documents every variable.
- `internal/config`: `LoadObjectStore()` (every variable required, `S3_ENDPOINT` an absolute http(s) URL,
  `BINARY_KEK` base64 of exactly 32 bytes), `LoadClamd()` (`CLAMD_ADDR` host:port), `Config.APIBaseURL()`
  (`FHIR_BASE_URL` without `/fhir/R4`), and `Load` refuses a `FHIR_BASE_URL` that does not end in `/fhir/R4`.
  Errors name the variable, never a value.
- `migrations/00027_binary.sql`: the `"binary"` table with the SPEC's columns and checks, two more consistency
  checks (`binary_scanned`: a scan time exactly when not quarantined; `binary_key`: a wrapped key exactly when not
  rejected), the index on (`questionnaire_response_id`, `link_id`), the foreign key `ON DELETE CASCADE`, RLS
  (patient: select, delete, insert for themselves on their own response with that response's clinic and
  appointment; clinic: select through an `EXISTS` on `questionnaire_response`; job scope: select, update, delete of
  one response's files), the grants; `questionnaire_response.attachments` (`jsonb`, `[]`, an array, empty once
  withdrawn), its `UPDATE` grant and the job scope's select and update policies.
- `ownership.yaml` (`binary`, owner `binary`, personal data), `config.yaml` (`clinical: [binary, …]`),
  `internal/retention/rules.go` (`binary`, `keep`, with the basis of SPEC §5), CLAUDE.md (`binary` in `clinical`,
  `pkg/` `envelope`, `filetype`, `clamd`, the two compose services, the test variables).
- `go.mod`: `github.com/aws/aws-sdk-go-v2/service/s3 v1.114.5 // indirect` and its modules (`aws-sdk-go-v2`
  v1.47.4, `aws/protocol/eventstream` v1.7.24, `internal/configsources` v1.5.7, `internal/endpoints/v2` v2.8.7,
  `internal/v4a` v1.5.7, `service/internal/accept-encoding` v1.13.22, `service/internal/checksum` v1.11.8,
  `service/internal/presigned-url` v1.14.7, `service/internal/s3shared` v1.20.7, `smithy-go` v1.28.6).
- `api/openapi.yaml`: `POST /questionnaire-responses/{id}/items/{linkId}/attachments`, `GET` and `DELETE
  /binaries/{id}`, `GET /binaries/{id}/content` (`Range`, 200/206/416, the headers), the `Binary` schema, the item's
  `attachment` type with `accept` and `maxFiles`, submit's 409 `attachments_pending`, the answers' 422 for an
  answer to an attachment item, and the codes `not_attachment_item`, `length_required`, `too_large`,
  `response_full`, `too_many_files`, `unsupported_type`, `scanning`, `range_not_satisfiable`,
  `attachments_pending`. `make gen-clients`: `web/src/api/schema.d.ts`, `mobile/packages/salus_api` (a new
  `AttachmentsApi` and `Binary`).

Failing first (each seen before the code existed):
- `internal/config`: the build failed (`undefined: config.LoadObjectStore`, `config.LoadClamd`, `APIBaseURL`);
  with empty stubs, all seven new tests failed for the right reason (no values read, no error, the variables read
  nowhere).
- `internal/retention`: `TestRules_KeepBinariesWithTheirResponseUntilThePurgerExists`: "no rule for binary".
- `migrations`: every `TestBinary_*` failed with `relation "binary" does not exist (42P01)`;
  `TestQuestionnaireResponse_TheJobScopeReadsAndUpdatesItsResponsesAttachments` with `column "attachments" … does
  not exist (42703)`, the job scope seeing no row, and a withdrawal with attachments accepted.

Tests (`migrations/binary_test.go`, as `salus_app` on `NewAppDatabase`, `salus_retention` on
`NewRetentionDatabase`): each scope sees only its files (patient A and B, clinic A, clinic B after the revocation
the past completed response's file only, the job scope its one response's file even where the clinic no longer
sees it, the job scope named with another clinic, a response alone, an account set: nothing); the patient inserts
only for themselves on their own response with its clinic and appointment, clinic and job scope never; updates by
the job scope only, deletes by the patient (their own) and the job scope (its response's); the column grants
refuse every other update and `TRUNCATE`; the checks; a response's deletion removes its files through the
discard scope and as `salus_retention`, which cannot read `binary`; `attachments` is an array, empty once
withdrawn, and updated by the job scope only for its one response of its clinic. Config: table tests for each
variable missing or malformed, and that only `internal/config` reads the new variables.

Checks:
- `make lint`: gofmt, `go vet`, golangci-lint over both modules: 0 issues. Its last step,
  `scripts/check-migrations.sh`, cannot run inside the container from a git worktree (the worktree's `.git` file
  points at the main checkout's git dir, which is not mounted); run in the `tools` container with that git dir
  mounted read-only: "ok (no migration from main changed)".
- `make test` (`-count=1`): pass. `make cover`: pass, every package at 80% or more (total 94.8%,
  `internal/config` 100%, `internal/retention` 100%).
- `make archcheck`: pass.
- `make gen-clients`: run twice, the second run leaves no diff. `make web-typecheck`, `make web-lint`, `make
  web-test` (385 tests), `make mobile-analyze`, `make mobile-test`: pass against the new clients.
- Stack: the stack brought up with the new compose file (`docker compose … up -d --build --wait`; `make up`'s
  `--remove-orphans` was left out, see "Dev stack"): every service healthy; migration 27 applied on the dev
  database (`"binary"` exists, empty; `questionnaire_response.attachments` exists); `garage bucket info
  salus-binaries` finds the bucket (0 objects); clamd answers `PING` with `PONG`; `/readyz` 200.

Readings (ADR 0010 has the rest):
- Garage: 12.8 MiB resident idle; healthy within 2 s; restart idempotent.
- clamd: healthy about 50 s after its first start with the image's bundled signatures; 975 MiB resident idle.
  The image's `CLAMD_CONF_*` handling appends a second line for each setting on a container restart (not on a
  recreate); clamd starts and answers with the duplicates (checked), so it is left as is.
- With the SDK's default checksum settings an unseekable `PutObject` over plain HTTP fails in the client;
  `RequestChecksumCalculation` and `ResponseChecksumValidation` at `WhenRequired` and `UNSIGNED-PAYLOAD` are what
  package 1.4 must set (ADR 0010).

Decided in this package (not touching the SPEC):
- On `"binary"` the clinic policy excludes the job scope (`app.questionnaire_response_id` set), so the job scope
  sees exactly one response's files. `questionnaire_response`'s existing clinic policies are unchanged, so in the
  job scope (which sets `app.organization_id`) a response table read also shows the clinic's own view beside the
  one response, as the notice scope already does (`migrations/questionnaire_response_test.go`); binary's jobs
  query by response id.
- The patient's insert policy checks that the copied clinic, patient and appointment are the response's (an
  `EXISTS` through the patient's own view of `questionnaire_response`), instead of a four-column foreign key that
  would need a new unique index on `questionnaire_response`.
- `questionnaire_response_withdrawn_attachments`: a withdrawn response holds no attachments, as it holds no
  answers; 3.2's `Withdraw` clears both in one statement.
- `Config.APIBaseURL()` is a method, so the `config.Config` literals in `internal/api`'s tests get it too.
- The config test that only the api and the worker call `LoadObjectStore` and only the worker `LoadClamd`
  accepts a subset until 2.1 and 3.1 add the calls.
- No user-facing text was added (`COPY.md` unchanged); the API's error descriptions are developer text.

For later packages:
- 1.4: build the client with `s3.New(s3.Options{BaseEndpoint, Region, Credentials: aws.CredentialsProviderFunc,
  UsePathStyle: true, HTTPClient, RequestChecksumCalculation: WhenRequired, ResponseChecksumValidation:
  WhenRequired})` and `s3.WithAPIOptions(v4.SwapComputePayloadSHA256ForUnsignedPayloadMiddleware)` on each put; no
  `config` or `credentials` module. A missing key is `*types.NoSuchKey`. Drop `// indirect` for `service/s3` and
  the core module.
- 1.7: the generated `AttachmentsApi.uploadAttachment` takes a `MultipartFile`; a streamed upload with progress
  may need a `StreamedRequest` on the same client base and auth instead.
- 2.1, 3.1: `"binary"` is quoted in `query.sql`. 3.1: the grant to `salus_retention` comes in a new migration (O16).
- Wave runs: the main checkout has a git-ignored `deployments/.env` with `SALUS_API_PORT`, `SALUS_MAILPIT_PORT`
  and `SALUS_LANDING_PORT`; a worktree without a copy resolves other ports, and any `docker compose run` without
  `--no-deps` (that includes `make lint` and `make archcheck`) then recreates mailpit (and fails: the default port
  8025 is held by another project) and livekit. Copy that file into each worktree. livekit still bind-mounts
  `deployments/livekit.yaml`, so a run with deps from another worktree recreates livekit (harmless, but it
  restarts); the object store and clamd bind-mount nothing.

Dev stack: migration 27 applied (the empty `"binary"` table and the `attachments` column, `[]` on every existing
response); the new services `objectstore` and `clamd` running and healthy with the volumes `salus_objectstore`
and `salus_clamav`; the bucket `salus-binaries` holds no object (the ADR check deleted its keys); the app services
recreated from this worktree's compose file. `deployments/.env` copied (git-ignored) from the main checkout.
salus-demo-* containers untouched.

## 1.1 pkg/envelope

Status: not started

## 1.2 pkg/filetype

Status: not started

## 1.3 pkg/clamd

Status: not started

## 1.4 internal/binary: the object store

Status: not started

## 1.5 internal/questionnaire: attachment items

Status: not started

## 1.6 Web: builder options and the file viewer

Status: not started

## 1.7 Mobile: pickers, conversion and upload

Status: not started

## 2.1 internal/binary: service, Postgres, handlers and wiring

Status: not started

## 3.1 Scan, discard and purge

Status: not started

## 3.2 internal/questionnaireresponse: attachment answers

Status: not started

## 4.1 End-to-end and privacy

Status: not started
