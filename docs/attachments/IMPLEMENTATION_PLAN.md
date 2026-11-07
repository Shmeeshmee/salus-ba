# Questionnaire attachments implementation plan

Read `SPEC.md` first (approved 2026-11-07: the owner's settled decisions in §2, the open points with their
approved defaults in §3). Each package is one commit `attachments(<n>): <summary>`, made with `/run-package
docs/attachments <n>`. Packages run in waves (below): wave 0 and wave 2 alone on `feat/attachments`, the
packages of waves 1 and 3 at the same time, each in its own worktree and branch, merged back into
`feat/attachments` in the wave's integration step. One PR into `main` at the end.

## Rules for every package

- Test first: write the failing test, run it and record the failure in `PROGRESS.md`, then the least code to
  pass, then refactor. Service tests with hand-written fakes and `clock.Fixed`; repository tests on
  `pgtest.NewDatabase` (and `NewAppDatabase` for every RLS check); the object store and clamd tested against the
  compose services, never faked in their own package's tests; handlers with `httptest`, FHIR handlers against
  golden files.
- Build to `SPEC.md` §2 and to the defaults of §3. Do not reopen a settled decision; if the code and the plan
  disagree, stop and report.
- `binary` is added to `docs/architecture/archcheck/config.yaml` (context `clinical`), CLAUDE.md's `clinical`
  line, `ownership.yaml` (with `personal_data: true`) and the retention rules in package 0.1, with the table. The
  archcheck allowlist never grows. `questionnaireresponse` may import `binary`; `binary` imports no domain but
  `auditevent` (SPEC §2 "Dependency direction").
- Every migration is a new file, with its grants, `ENABLE ROW LEVEL SECURITY` and policies. Policies read
  settings as `NULLIF(current_setting('…', true), '')`.
- Every upload, status read, download (every range request), deletion, scan result and discard audits in the
  same transaction, refusals included; a test per service method asserts the event.
- No personal data in a log line, error, panic or job argument: never a title, file bytes, a content type or a
  signature name next to a patient id. A test asserts the log output of each new handler and job holds none of
  them (a synthetic title is asserted absent). Fixtures are generated in the tests (images, PDF, MP4); the EICAR
  string is assembled in the test, never stored as a file.
- A change to `api/openapi.yaml` is followed by `make gen-clients` in the same commit; after 0.1 nobody but
  0.1 changes `api/openapi.yaml` (a package that needs a change stops and reports).
- Dependencies: Go only the S3 client of ADR 0010 (added in 0.1); web none; mobile only what 1.7 justifies in its
  `PROGRESS.md` section (SPEC §2 "Mobile"). Pin every version exact; record each in `PROGRESS.md`.
- In a wave, a package edits only the files it owns (below) and its own section of `PROGRESS.md`, under the
  heading 0.1 wrote for it; the status table at the top of `PROGRESS.md` is updated in the integration step.
- Timing tests wait for a condition, never on a longer sleep; a flaky one is fixed, not slowed or skipped.

## Shared stack

`deployments/docker-compose.yml` has a fixed project name (`salus`), so every worktree shares one stack: one
Postgres, one Redis, one object store, one clamd. The stack is started once from the main checkout (the
repository's own working tree, on `feat/attachments` after 0.1) with `make up`, before the wave starts, and
stopped after its integration step. Inside a wave:

- a package never runs `make up` or `make down`, and never `make test`, `make cover` or `make e2e`: `redistest`
  flushes the Redis database it is given and the full runs would collide;
- a package runs only its own tests, `docker compose -f deployments/docker-compose.yml run --rm tools go test
  -count=1 ./<its packages>/...`, plus `make lint` and `make archcheck` (neither touches the stack); the web
  and mobile packages run their own `make web-*` or `make mobile-*` targets, which use no shared state;
- object store tests use a key prefix unique to the test (`t.Name()` and a random suffix) and delete what they
  wrote; nothing in a wave deletes the bucket or another test's keys.

The full `make lint`, `make test`, `make archcheck`, `make cover` and `make e2e` run in each wave's integration
step, on `feat/attachments`, with nothing else running against the stack.

## Check sets

Every check runs through `make` or `docker compose … run --rm tools`, in Docker. Never `go`, `npm` or `flutter`
on the host.

- **Go (in a wave)**: `make lint`, `make archcheck`, and the package's own tests as above, with `-cover`: each
  package it owns at 80% or more.
- **Go (alone: 0.1, 2.1, 4.1, and every integration step)**: `make lint`, `make test`, `make archcheck`, `make
  cover` (every package at 80% or more).
- **FHIR**: `make fhir-validate`.
- **Clients**: `make gen-clients` leaves no diff.
- **Web**: `make web-lint`, `make web-typecheck`, `make web-test`, `make web-build`.
- **Mobile**: `make mobile-analyze`, `make mobile-test`.
- **Stack** (alone only): the stack from the main checkout, the package's manual check against it.
- **E2E** (integration steps from wave 2 on, and 4.1): `make e2e`, twice in a row in 4.1.

## Waves

| Wave | Packages | Why their files don't overlap |
| --- | --- | --- |
| 0 | 0.1 | Alone: it writes every shared file (compose, config, migration, ownership, archcheck config, CLAUDE.md, `go.mod`, `api/openapi.yaml` and the generated clients, the `PROGRESS.md` headings), so nothing in later waves needs to |
| 1 | 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7 | Each owns one directory or one file set: `pkg/envelope`, `pkg/filetype`, `pkg/clamd`, `internal/binary/repository*.go` (the object store port and its S3 implementation only), `internal/questionnaire`, `web/`, `mobile/`. None imports another's new code: the Go packages are leaves, 1.4 uses only `go.mod`'s S3 client from 0.1, and the apps code against the clients 0.1 generated and the JSON shapes in SPEC §2 |
| 2 | 2.1 | Alone: it joins 1.1 to 1.4 in `internal/binary`, implements `binary.Holder` in `internal/questionnaireresponse`, and wires `internal/api/app.go`, the router and `cmd/api/main.go` |
| 3 | 3.1, 3.2 | 3.1 is the only package in `internal/binary` (its new service files and the repository and query additions they need), `internal/worker`, `cmd/worker/main.go` and `internal/retention/rules.go`; 3.2 is the only one in `internal/questionnaireresponse` and its `testdata`, and leaves `service_attachments.go` as 2.1 wrote it. They meet only at `binary.DiscardArgs` and the `Holder` methods, both written in 2.1 |
| 4 | 4.1 | Alone: end-to-end tests, `docs/privacy/`, the acceptance walk |

### Running a wave

1. From the main checkout on `feat/attachments`, with the previous wave integrated: `make up`.
2. Per package: `git worktree add ../salus-attachments-<n> -b attachments/<n> feat/attachments`, then in that
   worktree `/run-package docs/attachments <n>`. The runner commits once on `attachments/<n>` and does not
   push.
3. Integration step, in the main checkout, once every package of the wave is committed: `git merge --no-ff
   attachments/<n>` for each, in package order; the status table in `PROGRESS.md` updated; then the full Go,
   FHIR, Clients, Web and Mobile check sets, and from wave 2 on `make e2e`; in wave 1 also 1.6's builder walk
   against the stack (1.5 merged), recorded under 1.6. A fix the merge needs is one commit
   `attachments(wave <w>): integration` with what it fixed in `PROGRESS.md`. Then `git worktree remove` each
   worktree and delete its branch, and `make down`.

`scripts/run-feature.sh docs/attachments` also works as a fallback, one package after another on
`feat/attachments`; it skips packages whose commit is already on the branch.

## Packages

### 0.1 Baseline

Reads: CLAUDE.md; SPEC §2 and §5; `deployments/docker-compose.yml`, `configs/.env.example`, `internal/config`,
`migrations/00026_questionnaire_response.sql` and its test, `ownership.yaml`, `config.yaml`, `internal/retention/rules.go`,
`api/openapi.yaml` (the questionnaire and response paths), `docs/adr/`; the `database-migrations` skill.

Owns: `docs/adr/0010-object-store-and-file-scanning.md`; `deployments/docker-compose.yml` and the object store's
config file in `deployments/`; `configs/.env.example`; `internal/config/config.go` and its test;
`migrations/00027_binary.sql` and `migrations/binary_test.go`; `ownership.yaml`; `config.yaml`;
`internal/retention/rules.go` and its test; `CLAUDE.md`; `go.mod`, `go.sum`; `api/openapi.yaml`, the generated
clients in `web/` and `mobile/packages/salus_api`; `docs/attachments/PROGRESS.md` (the headings).

Depends on: nothing.

Builds:
- ADR 0010: the local S3-compatible store (Garage and SeaweedFS at least: licence, image size, maintenance,
  what `PutObject` with a known length and unsigned payload, ranged `GetObject`, `DeleteObject` and
  `ListObjectsV2` need, how the bucket and key are created), the Go S3 client (`aws-sdk-go-v2/service/s3`,
  `minio-go`, hand-written SigV4, against CLAUDE.md's dependency rule: its transitive modules listed), and
  ClamAV for scanning (clamd over TCP, `StreamMaxLength`, signature updates, memory). The choice is checked by
  running the four operations against the chosen store from the `tools` container (the AWS SDK's checksum
  defaults set to `WhenRequired`, as some S3-compatible stores reject the new trailing checksums), and the
  result recorded in the ADR. SPEC §3 O10 is the default.
- Compose: the store (pinned) and its one-shot init (bucket `salus-binaries`, the synthetic local key), clamd
  (`clamav/clamav`, pinned, the 250 MB stream limits, a signature volume, a `PING` health check); the api and
  worker get `S3_*` and `BINARY_KEK*`, the worker `CLAMD_ADDR`; `tools` gets `TEST_S3_*` and `TEST_CLAMD_ADDR` and
  depends on both. `configs/.env.example` documents every new variable.
- `internal/config`: `LoadObjectStore()` and `LoadClamd()` (SPEC §2 "Configuration"), and the API base derived
  from `FHIR_BASE_URL`; table tests for each variable missing or malformed, the error naming the variable only.
- Migration `00027_binary.sql`: the `binary` table, its checks (`content_type` one of four, `status` one of three,
  `size > 0`, `title` at most 200 characters, `sha1` 20 bytes), the index on (`questionnaire_response_id`,
  `link_id`), the foreign key with `ON DELETE CASCADE`, RLS and its policies (patient, clinic through the
  response, job scope), the grants (SPEC §2 "Tables"); `questionnaire_response.attachments jsonb NOT NULL DEFAULT
  '[]'`, its `UPDATE` grant and the job-scope policy on `questionnaire_response`.
- `ownership.yaml`: `binary: {owner: binary, personal_data: true}`. `config.yaml`: `clinical: [binary,
  questionnaire, questionnaireresponse]`. `rules.go`: `binary`, `ThenKeep`, with the basis of SPEC §5 (3.1
  changes it to `delete`). CLAUDE.md: `binary` in the `clinical` line; `pkg/` gains `envelope`, `filetype`,
  `clamd`; the compose services and the new test variables under "Checks".
- `go.mod`: the S3 client, pinned (`// indirect` until 1.4 imports it).
- `api/openapi.yaml`: the four binary operations, the `Binary` schema, the error codes; the questionnaire item's
  `attachment` type with `accept` and `maxFiles`; the submit's 409 `attachments_pending`. `make gen-clients`.
- `PROGRESS.md`: a heading per package, each with the line "Status: not started", so wave packages write under
  their own.

Tests: as `salus_app`, the patient scope sees and deletes only its own `binary` rows and inserts only for
itself; the clinic scope sees a file only while it can see the response (after a revocation, a past completed
response's file yes, a future one's no); the job scope sees one response's files; an unset scope sees nothing;
deleting a response deletes its files' rows, also as `salus_retention`; the job scope reads and updates only
`attachments` of its one response; the column grants refuse every other update. Config table tests.

Done when: checks Go (alone), Clients, Stack (`make up` starts the store and clamd healthy; the bucket exists;
clamd answers `PONG`), and the ADR is written with the check's result.

### 1.1 pkg/envelope

Reads: SPEC §2 "Encryption", §3 O4; `crypto/cipher` docs.

Owns: `pkg/envelope/`.

Depends on: 0.1.

Builds: `NewDataKey`; `KEK` (`Wrap`, `Unwrap`, AES-256-GCM, the binary id as additional data, its id); a
`Writer` that seals 64 KiB segments as it is written, and `CiphertextSize(n)`; `Open` for a whole file and
`OpenRange(dek, id, size, from, to)` that returns the ciphertext byte range to fetch and a reader that decrypts
those segments and yields exactly the requested plaintext. Standard library only.

Tests: round trips at 0, 1, 65,535, 65,536, 65,537 bytes and several MB; every range at segment boundaries and
inside one segment; a flipped bit, a swapped segment, a segment from another file (another id), a truncated file
and an extended file each fail to open; a wrong KEK or id fails to unwrap; `CiphertextSize` matches what the
writer wrote. `BenchmarkSeal`, `BenchmarkOpen`, `BenchmarkOpenRange` (1 MiB from a 200 MB file), MB/s in
`PROGRESS.md`.

Done when: checks Go (in a wave).

### 1.2 pkg/filetype

Reads: SPEC §2 "Upload pipeline", §3 O1 and O2.

Owns: `pkg/filetype/`.

Depends on: 0.1.

Builds: `Detect(head []byte) (Type, error)` for PDF (`%PDF-` within the first 1,024 bytes), JPEG, PNG and MP4
(an `ftyp` box with an ISO base media brand); an MP4 `Checker` (an `io.Writer` the pipeline tees into) that walks
the boxes as they stream, wherever `moov` is, buffers only `moov` (refused past 16 MB), and at `Close` accepts
only `avc1` video and `mp4a` audio sample entries; a `LimitReader` that fails with `ErrTooLarge` past `n` bytes and
with `ErrShort` when fewer than declared arrive. The size limits themselves are product rules and stay in
`internal/binary`.

Tests: each type detected from generated bytes; a renamed type (a PNG called `.pdf`, text called `.jpg`), HEIC,
MOV (`qt  ` brand), WebM and an empty input are refused; MP4 with `moov` first and last, with HEVC (`hvc1`), with
Opus audio, without audio; a box size that lies; the limit reader at, below and past the limit. `FuzzDetect` and
`FuzzChecker`: never a panic, memory bounded (run for 60 s in the package, the time recorded).

Done when: checks Go (in a wave).

### 1.3 pkg/clamd

Reads: SPEC §2 "Scan", §3 O5; clamd's protocol (`PING`, `VERSION`, `INSTREAM`).

Owns: `pkg/clamd/`.

Depends on: 0.1 (the compose clamd).

Builds: a client for clamd over TCP: `Ping`, `Version`, `Scan(ctx, io.Reader) (Result, error)` with
`INSTREAM` in length-prefixed chunks, the dial and every read and write bound to the context (`noctx`), a
`Result` of clean or found with the signature name, and distinct errors for the stream limit and for clamd
being unreachable. Standard library only.

Tests (against the compose clamd, `TEST_CLAMD_ADDR`): `PING`; the EICAR string, assembled in the test, is found;
synthetic clean bytes are clean; a stream past clamd's limit returns the limit error; a cancelled context stops
the scan; an unreachable address returns the unreachable error.

Done when: checks Go (in a wave).

### 1.4 internal/binary: the object store

Reads: SPEC §2 "Upload pipeline", "Download"; ADR 0010; the `go-domain` skill.

Owns: `internal/binary/repository.go` (the `Store` port only), `internal/binary/repository_s3.go`,
`internal/binary/repository_s3_test.go`; `go.mod`, `go.sum` only to drop `// indirect`.

Depends on: 0.1.

Builds: the `Store` port (`Put(ctx, key, r, size)`, `GetRange(ctx, key, from, to)`, `Delete(ctx, key)`,
`ListOlderThan(ctx, before, fn)` page by page) and `NewS3Store`, the one export of `repository_s3.go`, over the
ADR's client with an injected `*http.Client`; a key that does not exist is `ErrObjectNotFound`.

Tests (against the compose store, `TEST_S3_*`, keys prefixed per test): a put of a streamed body of known length
and a get of it; ranges at the start, the middle and the end; a missing key; delete twice; the listing across
more than one page (a small page size in the test) and its time filter; a body shorter than its declared size
fails without leaving an object. A put of 200 MB from a generating reader keeps the heap flat (sampled, recorded).

Done when: checks Go (in a wave).

### 1.5 internal/questionnaire: attachment items

Reads: SPEC §2 "Item model"; `internal/questionnaire` (model, `validate.go`, the FHIR handler and its golden files);
questionnaire SPEC §2 "Item model".

Owns: `internal/questionnaire/` and its `testdata/`.

Depends on: 0.1 (the OpenAPI item schema).

Builds: `ItemAttachment` accepted; `Item.Accept` (`photo`, `document`, `video`) and `Item.MaxFiles` (1 to 10),
in `items` JSON as `accept` and `maxFiles`; validation of both, and the refusal of an `attachment` item as the
question of a condition; the JSON API decodes them; the FHIR handler writes `type: attachment`, `repeats`, the
`mimeType` extensions and `questionnaire-maxOccurs`.

Tests: validation table (no kind, an unknown kind, `maxFiles` 0, 1, 10, 11, `accept` or `maxFiles` on another
type, a condition on an attachment item); a revise keeps an attachment item's `linkId` and refuses it as another
type; golden files: a version with an attachment item of each combination of kinds and of 1 and 10 files;
`make fhir-validate` passes on them.

Done when: checks Go (in a wave), FHIR.

### 1.6 Web: builder options and the file viewer

Reads: SPEC §2 "Item model", "Download", "FHIR", "Web"; `COPY.md` (approved); CLAUDE.md "UI direction";
`web/src/Builder.tsx`, `builder.ts`, `AppointmentDetail.tsx`, `messages.ts`.

Owns: `web/` (no change to the generated client).

Depends on: 0.1; `COPY.md` approved (2026-11-07).

Also (owner, 2026-11-07, folded in): the React duplicate-key warning in `web/src/Availability.tsx` fixed, with a
test that fails on the warning; the spacing under "Edit as new version" and "Retire" (questionnaire W23) in the
builder made the same as under the other action rows.

Builds: "File upload" in the palette (W1); the opened card's accepted kinds and number of files with the hint and
the errors (W2 to W6), the closed card's summary; in the submitted answers, each attachment item's file list
(W7, W8), "Open" and "Download" (W9); a viewer overlay for photos and PDFs with previous and next (W10, W11; the
PDF in the browser's own viewer), and a `<video controls preload="metadata">` on the content URL, so the browser
seeks with `Range`; W12 and W13. The URL is built from the binary id through the client's base, with the session
cookie, never from `valueAttachment.url`. Focus moves into the viewer and back; escape closes it; reduced motion
respected.

Tests (Vitest): the builder's new fields, their limits and errors; the payload sent with `accept` and
`maxFiles`; the file list from a fixture `QuestionnaireResponse` with each kind; the viewer's keyboard, focus and
previous and next; the video element's `src`; a 404 shows W13.

Done when: checks Web; the walk against the stack is in the wave 1 integration step (the builder with 1.5
merged) and 4.1 (the viewer with real files), at desktop width and 390 px, keyboard only once.

### 1.7 Mobile: pickers, conversion and upload

Reads: SPEC §2 "Mobile", §3 O1, O3, O9, O12; `COPY.md` (approved); `mobile/lib/questionnaire_screen.dart`,
`questionnaire.dart`, `messages.dart`; `mobile/ios/Runner/Info.plist`, `mobile/android/app/src/main/AndroidManifest.xml`.

Owns: `mobile/` (no change to the generated client).

Depends on: 0.1; `COPY.md` approved (2026-11-07).

Also (owner, 2026-11-07, folded in): the metadata of photos and videos stripped before upload, orientation kept
(SPEC §2 "Image and video metadata"), tested on generated files carrying GPS, device and orientation tags;
`compileSdk = 37` in `mobile/android/app/build.gradle.kts`; the questionnaire screen's saved state (questionnaire
M6) in the footer, with a background and a top border, always visible, and the test that expects it elsewhere
changed to the footer; the slot list no longer offers a slot that starts within 5 minutes of now (a test with
the clock at 4 min 59 s and 5 min 0 s before a slot).

Builds: the file upload item (M1, M2), the source sheet (M3) with only the accepted sources; camera and library
through `image_picker`, documents through `file_picker` (each justified in `PROGRESS.md`: what the platform
needs, why a channel of our own would be larger, licence, runtime dependencies, exact versions); HEIC/HEIF to JPEG
and video to H.264/AAC MP4 on the device (`AVAssetExportSession` on iOS, Media3 Transformer on Android only when
the source is not already H.264/AAC MP4), M9 on failure; the upload as a streamed request with
`Content-Length` and `Content-Disposition` through the client's base and auth, with progress (M4, M14), one file
at a time, "Try again" and "Remove" (M5, M6); polling the status until `clean` or `rejected` (M7); the server's
errors (M8), `attachments_pending` (M10), leaving during an upload (M11), the read-only list (M12, M13); the
permission text: M15 and M16 replace the two existing iOS strings, M17 added, M18 to M20 in the app.

Tests (widget, `MockClient`): the sheet shows only accepted sources; the hint per combination; progress and the
status sequence; a failed upload retried sends the whole file again; each error code's message; Submit disabled
while a file is pending and M10 on a 409; the leave prompt during an upload; the read-only list. The conversion
is behind one small Dart interface with a fake in the tests; the platform code is checked by hand on a device
(`README.md`), recorded in `PROGRESS.md`.

Done when: checks Mobile; the manual device check recorded (or named as not done, with the reason).

### 2.1 internal/binary: service, Postgres, handlers and wiring

Reads: SPEC §2 entire; the `go-domain` skill; `internal/questionnaireresponse` (service, `service_answers.go`,
`repository_postgres.go`, the access rule's use); `internal/api/app.go`, `internal/api/router`; `cmd/api/main.go`;
1.1 to 1.4's code.

Owns: `internal/binary/` (`model.go`, `errors.go`, `service.go`, `service_upload.go`, `service_download.go`,
`repository.go` gains the Postgres port, `repository_postgres.go`, `query.sql`, `binarydb/`, `handler_http.go`,
their tests); `sqlc.yaml`; `internal/questionnaireresponse/service_attachments.go`, its test, and the repository
and query additions it needs; `internal/api/app.go`, `internal/api/router`; `cmd/api/main.go`; `Makefile`
(`bench-binary-http`).

Depends on: wave 1 integrated (it uses 1.1, 1.2, 1.4 and 1.5; 1.3 is used by 3.1).

Builds: the model (`Binary`, `Status`, `Kind`, the limits of §3 O2, `Caller`, `Target`, `Grant`, `Attachment`,
`DiscardArgs` and `ScanArgs` with their kinds); `Holder` and its implementation in `questionnaireresponse`
(SPEC §2 "`binary.Holder`"); `Upload` (the pipeline in SPEC §2, step by step, with the identical-retry rule),
`Status`, `Download` (the transaction, then the ranged stream), `Delete`; the job-scope setter; the handlers for
the four routes, their deadlines (`http.ResponseController`) and the exemption from the body limit and request
timeout; the download headers of SPEC §2; `app.go` builds `questionnaireresponse.Service`, then `binary.Service`
with it as the holder, the `Store`, the KEK and the River client; the router mounts the routes.
`bench-binary-http`: a 200 MB synthetic MP4 uploaded through the handler with the real Postgres and store
(throughput, peak heap by `runtime/metrics`), then 1,000 random 1 MiB ranges (p50, p95, p99), in `PROGRESS.md`.

Tests: service with fakes for every refusal (each holder refusal, 411, 413 by header and by kind, 415 by sniff,
409 `too_many_files` and `response_full` including two concurrent uploads that would together pass the limit,
real database), the retried upload returning the first file, a failure after the put deleting the object, the
audit event of each call and refusal; `Download` refused while `quarantined` (409 `scanning`), for a `rejected`
file, for a practitioner on an unsubmitted response, after a revocation for a future appointment, allowed under
D9; ranges (first byte, last byte, a suffix, past the end 416, several ranges 200) against the real store;
`Delete` on a submitted response (409); the holder methods in `questionnaireresponse` with their RLS scopes on the
real database; handlers through HTTP for every status code and header, the title from `Content-Disposition`, the
log holding no title. The server timeouts: an upload slower than 15 s completes.

Done when: checks Go (alone), Clients (no diff), Stack: a file uploaded with `curl` from the `tools` container,
its status read, its download refused until a hand-set `clean` in the dev database (recorded and reverted).

### 3.1 Scan, discard and purge

Reads: SPEC §2 "Scan", "Discard", §5 "Retention"; `internal/worker` (`worker.go`, `scheduler.go`,
`retention.go`, `questionnaire.go` and their tests); `internal/retention`; 2.1's `internal/binary`.

Owns: `internal/binary/service_scan.go`, `service_discard.go`, `service_retention.go` and their tests, and the
additions they need in `repository.go`, `repository_postgres.go`, `query.sql` and `binarydb/`; `internal/worker/binary.go`, `worker.go`, `scheduler.go`,
`retention.go` and their tests; `cmd/worker/main.go`; `internal/retention/rules.go` and its test.

Depends on: 2.1 (and 1.3, integrated in wave 1).

Builds: `Scan` (SPEC §2 "Scan") and `Discard` (SPEC §2 "Discard") in `binary`; the purger (rejected rows past
24 h, then every object older than 24 h with no row; it counts both); the `binary_scan` and `binary_discard`
workers, a `scan` queue with 2 workers, the worker's config (`LoadObjectStore`, `LoadClamd`); the purger
registered in `newRetention` and the rule changed to `delete`, 24 h; the scheduler test lists both kinds.

Tests: a clean file becomes `clean` and its attachment no longer `pending`; a file with the EICAR string inside a
generated PDF (the fixture this package finds clamd flags, recorded) becomes `rejected`, is detached, its key
cleared and its object gone, audited with the signature; clamd stopped (an unreachable address in the test) leaves
the file `quarantined` and the job retried; a repeated scan changes nothing; `Discard` after a withdrawal deletes
every file of the response and audits each, and repeated does nothing; the purger keeps a fresh rejected row and
deletes an old one, deletes an old object without a row and keeps a young one and one with a row; the retention
service refuses to build without the purger; the scheduler test fails without the two kinds; the job logs hold
no title or signature next to a patient id. The 200 MB scan time and clamd's memory, recorded.

Done when: checks Go (in a wave).

### 3.2 internal/questionnaireresponse: attachment answers

Reads: SPEC §2 "Where an attachment answer lives", "Submit", "FHIR"; `internal/questionnaireresponse`
(`service_answers.go`, `enabled.go`, `validate.go`, `service.go` Withdraw, `handler_fhir.go`, its golden files);
2.1's `service_attachments.go`.

Owns: `internal/questionnaireresponse/` (except `service_attachments.go`, unchanged) and its `testdata/`.

Depends on: 2.1.

Builds: `PUT /answers` refuses an answer to an attachment item (422 with its `linkId`); `Submit` requires a file
on a required, enabled attachment item, refuses while any file is `pending` (409 `attachments_pending`), and detaches
the files of hidden items and enqueues `binary_discard`; `Withdraw` clears `attachments` and enqueues
`binary_discard`; a `Jobs` port and its River implementation for that one kind; the FHIR handler renders
`valueAttachment` (SPEC §2 "FHIR") under the item, for the patient with pending files, for the practitioner only
completed.

Tests: each refusal and the 409 through HTTP; a required attachment item hidden by a condition submits without a
file and its files are detached and the job enqueued in the transaction (a rollback leaves none); withdraw enqueues
the job with the two ids only; a cancellation's discard and the retention purger remove the files' rows (real
database, through the foreign key); golden files: an in-progress response with a pending and a clean file, a
completed one with a photo, a PDF and a video; `make fhir-validate` passes on them.

Done when: checks Go (in a wave), FHIR.

### 4.1 End-to-end and privacy

Reads: SPEC entire; `test/e2e/questionnaire_test.go`; `docs/privacy/`.

Owns: `test/e2e/attachments_test.go`, `test/e2e/web/attachments.spec.ts`; `docs/privacy/dpia.md`,
`processors.md`; the mobile CI workflow in `.github/workflows/`.

Also (owner, 2026-11-07, folded in): a CI job that builds the debug APK (`flutter build apk --debug` in the
pinned Flutter image) on every change under `mobile/`.

Depends on: 3.1, 3.2.

Builds: `test/e2e/attachments_test.go` (build tag `e2e`, synthetic data fresh per run): a practitioner publishes a
questionnaire with a photo-and-document item and a video item and sets it as their role's default; a patient
books; the patient uploads a generated JPEG and a generated 50 MB MP4 through the api; both become `clean`
(waited for); an EICAR PDF is `rejected` and detached (waited for); the patient submits; the practitioner reads the
`QuestionnaireResponse` over FHIR with both `valueAttachment`s, downloads the photo whole and the video by three
`Range` requests (start, middle, end) whose bytes match the original; the audit holds an event per request. Second
journey: the patient uploads a file to a second appointment's questionnaire and cancels; the file's row is gone at
once and its download is 404 (waited for). A Playwright spec opens the submitted answers, the photo in the viewer,
and plays and seeks the video (its `currentTime` moves and a `206` was served). The DPIA section and the
processors lines of SPEC §5. The acceptance walk: the builder, the patient app on a device or emulator (manual,
recorded), the practitioner's viewer and player, at desktop width and 390 px.

Done when: every check set, `make e2e` twice in a row, the walk recorded in `PROGRESS.md`.

## Definition of done

Every package committed, every wave integrated, `PROGRESS.md` complete with the failing tests and the readings of
SPEC §2 "Performance", all check sets pass locally and in CI on the PR into `main`.
