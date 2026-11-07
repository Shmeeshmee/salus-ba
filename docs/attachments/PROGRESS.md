# Questionnaire attachments progress

| Package | Status | Commit |
| --- | --- | --- |
| 0.1 Baseline | done, 2026-11-07 | `attachments(0.1)` |
| 1.1 pkg/envelope | done, 2026-11-07 (wave 1) | `attachments(1.1)` |
| 1.2 pkg/filetype | done, 2026-11-07 (wave 1) | `attachments(1.2)` |
| 1.3 pkg/clamd | done, 2026-11-07 (wave 1) | `attachments(1.3)` |
| 1.4 internal/binary: the object store | done, 2026-11-07 (wave 1) | `attachments(1.4)` |
| 1.5 internal/questionnaire: attachment items | done, 2026-11-07 (wave 1) | `attachments(1.5)` |
| 1.6 Web: builder options and the file viewer | done, 2026-11-07 (wave 1) | `attachments(1.6)` |
| 1.7 Mobile: pickers, conversion and upload | done, 2026-11-07 (wave 1) | `attachments(1.7)` |
| 2.1 internal/binary: service, Postgres, handlers and wiring | done, 2026-11-07 (wave 2) | `attachments(2.1)` |
| 3.1 Scan, discard and purge | done, 2026-11-07 (wave 3) | `attachments(3.1)` |
| 3.2 internal/questionnaireresponse: attachment answers | done, 2026-11-07 (wave 3) | `attachments(3.2)` |
| 4.1 End-to-end and privacy | done, 2026-11-07 (acceptance walk after the PR, `WALK.md`) | `attachments(4.1)` |

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

- 2026-11-07, wave 1 integration: the packages' own decisions are in their sections (1.1 to 1.7, "Decided while
  the owner was away"); the three that touch the SPEC are SPEC §3 O18 (which MP4 files, 1.2), O19 (no
  `questionnaire-maxOccurs` on a one-file item, 1.5) and O20 (the colour metadata kept, 1.7). Wave packages ran
  their own tests with `--no-deps` and the lint script with the main `.git` mounted read-only, because
  `scripts/check-migrations.sh` cannot read git from a worktree inside the container; 1.6 ran npm through `docker
  run` on the node image so the owner's `salus-demo-web` (the compose `node` service) was never touched; 1.7
  compiled the debug APK once in a throwaway container from the local `salus-demo-android` image, because the
  `flutter` service has no Android SDK.

- 2026-11-07, wave 1 integration: the Stop hook of this session runs `make arch-gate` from the main checkout (on
  `main`), and `make`'s `docker compose run` there recreated LiveKit with that checkout's `livekit.yaml` path while
  the integration checks ran from `../salus-attachments`, failing both. From then on every integration check runs
  the make targets' own commands with `--no-deps` (`checks.sh` in the session's scratchpad), so no check recreates
  a shared service; `make e2e`'s `up` runs without `--remove-orphans`, so the owner's `salus-demo-*` containers are
  never removed. The 1.6 builder walk planned for this step moves into the acceptance walk after the PR, which
  walks the builder in Chromium against the stack anyway; running it here would have meant a second Vite server
  beside the owner's demo.
- 2026-11-07, wave 1 integration checks (all `--no-deps`, on the merged branch): lint 0 issues, check-migrations
  ok; archcheck ok; `go test -count=1` both modules ok; cover ok (total 94.8%, every package at 80% or more); `make
  fhir-validate` ok; gen-clients no diff; web lint, typecheck, 432 tests and build ok; mobile analyze ok and 292
  tests ok.

- 2026-11-07, wave 2 integration (2.1 ran alone on `feat/attachments`): lint, archcheck, `go test -count=1`, cover
  (total 94.5%), fhir-validate, gen-clients, web and mobile checks and `make e2e` (its commands with `--no-deps`,
  `up` without `--remove-orphans`) all pass. SPEC §3 O21 to O24 are 2.1's.

- 2026-11-07, wave 3 integration (all checks and e2e pass): 3.1 and 3.2 merged without conflict. SPEC §3 O25 (the signature is logged,
  not audited) and O26 (`Discard`'s signature) are 3.1's; 3.2 needed no SPEC row.

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

Status: done, 2026-11-07.

Built (`pkg/envelope/`, standard library only):
- `envelope.go`: `SegmentSize` (64 KiB), `Overhead` (16), `KeySize` (32); `NewDataKey` (32 bytes from
  `crypto/rand`); `KEK` (`NewKEK(id, key)`, `ID`, `Wrap`, `Unwrap`: AES-256-GCM, the binary id as additional data,
  a random 96-bit nonce per wrap, stored as nonce followed by the sealed key, 60 bytes); `ErrOpen`, `ErrUnwrap`,
  `ErrRange`, `ErrKeySize`.
- `seal.go`: `Writer` (`NewWriter(dst, dek, id)`, `Write`, `Close`) holds one segment and seals it once the next
  byte shows it is not the last; `Close` seals the last one (an empty file is one empty segment) and does not close
  the destination. `CiphertextSize(n) = n + 16 × max(1, ceil(n / 65,536))`.
- `open.go`: `Open(src, dek, id)` streams a whole ciphertext, telling the last segment by what follows it;
  `OpenRange(dek, id, size, from, to)` returns a `Range` with `Start` and `End`, the inclusive ciphertext bytes to
  fetch (whole segments), and `Range.Reader(fetched)` decrypts them and yields exactly bytes `from` to `to`.
  Nonce: the segment index, big-endian, 96 bits; additional data: the id, the index (8 bytes) and the last-segment
  flag (1 byte). Every tampering is `ErrOpen`; a source's own error is passed on.

Failing first: the tests and benchmarks were written before any code; the build failed with `undefined:
envelope.SegmentSize`, `envelope.NewDataKey`, `envelope.NewWriter`, `envelope.Range`, `envelope.Open`,
`envelope.OpenRange`. Once green, the last-segment flag and the nonce index were taken out of the code on purpose:
`TestOpen_FailsOnTampering` failed ("truncated at a segment boundary", "full segments truncated": opened); restored.

Tests (`envelope_test.go`, `range_test.go`): round trips at 0, 1, 65,535, 65,536, 65,537, 131,072 bytes and
5 MiB + 12,345, written in uneven chunks, each checked against `CiphertextSize`; the ciphertext holds no plaintext;
every pair of boundary points (0, 1, 100, each segment edge ±1, the last two bytes) as a range on files of 1 byte,
one segment, two segments and 3 segments + 500, the bytes and the fetched ciphertext span checked; reads of 7 bytes;
ranges outside the file refused; a flipped bit, two swapped segments, a segment from another file (another id), a
truncation inside and at a boundary, an extension by a byte and by a segment, an empty and a sub-tag ciphertext,
another id and another data key all fail to open, whole and ranged; the last segment does not open as a middle one;
a wrong KEK, another id, a flipped bit and a truncated wrapped key fail to unwrap; two wraps differ; key sizes;
writes after `Close` and to a failing destination.

Checks:
- tests: `go test -count=1 -cover ./pkg/envelope/...`: pass, coverage 96.4%.
- lint (`scripts/lint.sh`, the main checkout's git dir mounted read-only): 0 issues, check-migrations ok.
- archcheck: pass.

Readings (`go test -run '^$' -bench . -benchmem -count=3`, tools container, i7-14700HX, one core per benchmark):
- `BenchmarkSeal` (16 MiB): 3,817 to 4,429 MB/s, 153 KB and 261 allocations per file.
- `BenchmarkOpen` (16 MiB): 4,651 to 4,786 MB/s, 157 KB per file.
- `BenchmarkOpenRange` (1 MiB from the middle of a 200 MB file): 3,642 to 3,899 MB/s, 0.27 to 0.29 ms per range,
  142 KB and 23 allocations. Memory per file is a segment buffer or two, whatever the file's size; encryption is
  far faster than any phone's upload.

### Decided while the owner was away

- 2026-11-07, session (1.1): `OpenRange` returns a `Range` value (the ciphertext span to fetch) whose `Reader`
  takes the fetched bytes, because the reader cannot exist before the ranged `GetObject` that feeds it; `from`
  and `to` are inclusive, like the HTTP `Range` header 2.1 parses. No SPEC row needed.
- 2026-11-07, session (1.1): an empty plaintext is one empty sealed segment (16 bytes), so even a file cut to
  nothing fails to open; a plaintext of an exact multiple of 64 KiB has no extra empty segment. The `binary` table
  allows no empty file anyway. No SPEC row needed.
- 2026-11-07, session (1.1): the wrapped key's nonce is random (a key-encryption key wraps many keys, so a counter
  would need state), stored in front of the sealed key in `wrapped_key` (60 bytes). The binary id is the additional
  data as the string the caller passes: 2.1 must always pass the canonical lowercase UUID. No SPEC row needed.

Copy added by the session, not yet reviewed: none.

Dev stack: untouched (only `docker compose run --rm --no-deps tools` containers, removed on exit).

## 1.2 pkg/filetype

Status: done, 2026-11-07.

Built (`pkg/filetype/`, standard library only, imports nothing of the repository):
- `Detect(head []byte) (Type, error)`: PNG and JPEG by their signatures at byte 0, MP4 by an `ftyp` box at byte
  0, at least 12 bytes long, whose major brand is `isom`, `iso2` to `iso6`, `mp41`, `mp42` or `avc1`, PDF by
  `%PDF-` wholly inside the first 1,024 bytes; anything else `ErrUnsupported`. `Type` is the content type
  (`application/pdf`, `image/jpeg`, `image/png`, `video/mp4`), so 2.1 stores it as is.
- `Checker` (`NewChecker`, `Write`, `Close`): an `io.Writer` fed the whole file from byte 0. It walks the top-level
  boxes as they arrive (32- and 64-bit sizes, size 0 only for a last box that is not `ftyp` or `moov`), requires
  `ftyp` first and exactly one `moov`, before or after `mdat`, buffers only `moov` (grown as its bytes arrive,
  never from the declared size; a declared size past 16 MiB, header included, is refused at the header), checks
  it as soon as it is complete and frees it: every `trak` → `mdia` → `minf` → `stbl` → `stsd` must hold exactly
  its entry count of sample entries, each `avc1` or `mp4a`, and at least one track `avc1`. Structure errors are
  `ErrMalformed`, codecs `ErrUnsupported`; the first error sticks (a `TeeReader` stops the upload there). `Close`
  refuses a file that ends inside a box (a size that lies) or without `moov`.
- `LimitReader(r, n)`: exactly `n` bytes; `ErrShort` when `r` ends early, `ErrTooLarge` when a byte follows the
  `n`th (that byte is never returned); other errors from `r` pass through.

Tests first: the four test files were written before any code and failed to build (`undefined: Type`, `Detect`,
`NewChecker`, `LimitReader`); with stubs returning zero values, `TestDetect_TheAcceptedTypesFromTheirBytes`,
`TestDetect_RefusesEverythingElse`, `TestDetect_ARenamedFileIsWhatItsBytesSay`, `TestLimitReader`,
`TestChecker_RefusesCodecs`, `TestChecker_RefusesMalformedFiles`, `TestChecker_MoovUpToSixteenMegabytes`,
`TestChecker_RefusesAnOversizedMoovFromItsHeader`, `TestChecker_StaysFailed` and `FuzzDetect` failed on their
assertions. Fixtures are generated in the tests (`image/png` and `image/jpeg` of a 4x4 picture, a minimal PDF, an
MP4 box skeleton with zeroed media); every MP4 case is streamed in writes of 1, 7, 4,096 bytes and all at once.

Checks:
- `go test -count=1 -cover ./pkg/filetype/...`: ok, coverage 97.8%.
- `FuzzDetect`, 60 s, `-parallel 4`: 10,750,156 executions, no failure. `FuzzChecker` (the file and a write size),
  60 s, `-parallel 4`: 2,814,126 executions, no failure; it asserts the buffered `moov` never exceeds four times
  the bytes written plus 64. No crasher, so no corpus file in `testdata/fuzz`.
- `scripts/lint.sh` (gofmt, `go vet`, golangci-lint, check-migrations): 0 issues, after fixing a fieldalignment
  in a test table and a tagged switch.
- archcheck: ok.

### Decided while the owner was away

- 2026-11-07, session (1.2): the MP4 major brands accepted are `isom`, `iso2` to `iso6`, `mp41`, `mp42` and `avc1`
  (what Android's MediaMuxer and iOS's MPEG-4 export write); the major brand decides, not the compatible list, so
  a HEIF or MOV that lists `isom` as compatible is still refused. `M4V `, `3gp*`, `dash` major brands are refused.
  Reason: the narrowest set that the converted uploads of SPEC §3 O1 use; the codec check stays the real gate.
  SPEC row needed (O1 reading).
- 2026-11-07, session (1.2): an MP4 needs at least one `avc1` track; an `mp4a`-only file is refused
  (`ErrUnsupported`), since audio is out of scope (SPEC §6). Any other sample entry in any track (`hvc1`, `hev1`,
  `Opus`, `encv`, a timed-metadata or `tmcd` track) refuses the file; the app's conversion and metadata stripping
  (O1, O13) leave none. SPEC row needed (O1 reading).
- 2026-11-07, session (1.2): the 16 MB `moov` limit is 16 MiB (16,777,216 bytes) of the whole box, header
  included; a larger one is `ErrMalformed`, which 2.1 can map like `ErrUnsupported` (415). Only the first `mdia`,
  `minf`, `stbl` and `stsd` of a track are read, as players read them.
- 2026-11-07, session (1.2): the `Checker` takes the file from its first byte (2.1 tees the 4 KiB head and the
  rest into it), and requires `ftyp` as the first box.

Deferred: none. No copy added (the package has no user-facing text).

## 1.3 pkg/clamd

Status: done, 2026-11-07.

Built:
- `pkg/clamd/clamd.go` (standard library only): `New(addr)`, `Ping`, `Version`, `Scan(ctx, io.Reader) (Result,
  error)`. Null-terminated commands (`zPING`, `zVERSION`, `zINSTREAM`); `INSTREAM` sends 64 KiB chunks, each
  with its 4-byte big-endian length, then the zero-length terminator. The reply is read alongside the writes, so
  clamd's early answer at its size limit is kept even though clamd then closes the connection and the next
  write fails. The dial uses `DialContext`; the connection takes the context's deadline and, through
  `context.AfterFunc`, a past deadline once the context is done, so every blocked read or write stops.
- `Result{Signature, Found}`; `ErrStreamLimit` ("INSTREAM size limit exceeded"), `ErrUnreachable` (the dial
  failed and the context is not done); a cancelled or expired context returns the context's error, never
  `ErrUnreachable`; an error from the caller's reader is returned wrapped, never blamed on clamd.

Failing first (each seen before the code existed):
- The build failed (`undefined: Result`, `ErrStreamLimit`, `parseScanReply`). With empty stubs every test failed
  for the right reason (Version empty, EICAR not found, no limit error, no cancellation error, no source error,
  no unreachable error for `Ping`, `Version` and `Scan`, the reply table wrong), except `Ping` against the real
  clamd, which a stub returning nil passes; the unreachable test covers `Ping`'s failing side.

Tests (`pkg/clamd/clamd_test.go` against the compose clamd at `TEST_CLAMD_ADDR`, failing without it, never
skipping; `reply_test.go` table over the reply parser, including replies the compose clamd cannot be made to
send): `PING`; `VERSION` starts with "ClamAV "; the EICAR string, assembled from three parts in the test, is
found (signature `Eicar-Test-Signature`); 390 kB of synthetic clean bytes (several chunks) and an empty stream
are clean; an endless stream of zeros returns `ErrStreamLimit`; a context cancelled during the scan stops it with
`context.Canceled`; an already cancelled context is `context.Canceled` and not `ErrUnreachable`; a failing reader's
error is returned; a closed local port returns `ErrUnreachable` from all three calls.

Checks:
- Own tests: `go test -count=1 -cover ./pkg/clamd/...` pass, coverage 87.5%; `-count=5 -race` pass (2.8 s).
- Lint (`scripts/lint.sh`, `--no-deps`, main git dir mounted read-only): 0 issues, check-migrations ok.
- Archcheck (`go -C tools run ./archcheck -root /src`): pass.

Readings:
- The limit test streams until clamd stops: 252.4 MiB sent (the 250 MiB `StreamMaxLength` plus what is in
  flight in the buffers), 0.25 s on the compose network; that is the smallest a past-the-limit test can be with
  the configured limit, and it does not hard-code the limit. clamd's resident memory after the runs: 953 MiB
  (975 MiB idle in 0.1); clamd was not restarted or reconfigured.
- Version against the compose clamd: `ClamAV 1.5.4/28176/Sat Nov  7 06:24:18 2026`.

Decided in this package (not touching the SPEC):
- No timeout of the client's own: the caller's context bounds every call (3.1's job timeout).
- One connection per call, no pool and no `IDSESSION`: the scan queue runs 2 jobs at a time (SPEC §2 "Scan").
- `ErrUnreachable` only for a failed dial; a connection lost mid-scan is a plain wrapped error. 3.1 retries
  every error alike, so a finer split buys nothing.
- An unexpected clamd reply is quoted in the error (clamd's own text, never the scanned bytes); the signature
  name is only in `Result`, never in an error, so a caller decides where it is logged.

No user-facing text was added.

## 1.4 internal/binary: the object store

Status: done, 2026-11-07.

Built:
- `internal/binary/repository.go`: the `Store` port (`Put(ctx, key, r, size)`, `GetRange(ctx, key, from, to)` with
  both offsets inclusive, `Delete(ctx, key)`, `ListOlderThan(ctx, before, fn)` calling `fn(keys)` once per page with
  that page's keys last modified before `before`, empty pages skipped, stopping at `fn`'s first error) and
  `ErrObjectNotFound`.
- `internal/binary/repository_s3.go`: `NewS3Store(hc, endpoint, region, bucket, accessKeyID, secretAccessKey,
  pageSize)`, its one export, over `aws-sdk-go-v2/service/s3` as ADR 0010 says: `s3.New(s3.Options{…})` with
  `BaseEndpoint`, `Region`, an `aws.CredentialsProviderFunc` over the key pair, `UsePathStyle`, the injected
  `*http.Client`, `RequestChecksumCalculation` and `ResponseChecksumValidation` at `WhenRequired`; each put with
  `v4.SwapComputePayloadSHA256ForUnsignedPayloadMiddleware` and one attempt only (the body is read once). No
  `config` or `credentials` module. `*types.NoSuchKey` → `ErrObjectNotFound`; any other S3 answer keeps only its
  error code, so no vendor error type leaves the file.
- `go.mod`: `// indirect` dropped for `aws-sdk-go-v2` v1.47.4, `service/s3` v1.114.5 and `smithy-go` v1.28.6 (the
  three modules the file imports); `go.sum` needed no change.

Failing first: `repository_s3_test.go` written against the port alone; the build failed with `undefined:
binary.NewS3Store`.

Tests (`repository_s3_test.go`, against the compose Garage through `TEST_S3_*`, every key under
`test/<t.Name()>-<random>/` and deleted by the test): a put of a body that cannot be seeked (3 × 65,552 + 123
random bytes) and ranged gets of the whole, the first segment, a middle segment, the end and one byte; a missing
key is `ErrObjectNotFound`; delete twice, then the key is gone; a body of 1,000 bytes declared as 2,000 fails and
leaves no object; the listing with a page size of 2 over five objects calls `fn` three times or more and finds the
five keys with `before` an hour ahead and none of them an hour back; `fn`'s error stops the listing after one
call; a 200 MB put from a generating reader keeps the heap flat and its last 4 bytes read back.

Checks:
- `go test -count=1 -cover ./internal/binary/...` (`--no-deps`, `GOFLAGS=-p=4`): pass, coverage 94.1%.
- `scripts/lint.sh` (gofmt, `go vet`, golangci-lint over both modules, `check-migrations.sh` with the main git
  dir mounted read-only): 0 issues, "ok (no migration from main changed)".
- archcheck (`go -C tools run ./archcheck -root /src`): pass (`binary` was already in `config.yaml` from 0.1).

Readings:
- The 200 MB put, the heap sampled (`runtime.ReadMemStats`, `HeapAlloc`) every 10 ms: 999 KiB before, peak 1,272 KiB
  during the put, growth 273 KiB; the put took about 0.46 s against the local Garage. The test fails above 16 MiB
  of growth.
- The bucket holds 0 objects after the run (`garage bucket info salus-binaries`).

### Decided while the owner was away

- 2026-11-07, session (1.4): `ErrObjectNotFound` is declared in `repository.go` beside the port, not in
  `errors.go`: in wave 1 this package owns only the repository files, and no `errors.go` exists yet. Package 2.1,
  which writes the domain's `errors.go`, may move it there. Not a SPEC change.
- 2026-11-07, session (1.4): the listing's page size is a parameter of `NewS3Store` (the composition root passes
  1000, S3's maximum), so the test can page over five objects instead of writing a thousand; the only other way
  would be an export or an internal test. Not a SPEC change.
- 2026-11-07, session (1.4): `ListOlderThan` hands `fn` the keys of one page at a time (not one key at a time), so
  the purger (3.1) can look up a page's rows in one query. Not a SPEC change.
- 2026-11-07, session (1.4): a put is never retried by the SDK (`RetryMaxAttempts` 1): the body is a stream read
  once; the app retries the whole upload (SPEC §2 "Retried uploads"). Not a SPEC change.

For later packages:
- 2.1: build the store with `binary.NewS3Store(hc, s.Endpoint, s.Region, s.Bucket, s.AccessKeyID,
  s.SecretAccessKey, 1000)` from `config.ObjectStore`; `hc` needs no overall `Timeout` shorter than the upload's
  30-minute deadline (the request context bounds each call). `GetRange` takes inclusive offsets, as a `Range`
  header does.
- Integration: `go mod tidy` also moves `github.com/pion/webrtc/v4` from `// indirect` to a direct requirement
  (something already imports it); that drift predates this package and was left out of its diff.

Dev stack: nothing left; every test key deleted (the bucket at 0 objects); no service restarted or reconfigured.

## 1.5 internal/questionnaire: attachment items

Status: done, 2026-11-07.

Built:
- `model.go`: `FileKind` (`photo`, `document`, `video`), `Item.Accept` and `Item.MaxFiles`, stored in the `items`
  JSON and sent by the JSON API as `accept` and `maxFiles` (the OpenAPI shape 0.1 wrote; nothing in the JSON
  handler changed, it decodes the model's own JSON).
- `validate.go`: `ItemAttachment` accepted with `accept` a non-empty set of the three kinds (no unknown kind, none
  twice) and `maxFiles` 1 to 10; `accept` or `maxFiles` on any other type refused; `repeats` and options stay
  choice only. An attachment item as the `question` of a condition was already refused (neither boolean nor
  choice) and is now tested. The kind a new version must keep is `attachment`, whatever its settings.
- `handler_fhir.go`: an attachment item is `type: attachment`, `repeats: true` when `maxFiles` is more than 1, a
  `mimeType` extension (`valueCode`) per content type of each accepted kind in the stored order (photo:
  `image/jpeg`, `image/png`; document: `application/pdf`; video: `video/mp4`), then `questionnaire-maxOccurs`
  (`valueInteger`) when `maxFiles` is more than 1. The item's extension type is generalised (url, then
  `valueCodeableConcept`, `valueCode`, `valueInteger`) so the help item control's JSON is byte for byte as before
  (the questionnaire response feature's golden files embed it).
- Golden file `testdata/questionnaire_attachments.json`: an item for each of the seven combinations of kinds, of 1
  and of 10 files (all three kinds both ways, in two orders), one required, with help text and under a condition,
  two in a section. The other golden files are unchanged.

Failing first (each seen before the code existed):
- Build failed: `undefined: questionnaire.FileKind`, `FilePhoto`, `FileDocument`, `FileVideo`.
- With the fields only: `TestCreate_ValidatesTheQuestionnaire` failed on every valid attachment case ("invalid
  questionnaire, want it stored"), on `accept`/`maxFiles` on a string, choice, section and instructions ("= <nil>,
  want an InvalidError"), and on the two conditions on an attachment (the attachment itself named as well);
  `TestRevise_KeepsAnAttachmentItemAndRefusesItsLinkIDAsAnotherType` failed (the revised draft's attachment
  refused on save); `TestHandlerRead_SendsTheQuestionnaire` failed on the missing golden file.
- `make fhir-validate` then failed on the new golden file, 8 errors: `questionnaire-maxOccurs` "is not allowed to
  be used at this point (based on context invariant 'repeats=true')" and "integer value is less than permitted
  minimum value of 2" on each one-file item; after the change below the golden file differed from the response
  (seen), was rewritten with `-update` and its diff reviewed (only the four one-file items lost the extension).

Tests:
- Validation table (`attachmentCases`, run by `TestCreate_ValidatesTheQuestionnaire`): each kind alone, every
  kind with 10 files, required with help text in a section under a condition: valid; no kind, an unknown kind, a
  kind twice, `maxFiles` 0, -1 and 11, `repeats` or options on an attachment, `accept` on a string or a section,
  `maxFiles` on a choice or instructions, a condition on an attachment item (boolean and string answers): refused,
  naming the item. The earlier "reserved attachment type" case is replaced (the SPEC enables the type) by "an
  attachment without its settings", still refused; `TestUpdateDraft_…` gains "a boolean become an attachment".
- `TestRevise_KeepsAnAttachmentItemAndRefusesItsLinkIDAsAnotherType`: a revise carries the item over with its
  `linkId`, `accept` and `maxFiles`; the draft may change its settings; making the `linkId` a string is refused
  and audited as a minor failure.
- `TestHTTPHandler_DecodesAndSendsAnAttachmentItemsSettings`: `accept` and `maxFiles` decoded into the service
  call and encoded in the response.
- `TestPostgres_ReadsSeeOnlyTheClinicSet`: its tree gains an attachment item and `equalItem` compares `Accept` and
  `MaxFiles`, so the round trip through `questionnaire.items` (create, update, revise) is checked.

Checks:
- `go test -count=1 -cover ./internal/questionnaire/...` (`--no-deps`, `GOFLAGS=-p=4`): pass, 95.1%.
- `make fhir-validate`: pass, 0 errors on every file (`questionnaire_attachments.json`: 0 errors, the usual
  dom-6 warning, information notes that the `mimeType` codes cannot be checked without a terminology server).
- Lint (`scripts/lint.sh`, `--no-deps`, the main checkout's git dir mounted read-only): 0 issues on both modules,
  check-migrations ok.
- Archcheck (`go -C tools run ./archcheck -root /src`, `--no-deps`): pass.

### Decided while the owner was away

- 2026-11-07, session (1.5): `questionnaire-maxOccurs` is written only when `maxFiles` is more than 1. The R4
  extensions package the validator loads (`hl7.fhir.uv.extensions` 5.4.0) allows it only on an item with
  `repeats = true` and with a value of 2 or more, so `make fhir-validate` refuses it on a one-file item; there,
  `repeats` absent already says at most one file. A FHIR reader takes `maxFiles` as `maxOccurs`, or 1 when the
  item does not repeat. The JSON API and the stored item always carry `maxFiles`. SPEC row needed (§2 "Item
  model" says the extension carries `maxFiles`, without the exception).
- 2026-11-07, session (1.5): an attachment item's kind, which a later version must keep for its `linkId`, is
  `attachment` whatever its `accept` and `maxFiles`: responses are tied to their version, so a new version may
  change which files and how many without misreading an earlier answer (unlike single and multiple choice, whose
  answers change shape). Not touching the SPEC.
- 2026-11-07, session (1.5): the `mimeType` extensions follow the stored order of `accept`; the service does not
  reorder or normalise the set. Not touching the SPEC.

For later packages:
- 2.1: `questionnaire.Item.Accept` (`[]questionnaire.FileKind`) and `Item.MaxFiles` (`int32`, 1 to 10 on every
  stored attachment item) are what `Holder.UploadTarget` returns as the item's accepted kinds and limit.
- Integration: `internal/api/app_questionnaire_test.go` names a step "the reserved attachment type"; it still
  passes (an attachment item without `accept` and `maxFiles` is refused 422 with its `linkId`), only its name is
  out of date. Outside this package, so not changed here.

Dev stack: nothing changed; only `docker compose run --rm --no-deps` containers (tools, fhir-validator), each
removed on exit, and `pgtest`'s throwaway databases.

## 1.6 Web: builder options and the file viewer

Status: done, 2026-11-07.

Built:
- Builder: "File upload" (W1) in the palette and in the type dropdown (`attachment` in `builder.ts`'s kinds). The
  opened card has "Accepted files" (W2, checkboxes Photos, Documents, Videos) with the W3 hint, and "Number of
  files" (W4, a number input 1 to 10, step 1) with its hint; the closed card and the published version's read-only
  list show W5 ("1 file", "Up to [n] files") after the type. W6 on the item: no kind ticked, or a number that is
  not a whole number from 1 to 10 (an emptied field included); the draft is then not offered for publishing, as
  with every other item problem. `Item` gains `accept` and `maxFiles`, set on file uploads only; a change of type
  to a file upload gives it the defaults, a change away drops them. Read from FHIR (`mimeType` extensions mapped to
  the kinds, `questionnaire-maxOccurs`) and from the JSON API (`accept`, `maxFiles`), saved as `accept` (in the
  order photo, document, video) and `maxFiles`. An attachment item cannot be a condition's question (as before:
  only yes or no and choice questions can).
- Appointment detail, answers of a submitted response: a file upload item's answer is W7 ("1 file", "[n] files")
  and the file list (W8: the title, then "[Kind], [size]"), each with "Open" and "Download" (W9). No file: "Not
  answered". `Files.tsx`: the viewer, a modal dialog over the page with "File [n] of [total]", "Previous file",
  "Next file" and "Close" (W10); a photo as an `<img>`, a PDF in an `<object type="application/pdf">` (the
  browser's own viewer, no PDF library), a video in `<video controls preload="metadata">` on the content URL, so the
  browser makes the `Range` requests and seeks; each labelled with W11. The focus moves to Close on open, Tab and
  Shift+Tab stay inside, escape closes, and the focus goes back to what opened it. The fade-in runs only under
  `prefers-reduced-motion: no-preference`. At 520 px and below the viewer takes the whole screen.
- Failures: when the image, the PDF or the video fails to load, the viewer asks `GET /binaries/{id}`: 404 (or
  `rejected`) is W13 "This file is no longer available."; `clean` means the file is there but the browser cannot
  show it, W12 with "Download"; anything else (an error, no answer, `quarantined`) is W13 "The file could not be
  loaded." with "Try again", which mounts the media again. W12 is also the `<object>`'s fallback content, which a
  browser without a PDF viewer shows. An answer about a file the viewer has stepped away from is dropped.
- The content URL is `binaryContentURL(id)` in `web/src/api/client.ts`: the client's base (the page's origin),
  `/binaries/<id>/content`, the id encoded; same origin, so the session cookie goes with it. Nothing is ever loaded
  from `valueAttachment.url` itself (below).
- `web/vite.config.ts`: the dev server proxies `/binaries` to the api.
- Folded in (owner): `Availability.tsx` keyed the hours editor and the default questionnaire, two siblings, both by
  the role id, so React warned about a duplicate key; they are now `hours-<role>` and `default-<role>`. The
  published version's "Edit as new version" and "Retire" row now has 16 px above and below
  (`.published-actions`, and `.items.readonly` 16 px above for a retired version, which has no row), as the
  editor's action row and the failure alert have.

Failing first (each seen before the code existed):
- `Availability.test.tsx` "renders the hours and the default questionnaire without a duplicate-key warning":
  received React's "Encountered two children with the same key, … a4000000-…-0000000000a1" from `console.error`.
- `Builder.test.tsx` "leaves the same space under "Edit as new version" and "Retire" …": `expected +0 to be 16`
  (`app.css` read into the page; jsdom computes the margins).
- `builder.test.ts`: the file upload defaults on `newItem` and `changeKind`, and the five W6 cases (no message:
  the messages and the check did not exist). `questionnaire.test.ts`: the three file upload cases (FHIR read,
  JSON read, `toRequest`: `accept` and `maxFiles` missing). `attachment.test.ts`: the module did not exist.
- `Builder.test.tsx` "a file upload item": all eight failed (no "Accepted files" group, no W5 tag, no W6, no
  `accept` in the saved body). `Files.test.tsx`: the module did not exist. `AppointmentQuestionnaires.test.tsx`
  "list the files of a file upload item …": "Unable to find an element with the text: 2 files".

Tests (Vitest, synthetic fixtures, placeholder titles): the builder's new fields, their limits (0, 11, 1.5, empty)
and W6, the closed card's W5 and the toggle's accessible name, the saved payload with `accept` and `maxFiles`, the
published version's summary; the FHIR and JSON reads; the file list from a fixture `QuestionnaireResponse` with a
photo, a video and a PDF, and an item with no file; W8's sizes (1 KB at the least, the switch to MB at 999.5 KB);
the id taken only from a `/binaries/<uuid>/content` path (another origin's path, a path with `..`, no url: no id,
no actions); the viewer's focus on open and back on Close and on escape, the Tab trap both ways, previous and next
with the ends marked, none for one file; the `<img>` `src`, the `<object>` `data` and type, the `<video>` `src`,
`controls` and `preload`; a 404 shows W13 without Try again, a 500 W13 with Try again that remounts the image, a
`quarantined` status W13, a `clean` status W12 with Download, and a late answer about a file left is ignored.

Checks (run with `docker run --rm` on the image `salus-node:24.22.0`, the worktree mounted at `/src`, `-w
/src/web -u 1000:1000 -v salus_npm:/cache/npm`, the same npm scripts as the make targets; not through `docker
compose run node`, so the owner's `salus-demo-web`, which runs as the compose `node` service, was never touched):
- `npm ci`: done (the worktree had no `node_modules`).
- web-lint (`npm run lint`, oxlint): 0 warnings, 0 errors.
- web-typecheck (`npm run typecheck`): pass.
- web-test (`npm test`): 24 files, 432 tests, pass. Of the first full run one test failed and its name was not
  kept; 19 full runs after it passed, so it was not found again (the machine ran six other wave packages at the
  time). Recorded for the integration step to watch.
- web-build (`npm run build`): pass; the existing chunk-size warning (934 kB, the LiveKit client) unchanged in kind.
- The browser walk is not in this package (wave 1 integration step for the builder with 1.5 merged; 4.1 for the
  viewer with real files).

Decided while the owner was away:
- 2026-11-07, session (1.6): the binary id is read from the path of `valueAttachment.url` (`/binaries/<uuid>/content`,
  nothing else accepted), and the URL the app loads is built from that id on its own origin. SPEC §2 "FHIR" gives
  `valueAttachment` no other field that carries the id (`contentType`, `size`, `title`, `hash`, `url`), and the
  plan says the URL is built from the id, never taken from `url`; taking only a UUID from it means a resource can
  never point the browser at another origin or path. Package 3.2 writes `url` in exactly that form.
- 2026-11-07, session (1.6): a new file upload item accepts photos and documents, 1 file. The fewest files and no
  videos (the largest files) unless the practitioner asks for them: data minimisation, and the practitioner ticks
  what the question needs.
- 2026-11-07, session (1.6): the viewer also opens videos (in the `<video>` element of the plan), so previous and
  next go through every file of the item, whatever its kind, and the practitioner has one way to open a file.
- 2026-11-07, session (1.6): the viewer does not read `GET /binaries/{id}` before showing a file; it asks only when
  the browser fails to load one, to tell W12 from the two W13. A completed response holds only clean files (SPEC
  §2 "Submit"), so the extra audited status read would almost always say what the content read says.
- 2026-11-07, session (1.6): sizes count 1 MB as 1,000,000 bytes and 1 KB as 1,000 (as the limits of W3 read), a
  file under 1 KB shows as "1 KB", and the switch to MB comes where whole kilobytes would reach 1000.
- 2026-11-07, session (1.6): a file with an empty title (the API allows one) is named by its kind ("Photo"), and
  its W11 label is the kind alone, so no row or label is blank.
- 2026-11-07, session (1.6): the Vitest config reads `app.css` as text (`test.css.include`), so a test can check a
  spacing against the real stylesheet in jsdom; nothing else in the tests changes.

Copy added by the session, not yet reviewed: none. Every string is from `COPY.md` word for word.

For later packages:
- 2.1: the viewer shows a PDF in an `<object>`; with `Content-Security-Policy: sandbox` on the content response,
  check in Chrome that its PDF viewer still renders inside an `<object>` (SPEC §2 "Download"), and record the
  header per type.
- 3.2: the web app needs `valueAttachment.url` to end in `/binaries/<binary id>/content`; anything else is listed
  without Open or Download.
- Wave 1 integration and 4.1: walk the builder's file upload card and the viewer at desktop width and 390 px,
  keyboard only once; check the `<video>` seeks (Range requests in the network panel) and the PDF viewer.

Dev stack: nothing changed. No compose service started, stopped or recreated, no database touched; only the
worktree's `web/node_modules` (git-ignored) and `web/dist` (git-ignored) were written.

## 1.7 Mobile: pickers, conversion and upload

Status: done, 2026-11-07.

Built:
- The `attachment` item in the app (`lib/questionnaire.dart`): `ItemType.attachment`, `Item.accept` (from the
  `mimeType` extensions) and `Item.maxFiles` (from `questionnaire-maxOccurs`, 1 when absent); a `valueAttachment`
  answer read as an `Attached` file, its id taken from the download URL's `/binaries/<id>/content` (the URL itself
  is never fetched); `toApi` never sends a file with the typed answers.
- The file upload item on the questionnaire screen (`lib/questionnaire_files.dart`): the hint (M2), the files with
  their status (M4, M5, M13, M14), "Add file" (M1) while the item has room, the source sheet with only the item's
  sources (M3), the camera's permission (M18 on Android before the system prompt, M19), the photo library refused
  (M20), the errors on the item (M7, M8, M9), "Remove file" with its confirmation (M6), and the files read only once
  submitted (M12, M13).
- `lib/uploads.dart`: each picked file prepared, uploaded one at a time with its progress, then its status read every
  2 s, slowing by 2 s a read after the first 10 s to every 10 s, until `clean` (Added) or `rejected` (gone, M7); a
  lost connection or a 5xx is M5 with "Try again" (the whole file again) and "Remove"; `too_large`,
  `unsupported_type`, `too_many_files`, `response_full` and any other 4xx are M8 on the item; the response's own
  refusals (`submitted`, `withdrawn`, `past_due`, `consent_required`, `appointment_closed`, 404) end filling in as
  for the answers. Submit waits while a file is pending, with M10 in the footer; a 409 `attachments_pending` is M10
  and leaves the form open. Leaving during an upload asks with M11 and stops the upload. The files of an open
  response are read for their status when it opens.
- The upload (`Booking.upload`, `lib/booking.dart`): a hand-written streamed request on the generated client's base
  URL, bearer token and `http.Client` (the generated `uploadAttachment` sends a multipart form): `POST
  …/items/{linkId}/attachments`, `Content-Length` declared (not chunked), `Content-Type: application/octet-stream`,
  `Content-Disposition: attachment; filename*=UTF-8''…` (RFC 6266/8187, `'()*!` encoded too), the body read from the
  file as the connection takes it, abortable. `fileStatus` and `removeFile` go through the generated
  `AttachmentsApi`.
- Preparing a file (`lib/file_prep.dart`, `lib/metadata.dart`): the type read from the bytes; HEIC/HEIF converted to
  JPEG and a MOV or any video that is not H.264/AAC MP4 (brand, `stsd` sample entries) converted to MP4 by the
  platform, then checked again; then the metadata stripped in Dart: JPEG without every APP segment and comment
  (Exif, XMP, IPTC, JFIF, maker notes) except the ICC profile and Adobe's colour transform, with a new Exif holding
  only the orientation, and without anything after its end (appended previews and depth maps with their own Exif);
  PNG with only the chunks needed to show it and an `eXIf` holding only the orientation; MP4/MOV with `udta`, `meta`
  and `uuid` (top level, `moov`, `trak`) turned into `free` boxes of zeros and the creation and modification times of
  `mvhd`, `tkhd` and `mdhd` zeroed, so no offset moves and the track matrix (the orientation) stays; the video is
  copied in 1 MiB chunks with the patches written over it. PDFs go as they are. A photo past 20 MB is refused before
  it is read into memory (M8). The copies made, and the pickers' copies in the temporary directory, are deleted once
  uploaded, removed or left; a file outside the temporary directory is never touched.
- `lib/files.dart`: the `Files` interface (pick, prepare) the screen sees, faked in the tests
  (`test/fake_files.dart`), and `DeviceFiles` on image_picker (camera, video camera, library; `requestFullMetadata:
  false`, so iOS needs no photo library permission), file_picker (PDF) and the channel `ba.salus.salus/files`.
- Platform conversion: `MainActivity.kt` (HEIF through `ImageDecoder`, which writes the photo upright, to a JPEG with
  no metadata; video through Media3 Transformer to H.264/AAC), `AppDelegate.swift` (`CGImageDestination` with only
  the orientation; `AVAssetExportSession` with the 1920x1080 preset, `metadata = []` and `.forSharing()`). Errors carry
  no message to Dart. `README.md` lists the manual device check.
- Permission text: M15 and M16 replace the two consultation strings in `Info.plist`, M17 added
  (`NSPhotoLibraryUsageDescription`); M18 to M20 in the app. Android needs no new permission (the camera is
  declared for the consultation).
- Folded in (owner): `compileSdk = 37`; the questionnaire footer with a white background and a top border
  (`outlineVariant`), the saved state always in it ("Saved" with nothing unsaved); no slot offered that starts
  within 5 minutes of now (`slots_screen.dart`).

Failing first (each seen before the code existed):
- `book_test.dart` "a time 299 s away is not offered": the 14:00 button was found. `android_build_test.dart`:
  `Expected: contains 'compileSdk = 37'`.
- `questionnaire_screen_test.dart`: the two tests whose "Saved" and "Not saved…" expectations now name the footer
  (owner's change, below) and the new footer test: 0 widgets under `questionnaire-footer`.
- `file_prep_test.dart`: compilation failed (`lib/file_prep.dart` missing, `crc32`, `Format` undefined).
- `questionnaire_test.dart`: compilation failed (`ItemType.attachment`, `Attached`, `Item.accept`, `Item.maxFiles`
  not found).
- `copy_test.dart`: the attachments strings missing from `messages.dart` ("Add file" first), the new purpose strings
  missing from `Info.plist`.
- `attachments_test.dart`: compilation failed (`questionnaire_files.dart`, `uploads.dart` missing, `fileHint`,
  `pollDelay`, `fileSize`, `contentDisposition` undefined, `SalusApp` without `files`). With the code, three failed
  for the test's own reasons (Submit not yet built in the lazy list, "View answers" is not a FilledButton) and one
  for the code's: the file row and progress labels were merged into the question's node; they are now their own
  semantics nodes (M14).

Tests: `test/file_prep_test.dart` (30: detection; a generated JPEG with GPS, Make, Model, Orientation 6, XMP, IPTC, a
comment and an appended picture comes out with one Exif holding Orientation 6 and nothing else, the colour profile
and the picture bytes untouched; without an orientation or upright, no Exif; PNG text, XMP, time and Exif gone, the
orientation kept, the checksums valid; MP4 with `moov` first and last: location, device, Apple keys and XMP gone,
times zeroed, the rotation matrix and `mdat` untouched, the length unchanged; which videos are converted; a box that
lies; `prepareFile` with a fake conversion: JPEG, HEIC, MOV, H.264 MP4, PDF, the copies deleted and the picked file
kept, the refusals, a photo past 20 MB). `test/attachments_test.dart` (41: the hint per combination of kinds and
numbers, the sheet per combination, the library's request, a cancelled pick; the upload's body, declared length,
headers and empty query, the status sequence and the polls' timing, the progress and its screen-reader labels, a
failed upload retried with the whole file, a failed one removed locally, each error code's text, a video's limit, M9
and a photo too large before upload, a response refusal ending the form, a rejected file, one upload at a time,
nothing printed; remove with its confirmation and the DELETE, a required item on submit, the 409, the read-only list
after submit, the leave prompt; an open response's files read for their status, a submitted one's listed with kind
and size; M18 on Android, M19 with the settings, M20 and the picker's camera refusal; the sizes; the
Content-Disposition). `questionnaire_test.dart` (3 new), `copy_test.dart` (attachments strings, the three plist
strings), `book_test.dart` (2), `questionnaire_screen_test.dart` (1), `android_build_test.dart` (1). 292 mobile tests
in all.

Checks:
- `make mobile-analyze` (`flutter analyze` in the flutter service, `--no-deps`): no issues.
- `make mobile-test` (`flutter test`, same): 292 passed.
- `flutter build apk --debug`: built (`app-debug.apk`, 229 s for Gradle). The flutter service has no Android SDK
  (`precache --no-android`), so it ran once in a throwaway container from the local `salus-demo-android` image (Flutter
  3.47.7, Android SDK with platform 37, JDK 21) on a copy of `mobile/` in the session's scratchpad: `compileSdk = 37`,
  `MainActivity.kt` and Media3 1.11.1 compile. Gradle installed the NDK r28c, platform 35 and CMake 3.22.1 into that
  container only. The iOS code (`AppDelegate.swift`) could not be compiled here (no macOS); it is checked in 4.1.
- Device check: in 4.1 (the acceptance walk); the manual steps are in `mobile/README.md`.

Dependencies (pinned exact; versions at least two weeks old):
- `image_picker` 1.2.4 (2026-10-06), BSD-3-Clause, the Flutter team's plugin. The camera (photo, video) and the
  system photo picker on both platforms, with the camera and photo-library errors as codes. Our own channel would
  need `UIImagePickerController`/`PHPickerViewController` on iOS and the camera and photo-picker intents with a
  `FileProvider` and the activity results on Android: several hundred lines of untested platform code. Runtime
  dependencies: its federated implementations (`image_picker_android` 0.8.13+25, `image_picker_ios` 0.8.13+15,
  `image_picker_platform_interface` 2.11.2, and the unused desktop and web ones), `cross_file`, `mime` 2.1.0,
  `flutter_plugin_android_lifecycle` 2.0.35, and on desktop `file_selector_*`.
- `file_picker` 13.1.0 (2026-09-15; 13.2.0 was 14 days old and 13.3.x newer), MIT. The system document picker for
  PDFs (`UIDocumentPickerViewController`, the Storage Access Framework) with the file copied to a local path. Our
  own would be about 80 lines of Swift and Kotlin with activity results and content-URI copying, untested here.
  Runtime dependencies: `android_file_picker` 2.1.3, `file_picker_darwin` 2.4.2, `file_picker_platform_interface`
  4.2.0, `cross_file` and the unused desktop and web implementations. The SPEC names both.
- `http` 1.6.0: already the generated client's dependency (and a dev dependency here); now a direct one, for the
  streamed upload.
- Android: `androidx.media3:media3-transformer` and `media3-common` 1.11.1 (2026-09-11), Apache-2.0, Android's own
  library; it brings `media3-exoplayer`, `media3-effect`, `media3-muxer`, `media3-datasource`, `media3-container`
  (1.11.1) and AndroidX annotation and concurrent-futures. Transcoding through `MediaCodec` and `MediaMuxer` by hand
  would be several hundred lines. iOS uses only the system's AVFoundation and ImageIO.
- Metadata stripping uses no package: the JPEG, PNG and MP4 walkers are about 250 lines of Dart, tested on generated
  files, where a package (`image`, `exif`) would decode and re-encode the picture or cover only Exif.

Decided while the owner was away:
- The binary id of a `valueAttachment` is read from its `url` (`…/binaries/<id>/content`); SPEC §2 "FHIR" says the
  apps build the URL from the id but names no other place the id is carried. No SPEC row needed if 3.2 renders the
  URL as the SPEC says; if it renders the id elsewhere, the app follows it.
- The files of an open response are read for their status (`GET /binaries/{id}`, audited) once when the screen opens,
  since the FHIR answer does not say which are pending. One audited read per file per opening.
- A file that failed to upload (M5) does not hold Submit back; only files being prepared, sent or checked do (M10).
  Removing it needs no confirmation: the server never had it.
- A photo past 20 MB is refused on the device (M8) before it is read into memory to strip it; a video's and a PDF's
  limits are left to the server, whose 413 comes from `Content-Length` before any byte is sent. Sizes in decimal
  units (20 MB = 20,000,000 bytes), as shown in M13; should the server count MiB, the app is only the stricter.
- A status read answering 404 (the file is gone, for example after a withdrawal) removes the row without a message.
- The JPEG keeps its ICC colour profile (APP2) and Adobe's colour transform (APP14): they say how to show the colours,
  hold no location, time or person, and a skin photo shown in the wrong colours would mislead. Everything else goes,
  including JFIF. The PNG likewise keeps its colour chunks and `pHYs`. SPEC row needed (O13 names EXIF, XMP and IPTC
  only).
- Metadata is stripped in Dart after the platform conversion, not by the conversion alone, so the same tested code
  covers every path (a JPEG or MP4 that needs no conversion included). The orientation is kept as the only Exif tag
  (SPEC §2 allows either way); a converted HEIC on Android is written upright by `ImageDecoder` with no tag.
- A video conversion uses the 1920x1080 preset on iOS (H.264; HEVC presets are separate), which also brings a 4K
  video under the 200 MB limit sooner.
- The footer's "Saved" shows also before the first change: with nothing unsaved, everything is saved ("always
  visible", owner).
- The owner's change of the questionnaire tests: "Saved" and "Not saved…" are now expected inside the footer
  (`questionnaire-footer`), where they were expected anywhere on the page before; the test "an item type the app
  does not know is kept as unsupported" uses `quantity`, as `attachment` is now known; the iOS purpose-string test
  reads `docs/attachments/COPY.md` (M15, M16 replace the consultation's strings, M17 added).
- `files` is passed down the screens next to `media` (`SalusApp` to `QuestionnaireScreen`), as the camera and
  LiveKit are.

Copy added by the session, not yet reviewed: none. Every new string is in `docs/attachments/COPY.md`.

Dev stack: nothing run against it (no Go, no database, no object store); only the flutter service, and one throwaway
container from the `salus-demo-android` image (its Android SDK) for the APK build, on a copy of `mobile/` in the
session's scratchpad, removed after the run.

## 2.1 internal/binary: service, Postgres, handlers and wiring

Status: done, 2026-11-07.

Built:
- `internal/binary`: `model.go` (`Binary`, `Status`, `Kind` with its `Limit`, the O2 limits in bytes, `Caller`,
  `Target`, `Grant`, `Attachment` with the JSON of `questionnaire_response.attachments`, `Upload`, `File`, `ScanArgs`
  on the queue `scan` and `DiscardArgs` with their kinds); `errors.go` (every refusal, `ErrObjectNotFound` moved here
  from `repository.go`); `service.go` (the ports `Holder`, `Auditor`, `Transactor`; scope, `Status`, `Delete`, the
  audit); `service_upload.go` (`Upload`, SPEC §2 "Upload pipeline" step by step: the holder's target and the limits
  by the declared length in a transaction; 4 KiB head and `filetype.Detect`; the rest through
  `filetype.LimitReader`, SHA-1 and the MP4 `Checker` into `envelope.Writer` and one `Store.Put` of
  `CiphertextSize` through a pipe, outside any transaction, the last segment sealed only once the checker has seen
  the whole file; the commit under `pg_advisory_xact_lock` with the target and the limits again, the identical
  retry, the row, `Holder.Attach` pending, the scan job, the audit; the object deleted after any failure once
  stored); `service_download.go` (`Download`: the read, `Holder.Readable`, `clean` required, audited and committed,
  then the data key unwrapped; `Content`: `envelope.OpenRange`, one ranged `GetRange`, decrypted); `repository.go`
  (the `Repository` and `Jobs` ports beside `Store`); `repository_postgres.go`, `query.sql`, `binarydb/` (an entry
  in `sqlc.yaml`; `"binary"` quoted); `repository_river.go` (`InsertTx` on the `scan` queue); `handler_http.go`
  (`Routes`: `GET` and `DELETE /binaries/{id}`; `StreamRoutes`: the upload and `GET /binaries/{id}/content`, with
  their own read and write deadlines through `http.ResponseController`, 30 minutes and 10 minutes, the `Range`
  parsing of SPEC §2 "Download", every download header, the title from `Content-Disposition`).
- `repository_s3.go`: a fix found by the upload, below.
- `internal/questionnaireresponse`: `service_attachments.go`, the `binary.Holder` (`UploadTarget`, `Editable`,
  `Readable`, `Attach`, `Detach`, `Cleared`, `Attached`); the repository's `AddAttachment`, `RemoveAttachment`,
  `ClearAttachment` and `Attachments`, each one SQL statement on the `jsonb` column (append, filter in order,
  `pending` set false), `updated_at` changed, `version_id` not; `query.sql` selects `attachments` in its other
  queries too, so the regenerated `questionnaireresponsedb` keeps one row type (0.1 had not regenerated sqlc after
  its migration; every domain's `models.go` gains the `Binary` model and the column).
- `internal/api/router`: `Config.Streams`, the streaming routes, mounted on the same mux and served without
  `http.TimeoutHandler` (a second mux only tells their requests apart, so the access log still names the pattern and
  a wrong method still gets 405). The 256 KiB limit is each JSON handler's own `MaxBytesReader`; the binary handler
  has none, `filetype.LimitReader` bounds the upload to its `Content-Length`.
- `internal/api/app.go`: `New` takes `config.ObjectStore`; the keys are checked before anything connects
  (`checkKeys`, which now also holds the TOTP, rate-limit and FHIR base checks, so `New` stays under 80 lines);
  `questionnaireRoutes` builds `questionnaireresponse.Service`, then `binary.Service` with it as the holder, the S3
  store (`&http.Client{Timeout: binary.UploadDeadline}`, page size 1000), the KEK and the River jobs, and returns
  the routes and the streams. `cmd/api/main.go` calls `config.LoadObjectStore`.
- `Makefile`: `bench-binary-http` (`go test -tags loadtest -run TestBinaryHTTPReadings ./internal/api/`).

Failing first (each seen before the code existed):
- `internal/binary`: the service tests and fakes failed to build (`undefined: binary.NewService`, `binary.Ports`,
  `binary.Upload` …); with stubs every test failed on its assertions (`Upload()` returned nothing, so every seeded
  test stopped at its upload). Then, with the code: `s3 put: InvalidRequest` on every upload (below);
  `row-level security: the patient inserts only for themselves` (the upload's transactions did not scope the
  repository to the patient: fixed); the two "the response's bytes" and "the item's files" cases read the body
  (the same missing scope); the commit's refusal audited without the clinic (fixed).
- `TestS3Store_PutsABodyThatIsAReadCloser`: `Put() of a pipe = binary: s3 put: InvalidRequest`.
- `repository_postgres_test.go` failed to build (`undefined: binary.NewPostgresRepository`); the concurrency test
  was then run with the advisory lock taken out of the commit: both subtests "2 stored and 0 refused, 2 rows",
  three runs out of three; restored.
- `TestRiverJobs_…` with the queue option removed: "job binary_scan on default … want … on scan"; restored.
- `handler_http_test.go`: `undefined: binary.NewHTTPHandler`. `router_test.go`: `unknown field Streams`.
- `internal/questionnaireresponse`: `service_attachments_test.go` failed to build (`f.svc.UploadTarget undefined`
  …); `TestPostgres_AttachmentsInThePatientsAndTheJobScope` failed on its own assumption (another response's job
  scope named with the response's clinic sees the response through the clinic's policies, as 0.1 recorded); the
  test now names another clinic.
- `internal/api`: the wiring was written before `app_binary_test.go`; that test was seen failing with the file
  routes taken out of `questionnaireRoutes` (404 on the upload), then restored. `cmd/api`'s
  `TestRun_ServesUntilCancelledThenShutsDown` failed once `main.go` required the store (`S3_ENDPOINT is
  required`); it now sets the synthetic settings, and `TestRun_FailsWithoutTheObjectStoreSettings` was added.

Tests:
- Service (fakes for the repository, the holder, the audit, the jobs and the transaction; the real compose Garage):
  each type stored, its object opening to the same bytes, the row, the pending attachment, the scan job and the two
  audit events; every holder refusal without reading the body, audited without the response for `not_found`;
  411, 413 by header (every kind's limit, the photo's, the document's), `response_full` and `too_many_files` by
  header; 415 by sniff (text, a PDF to a photo item, a JPEG to a video item, an HEVC MP4 found by the checker
  during the stream), 413 by kind once the head is read, 400 for a body shorter than its length (at the head and
  at the end), each refused stream leaving no object; the identical retry returning the first file with 200 and
  deleting the second object, also on a full item; a different file of the same size on a full item read and then
  refused; a failure after the put (the holder's attach, a submit meanwhile, the store) deleting the object and
  leaving no row; a cancelled upload ending without a blocked goroutine. Status and download for the patient and
  the practitioner with their purposes, refused without disclosing the file (another patient, no profile, no such
  id, not a UUID, the holder's refusal); `scanning` while quarantined and `not_found` when rejected, audited with
  the entities; ranges against the real store (the whole file, the first byte, the last, across segments, a
  suffix; outside the file `ErrRangeNotSatisfiable`); another KEK's key a failure; `Delete` by the patient (row,
  attachment, object, audit), refused for `submitted`, `withdrawn`, `past_due`, `consent_required` and staff
  (`forbidden`), keeping the file; backend errors are failures, not refusals, and leave no event.
- Postgres (`NewAppDatabase`): the patient inserts and reads their file; the clinic and the job scope of the
  response see it, another patient and another response's job scope do not; an insert for another patient refused;
  `OfItem` oldest first without the rejected, `ResponseSize`; only the patient deletes in their scope and the job
  scope in its own; two concurrent uploads that each fit but together pass the item's files or the response's
  bytes: one stored, one refused, one row.
- Holder (`questionnaireresponse`, fakes): the target with the item's kinds and maximum and the appointment
  locked, no event of its own; `UploadTarget` and `Editable` refuse as the answers' changes do (every case of the
  answers' table but the stale version), mapped to binary's errors; an item that is no attachment (a boolean, a
  group, none) refused; `Readable` at `Full` only, with `PATRQT` or `TREAT`, D9 allowed, a future appointment
  after a revocation refused; the attachments added, cleared and removed in step, `version_id` unchanged. On the
  real database: the four repository calls in the patient's and the job scope, `updated_at` moved and
  `version_id` kept, another clinic's job scope refused.
- HTTP (`httptest`, a stub service): the upload's status and JSON, the title from `filename*` (path stripped on both
  separators, cut to 200 characters, control characters dropped) or `filename`, none on a malformed header; no
  length sent as -1 (411); every error's status and code on all four routes, 500 fixed, 401; status and
  deletion; content whole and by one range (first, last, middle, open end, suffix, a suffix past the size, an end
  past the size), several ranges and malformed ones the whole file, 416 with `Content-Range: bytes */<size>`, every
  header; the CSP per type and `Content-Disposition: inline` without a title; the log holds no title. On a real
  server with 300 ms read and write timeouts: a 10 KB upload sent over a second completes on the stream route and
  is cut on a plain route.
- Router: a stream route runs past the request timeout with the session and its deadlines, the same handler as an
  ordinary route gets 503, the access log names the stream's pattern, a wrong method 405.
- App (`internal/api`, `salus_app`, the compose store): upload 201, identical retry 200, a text file 415, a boolean
  item 422, the practitioner's upload 403 and deletion 403, another practitioner 404, nobody 401, status 200,
  `scanning` 409; after a hand-set `clean` the patient's download and a 206 range; the practitioner's download
  before the submit 404; the deletion 204, then 404; the audit (2 creates, 3 reads, 1 delete, 2 refusals on
  `Binary/<id>`, each beside its `QuestionnaireResponse/<id>` event); the log holds no title or file bytes.

Checks (all `--no-deps`, `checks.sh` in the session's scratchpad):
- lint (gofmt, `go vet`, golangci-lint both modules, check-migrations with the main git dir mounted read-only):
  0 issues, "ok (no migration from main changed)".
- archcheck: pass (after replacing a test's use of `auditevent.NewPostgresRepository` with the fake audit).
- test (`go test -count=1 ./...` both modules): pass.
- cover: pass, total 94.5%; `internal/binary` 91.8%, `internal/questionnaireresponse` 93.8%, `internal/api` 91.8%,
  `internal/api/router` 100%.
- fhir-validate: pass (no FHIR output changed). gen-clients: no diff (`api/openapi.yaml` unchanged).
- Web and mobile: not run (nothing under `web/` or `mobile/` changed).
- Stack: the api rebuilt from this worktree (`up -d --build --no-deps --wait api`, healthy). A synthetic patient
  registered and signed in through the api; a patient, consent, appointment (2027-03-03), questionnaire version
  with an attachment item and in-progress response inserted as the owner. From the `tools` container with `curl`:
  a 300,049-byte generated PDF uploaded (201, `quarantined`, title from `filename*`), its status read (200,
  `quarantined`), its download refused (409 `scanning`); no length 411, a text body 415; the attachment listed
  pending with its base64 SHA-1 and the `binary_scan` job on the `scan` queue; after a hand-set `clean` the
  download 200 with every header and the same SHA-1, `bytes=-100` 206, past the end 416; the deletion 204, then
  404. The api's log holds no title. Reverted: the file deleted through the api (row and object), the scan job, the
  response, questionnaire, appointment, consent, patient, sessions and account deleted; the bucket at 0 objects.

Readings (`make bench-binary-http`, run as its command with `--no-deps`; the api's `http.Server` with
`cmd/api`'s timeouts in the test process, the compose Postgres and Garage, i7-14700HX):
- The 200 MB generated MP4 uploaded through the handler: 425 to 563 ms, 356 to 471 MB/s over three runs; the
  process's heap (`/memory/classes/heap/objects:bytes`, sampled every 10 ms) 3.8 MiB before and 4.4 to 4.5 MiB at
  its peak, a growth of 0.6 to 0.7 MiB, whatever the file's size: one segment buffer, the pipe and the S3 client's.
  Far faster than any phone's upload; nothing to fix.
- 1,000 random 1 MiB ranges of it (each audited in its own transaction, then one ranged `GetObject` of 17
  segments and the decryption): p50 12.0 to 19.4 ms, p95 24.2 to 35.8 ms, p99 51.6 to 62.6 ms.

### Decided while the owner was away

- 2026-11-07, session (2.1): `binary_scan` carries the response's and the clinic's ids beside the binary id. §2
  "Scan" runs the job in the response's job scope, which row-level security requires before the job can read the
  row; with the binary id alone package 3.1 would need a new policy or a bypass. Opaque ids only. SPEC §3 O21 (and
  the §5 "River jobs" row).
- 2026-11-07, session (2.1): `audit_event` holds one entity, so each audited call writes one event per entity in
  its transaction (same request id, outcome and purpose): `Binary/<id>` and `QuestionnaireResponse/<id>`. A
  migration of the audit table would have been outside the package and the plan's tables. SPEC §3 O22.
- 2026-11-07, session (2.1): the download's CSP is `default-src 'none'` for PDFs and `default-src 'none'; sandbox`
  for the other types. Chrome was not available to the session; Chrome's PDF viewer refuses to render a sandboxed
  document, so the PDF keeps working in 1.6's `<object>` without `sandbox`, and the sandbox still covers what a
  browser shows natively. 4.1's walk checks it in Chrome. SPEC §3 O23.
- 2026-11-07, session (2.1): before the bytes are read, `too_many_files` and `response_full` wait when the item
  already holds a file of exactly the declared size, so the retry of a lost answer on a full item gets its file
  back (§2 "Retried uploads") instead of a 409; the commit checks both again under the lock. SPEC §3 O24.
- 2026-11-07, session (2.1): `Holder.Attach(ctx, responseID, Attachment)` without the separate `linkID` of §2
  "binary.Holder": the `Attachment` carries its `linkId`, as the stored JSON does. Same call, one argument fewer.
- 2026-11-07, session (2.1): the title is the `filename*` of `Content-Disposition`, or its `filename` when that is
  all a client sends (RFC 6266 allows both; Go's `mime` decodes `filename*` into the same parameter); control
  characters are dropped so a title never breaks a header or a log viewer; the response's JSON gives the title
  stored.
- 2026-11-07, session (2.1): `UploadTarget` reads the version's items through `questionnaire.Service.Referenced`,
  which audits that read as the patient's read of the Questionnaire, as `SaveAnswers` does. The holder's own
  checks stay unaudited; binary audits the upload.
- 2026-11-07, session (2.1): an event names the file's clinic once the file or the target is known, and the
  caller's clinic otherwise (none for a patient), as the response's own events do.
- 2026-11-07, session (2.1): `GET /binaries/{id}` returns a `rejected` file's status while its row lives (24 h, 3.1),
  so the app can stop polling; its content is 404.
- 2026-11-07, session (2.1): the content handler cuts the connection (`http.ErrAbortHandler`) when the store or the
  decryption fails after the status is sent, so a client never takes a short body for the whole file; the log line
  carries the route and the binary id only.
- 2026-11-07, session (2.1): the S3 client's `http.Client` has the upload's deadline (30 minutes) as its timeout, as
  CLAUDE.md asks of every injected client and as 1.4 required (no shorter one).
- 2026-11-07, session (2.1): `repository_s3.go` (1.4) sends the body to the SDK as a plain `io.Reader`: an
  `io.ReadCloser` body such as the upload's pipe went out without its length and Garage answered `InvalidRequest`.
  Tested (`TestS3Store_PutsABodyThatIsAReadCloser`).

Copy added by the session, not yet reviewed: none (the API's codes are developer text; `COPY.md` unchanged).

For later packages:
- 3.1: `Repository.SetJob(ctx, responseID, org)` is the job scope; `ScanArgs` carries all three ids; `Holder.Cleared`,
  `Detach` and `Attached` run in binary's transaction with that scope set (the `questionnaire_response_job`
  policies). A rejected row must keep `binary_key` (wrapped key cleared) and `binary_scanned`. Scan jobs inserted
  from 2.1 on wait on the `scan` queue, which no worker serves until 3.1.
- 3.1: in the job scope, `questionnaire_response`'s clinic policies also apply (0.1), so query by response id.
- 3.2: `StopQuestionnaireResponse` does not clear `attachments` yet, so until 3.2 a withdrawal of a response with
  files fails the `questionnaire_response_withdrawn_attachments` check (500); 3.2's `Withdraw` clears both and
  enqueues `binary_discard`. `Repository.Attachments` gives the list for the FHIR `valueAttachment`s
  (`hash` is base64 in the stored JSON already). `PUT /answers` does not refuse an attachment answer yet (3.2).
- 4.1: walk the PDF viewer under `Content-Security-Policy: default-src 'none'` in Chrome (O23).

Dev stack: the api container rebuilt from this worktree (the only service touched). On the dev database the stack
check's rows are deleted; its audit events stay (append-only: the registration, the login, the file's 7 events and
the response's). The bucket `salus-binaries` holds 0 objects (a 200 MB object left by the first bench run, whose
cleanup ran on a cancelled context, was deleted; the bench now deletes in a `defer`).

## 3.1 Scan, discard and purge

Status: done, 2026-11-07 (wave 3).

Built:
- `internal/binary/service_scan.go`: the `Scanner` port (`pkg/clamd.Client`) and `Scan(ctx, ScanArgs) (ScanResult,
  error)`, SPEC §2 "Scan": in the job scope of the file's response, the quarantined row read in one transaction; then,
  outside any transaction, the whole object read with one `GetRange`, opened with `envelope.Open` and streamed into
  clamd's `INSTREAM`; then one transaction: `MarkScanned` (only while quarantined), `Holder.Cleared` (clean) or
  `Holder.Detach` (rejected, wrapped key cleared), the audit; the rejected object deleted after the commit. Any error
  leaves the file quarantined and River retries. A file that is gone or no longer quarantined is left as it is.
- `internal/binary/service_discard.go`: `Discard(ctx, DiscardArgs) (int, error)`, SPEC §2 "Discard": in the job
  scope, every file of the response that `Holder.Attached` no longer lists is deleted (row, audit, then the object
  after the commit); a response without files asks the holder nothing.
- `internal/binary/service_retention.go`: `Purger` (`NewPurger(repo, store)`): the rejected rows scanned before the
  cut-off, then every object written before it whose key has no row (checked page by page with one query); it counts
  both.
- `repository.go`, `repository_postgres.go`, `query.sql`, `binarydb/`: `MarkScanned`, `OfResponse`,
  `DeleteRejectedBefore`, `Existing` (an id that is no UUID has no row).
- `service.go`: `Ports.Scanner`, and an audit entry may name a job as its agent (`AgentName`, purpose `HOPERAT`,
  `update` mapped to `U`).
- `migrations/00028_binary_retention.sql`: `GRANT SELECT, DELETE ON "binary" TO salus_retention` and the partial index
  `binary_rejected_scanned` (`scanned_at` where `rejected`), for the purger.
- `internal/retention/rules.go`: `binary` is `delete`, 24 h, with its basis.
- `internal/worker/binary.go`: `Files` (store, KEK, scanner), the binary service on the app pool with the
  questionnaire response as holder, `binary_scan` (timeout 10 minutes) and `binary_discard` workers and their log
  lines. `worker.go`: `New` takes `Files` and refuses to start without them; the worker works the default queue (10)
  and `scan` (2); both kinds registered for the scheduler too. `retention.go`: binary's purger on the retention pool
  and the store.
- `cmd/worker/main.go`: `config.LoadObjectStore`, `config.LoadClamd`, the S3 store (HTTP client timeout
  `binary.UploadDeadline`, page 1000), the KEK, `clamd.New`.

Failing first (each seen before the code existed):
- `migrations`: `TestBinary_TheRetentionRoleReads…` "permission denied for table binary (42501)" for the read and the
  purge; `TestBinary_RejectedFilesAreIndexedByTheirScanTime` "no index binary_rejected_scanned".
- `internal/retention`: `TestRules_DeleteRejectedBinariesAfterTwentyFourHours` "binary then keep after 0s"; every
  `Apply` test "no applier … binary" once the appliers named binary's purger; `no binary purger` accepted while the
  rule was keep.
- `internal/binary`: the repository tests failed to build (`missing method DeleteRejectedBefore`), the service tests
  (`undefined: binary.Scanner`, `Scan`, `Discard`, `NewPurger`). With the code, the EICAR test failed on its expected
  name (`Eicar-Signature`, not the bare string's `Eicar-Test-Signature`; the test now takes any `Eicar` name).
  Mutations seen failing: the rejected object's deletion taken out (the object still there), the `quarantined`
  guard of `MarkBinaryScanned` taken out (the repeated mark succeeds and changes the rejected row).
- `internal/worker`: the build failed (`undefined: workerQueues`, `worker.Files`); with the scan queue taken out of
  `workerQueues` the queue test failed and the scan job "did not finish within 15s".
- `cmd/worker`: the build failed until `main.go` passed `Files`; `TestRun_FailsWithoutTheFilesSettings` covers each.

Tests:
- Service (fakes for the repository, holder, audit and transaction; the real compose Garage and clamd): a clean
  generated MP4 becomes `clean` at the clock's time with its key, its attachment no longer pending, two `update`
  events (`Binary/<id>`, `QuestionnaireResponse/<id>`) by `binary_scan` for `HOPERAT`, the object unchanged; a
  generated PDF carrying the EICAR string becomes `rejected`, key cleared, detached, two `delete` events, the object
  gone; clamd unreachable (`127.0.0.1:1`) returns `ErrUnreachable` and changes nothing; a repeated scan, a gone file
  and another clinic's job scope do nothing; a commit that fails (the holder's detach) keeps the file quarantined,
  unaudited, with its object. `Discard`: a clean, a quarantined and a rejected file of which the holder lists one:
  the two others deleted with their objects and two events each by `binary_discard`, the listed one kept; repeated,
  nothing; another response or clinic's scope, nothing; a database error keeps every file.
- Postgres: the job scope marks its response's quarantined file once (clean; rejected with the key cleared),
  another response's job scope, the patient and a non-UUID never; `OfResponse` lists every status of its response
  only. As `salus_retention`: `DeleteRejectedBefore` deletes the old rejected row and keeps a fresh rejected, an old
  clean and a quarantined one; `Existing` names the rows among keys with an unknown id and a non-UUID key. The app
  role purges nothing.
- Purger (`salus_retention` and the real store, pages of 2): keeps a fresh rejected row and deletes an old one, then
  deletes the old rejected file's object and an old object without a row, keeps a young object without a row and an
  old one with a row; 3 counted. The store is seen through a key prefix of the test's own, so a test's purger never
  lists or deletes another test's or the dev stack's objects. A closed pool fails the purge.
- Migrations: `salus_retention` reads every clinic's files, deletes the rejected row past 24 h, cannot update, insert
  or truncate; the partial index. 0.1's cascade test revokes the new grant in its own database first, so it still
  proves the cascade needs no privilege of the role's.
- Worker (real database, store under a prefix, clamd): `binary_scan` on the `scan` queue marks a clean file clean
  (two `U` events) and rejects an EICAR PDF (key cleared, detached, two `D` events, object gone); clamd unreachable
  fails the attempt (River retries) and leaves it quarantined and pending; `binary_discard` deletes the file the
  response no longer lists (two `D` events) and keeps the other. Every job log holds the ids and the result and no
  title, patient id, e-mail or name; the signature only on the rejected line beside the binary id. `retention_apply`
  logs and audits `binary 0`; the retention pool reaches `binary.NewPostgresRepository` too; the scheduler knows both
  kinds; `New` refuses to start without the store, the key or clamd; the queues are default 10 and scan 2.
- `cmd/worker`: starts and stops with the synthetic settings; fails naming `S3_ENDPOINT`, `BINARY_KEK` and
  `CLAMD_ADDR` without each.

Checks (all `--no-deps`):
- Tests: `go test -count=1 -cover ./internal/binary/... ./internal/worker/... ./internal/retention/... ./migrations/...`
  pass: `internal/binary` 90.5%, `internal/worker` 93.6%, `internal/retention` 100%. Also `./cmd/worker/...` (Postgres
  only, no Redis) and `./internal/config/` (no stack) pass; `go vet ./...` passes.
- Lint (`scripts/lint.sh`, main git dir mounted read-only): 0 issues both modules, check-migrations ok.
- Archcheck (`go -C tools run ./archcheck -root /src`): pass.

Readings (`go test -tags loadtest -run TestScanReadings -v ./internal/binary/`, the compose Garage and clamd 1.5.4,
i7-14700HX; clamd's `VmRSS` sampled every 0.2 s from `/proc` in its container):
- The scan of a 200 MB generated MP4 (199,999,606 bytes, read from the store, decrypted and streamed to clamd): 6.05 s
  and 5.85 s (33 and 34 MB/s) on two files clamd had not seen; 0.57 s for one it had seen clean before (clamd caches
  the hashes of clean files).
- clamd's resident memory: 941 MiB idle, 969 MiB at its peak during the scans (+27 MiB): clamd spools an `INSTREAM`
  to a temporary file, so two scans at a time stay far inside its memory. The job's timeout (10 minutes) is a hundred
  times the measured scan; River's default minute would have been close for a slow production store.

### Decided while the owner was away

- 2026-11-07, session (3.1): the rejection's `AuditEvent` does not carry the signature name. `audit_event` has no
  column for it, and adding one means changing `internal/auditevent` and migrating the audit table, outside this
  package's files and against O22's "no migration of the audit table". The rejection is audited (a `delete` by
  `binary_scan`, since the key and the bytes are gone), and the signature goes on the job's log line beside the binary
  id only, which §5 allows (never beside a patient id). SPEC row needed (§2 "Scan" says "audited with the signature
  name").
- 2026-11-07, session (3.1): the scan's audit codes: a clean result is an `update` (`U`), a rejection a `delete` (`D`),
  each by the job (`AgentName` `binary_scan`, no account) for `HOPERAT`; the discard is a `delete` by `binary_discard`.
  One event per entity (O22). A scan that finds its file gone or already scanned writes nothing: it changed and
  disclosed nothing.
- 2026-11-07, session (3.1): `Discard(ctx, DiscardArgs)` instead of §2's `Discard(ctx, agent, org, responseID)`: the
  agent is always the job's kind, and the job's arguments are the response and the clinic, as `Scan(ctx, ScanArgs)`.
  Same behaviour, no unused parameter. It deletes every file the holder no longer lists, a rejected row included. SPEC
  row needed (the signature in §2 "Discard").
- 2026-11-07, session (3.1): the purger deletes every object older than 24 h whose key has no row, including a key
  that is no UUID (it can never have one). In production the bucket is binary's alone. In the shared dev bucket that
  also removes test objects left behind for more than a day; it never removes an object of a dev file with a row.
  The tests see the store through a key prefix of their own (`prefixStore` in the test files of `internal/binary` and
  `internal/worker`), so no test's purger lists or deletes the dev stack's or another test's objects.
- 2026-11-07, session (3.1): migration 00028 also adds the partial index `binary_rejected_scanned`, so the daily purge
  does not read every file to find the few rejected ones.
- 2026-11-07, session (3.1): files outside the narrowest reading of the package's list were edited because the rule
  change forces them, each within the package's own concern: `internal/binary/service.go` (the `Scanner` port field,
  the job agent in the audit entry), `fakes_test.go` (the job scope and the holder's `Cleared` and `Attached`),
  `internal/retention/service_test.go` (binary's purger among the appliers, and the refusal without it),
  `migrations/binary_test.go` (0.1's cascade subtest revokes the new grant in its own database first, keeping what it
  proves), and the worker's existing tests (`New`'s new argument, `binary 0` in the retention log and audit,
  `binary.NewPostgresRepository` on the retention pool, the two kinds in the scheduler). No test was weakened.
- 2026-11-07, session (3.1): the binary jobs need the store, the KEK and clamd, so `worker.New` refuses to start without
  them and `cmd/worker` requires the settings, as it requires LiveKit's. Compose already gave the worker every
  variable (0.1); no compose change.
- 2026-11-07, session (3.1): `internal/config`'s test still accepts a subset of the callers of `LoadObjectStore` and
  `LoadClamd`; both calls now exist (api and worker; worker), so the integration step may tighten it to the exact list
  (not this package's file).

Copy added by the session, not yet reviewed: none.

For later packages:
- 3.2: `binary_discard` is served on the default queue from this package on; enqueue `binary.DiscardArgs` (response
  and clinic ids) with `InsertTx` in the withdrawal's and the submit's transaction.
- 4.1: the dev worker must be rebuilt to serve the `scan` queue (it was not rebuilt here, by the wave rules); scan jobs
  inserted by 2.1's stack check were deleted then, so none wait. The DPIA's 3.1 subsection: the signature is logged
  with the binary id only, never audited (decision above); the purger's 24 h rule; clamd runs on our own
  infrastructure. The rejected-file e2e can use the same uncompressed PDF content stream (`Eicar-Signature`).

Dev stack: nothing changed. No service was rebuilt or restarted; the dev database was not touched (migration 28 is
applied by the integration step's `make migrate`); the tests' objects were deleted (the bucket `salus-binaries` at
0 objects after the runs); clamd keeps the hashes of the readings' clean files in its cache.

## 3.2 internal/questionnaireresponse: attachment answers

Status: done, 2026-11-07 (wave 3, in parallel with 3.1).

Built:
- `model.go`: `Response.Attachments` (`[]binary.Attachment`), read from `questionnaire_response.attachments` by every
  query (`toResponse`); a Name view (`seen`) drops it with the answers.
- `query.sql`: `StopQuestionnaireResponse` sets `attachments = '[]'` with the answers (`questionnaireresponsedb`
  regenerated with sqlc; only this domain's `query.sql.go` changed). This fixes 2.1's hand-off: a withdrawal of a
  response with files no longer fails the `questionnaire_response_withdrawn_attachments` check (500).
- `repository.go`, `repository_river.go`: the `Jobs` port (`EnqueueDiscard(ctx, binary.DiscardArgs)`) and its River
  implementation (`InsertTx` in the transaction in the context, `postgres.ErrNoTx` outside one; default queue).
- `service.go` `Withdraw`: after `Stop`, `binary_discard` enqueued in the same transaction with the response's and
  the clinic's ids; an enqueue failure rolls the withdrawal back.
- `service_answers.go` `Submit`: `submitted` counts a file as the answer of an attachment item, so a required,
  enabled attachment item without a file is named in the `*AnswersError` (422) and a hidden one is not required;
  then any file still `pending` refuses the submit (`ErrAttachmentsPending`, 409 `attachments_pending`), audited
  as an update that failed; the files of hidden items are detached (`RemoveAttachment`, one per file), the response
  completed, and `binary_discard` enqueued in the transaction, only when something was detached.
- `validate.go`: no change of behaviour; `fitsOne`'s comment now says why an attachment item has no answer in
  `PUT /answers` (422 `answers_invalid` with its `linkId`, as since the questionnaire feature's D1).
- `errors.go` `ErrAttachmentsPending`; `handler_http.go` maps it to 409 `attachments_pending`.
- `handler_fhir.go`: each file is a `valueAttachment` (`contentType`, `url`, `title`, `hash` base64 SHA-1, `size`)
  under its item, in upload order, beside the typed answers; `url` is exactly `<api base>/binaries/<id>/content`, the
  api base being the FHIR base without `/fhir/R4` (as `config.Config.APIBaseURL`), so 1.6's and 1.7's id parsing
  holds. Only a Full view renders items: the patient's own (pending files included) and the practitioner's of a
  completed response, which by the submit rule holds no pending file.
- `internal/api/app.go`: one line, `Jobs: questionnaireresponse.NewRiverJobs(p.jobs)` (unavoidable: the api's
  `Withdraw` and `Submit` need the port). `internal/worker` builds the service without `Jobs`; nothing there calls
  `Withdraw` or `Submit` (not touched, as the wave rules ask).

Failing first (each seen before the code existed):
- Every new test failed to build first (`Response has no field Attachments`, `unknown field Jobs`,
  `undefined: questionnaireresponse.ErrAttachmentsPending`, `undefined: NewRiverJobs`). With the declarations and a
  no-op `EnqueueDiscard`: `TestPostgres_ReadsTheAttachmentsAndStopClearsThem` — `ByID() attachments = []` and
  `Stop() of a response with files: ERROR: new row for relation "questionnaire_response" violates check constraint
  "questionnaire_response_withdrawn_attachments" (SQLSTATE 23514)` (2.1's 500, reproduced);
  `TestRiverJobs_EnqueuesTheDiscardInTheTransactionWithOnlyIDs` — `no rows in result set`;
  `TestSubmit_ARequiredAttachmentItemNeedsAFile` — the submit with a clean file `answers invalid`;
  `TestSubmit_IsRefusedWhileAFileIsPending` (both cases), `TestSubmit_DetachesTheFilesOfHiddenItemsAndEnqueuesTheDiscard`,
  `TestSubmit_RollsBackTheDetachWhenTheDiscardFails` — `answers invalid`; `TestWithdraw_ClearsTheFilesAndEnqueuesTheDiscard`
  — `enqueued [], want [...]` and the failing enqueue not failing the withdrawal; `TestRead_ANameViewCarriesNoFiles` — the
  practitioner's and the admin's Name views with 2 files; `TestHTTPHandler_MapsEachDomainErrorToItsStatusAndCode` — 500
  for `ErrAttachmentsPending`; `TestHTTPHandler_RefusesAnAttachmentAnswerAndASubmitWhileAFileIsPending` — the submit 422;
  the two FHIR golden tests — no golden file.
- Passed on their first run, because the behaviour predates this package: `TestPostgres_DeletingAResponseDeletesItsFilesRows`
  (the foreign key's `ON DELETE CASCADE` from 0.1, through the cancellation's `DeleteOfAppointment` and the purger on
  `salus_retention`'s pool) and the `PUT /answers` half of the HTTP test (an attachment item has had no valid answer
  since the questionnaire feature, `answerCases` "a reserved attachment item"). Kept as the plan's tests of them.

Tests:
- Service (fakes; the fake transaction now also rolls back the list of files and the jobs): a required attachment item
  without a file named, with a clean file submitted with it and no job; a pending file on a shown or a hidden item
  refuses the submit with nothing changed and one failed update event; the files of a hidden item (required, no
  file needed) detached, the shown item's kept, one `binary_discard` with the two ids, one success event; a failing
  enqueue rolls the submit back (files listed, in progress, no job, no event); a withdrawal clears the files and
  enqueues the job with the two ids, a refused one keeps them and enqueues nothing, a failing enqueue rolls it back;
  Read and Search give the patient the files and the practitioner's and admin's Name views none.
- Postgres (`NewAppDatabase`, `NewRetentionDatabase`): `ByID` reads the files; `Stop` clears them (the check holds) and
  leaves the `binary` rows to the job; the cancellation's discard and the purger delete the response's `binary` rows
  through the foreign key and keep another response's. River (`NewAppDatabase`): rolled back, no job; committed, one
  `binary_discard` whose args are exactly `{"organization_id", "questionnaire_response_id"}`; outside a
  transaction `ErrNoTx`.
- HTTP: `attachments_pending` 409 on every route's mapping; through the handler on the service, `PUT /answers` with an
  answer to the attachment item 422 `{"error":"answers_invalid","linkIds":["f"]}`, the submit with a pending file 409,
  the log holding no title.
- FHIR golden files (new, written with `-update`, reviewed; no existing golden file changed):
  `response_attachments_in_progress.json` (the patient: a text answer, a pending PDF and a clean photo under the item in
  a section) and `response_attachments_completed.json` (the practitioner: a photo and a PDF on one item, a video on
  another); a Name view renders no file.

Checks (all `--no-deps`, as the wave rules give them):
- tests `go test -count=1 -cover ./internal/questionnaireresponse/... ./internal/api/...`: pass;
  `internal/questionnaireresponse` 93.6%, `internal/api` 91.8%, `internal/api/router` 100%.
- lint (`scripts/lint.sh`, the main git dir read-only): 0 issues both modules, check-migrations ok (one fieldalignment
  finding in a new test struct fixed first).
- archcheck: pass.
- fhir-validate: pass, 77 files, 0 errors; the two new files carry warnings only: the validator cannot fetch the
  synthetic `https://api.example.test/binaries/…/content` URLs and does not know the synthetic Questionnaire canonical,
  as for the other golden responses.
- Not run (wave rules): `make test`, `make cover`, `make e2e`; they run in the wave's integration step.

### Decided while the owner was away

- 2026-11-07, session (3.2): a pending file refuses the submit wherever it is, on a hidden item too. SPEC §2 "Submit"
  says "any attached file"; the scan takes seconds, and the patient never has to act on a file they no longer see.
  The 422 of the answers comes before the 409, so the patient fixes their answers first.
- 2026-11-07, session (3.2): `Withdraw` enqueues `binary_discard` every time, files listed or not: a rejected file's
  row (detached, kept 24 h with its title) then goes at once, and a duplicate discard does nothing. Withdrawals are
  rare; a submit enqueues only when it detached a file (submits are many).
- 2026-11-07, session (3.2): the files are a field of the one model, `Response.Attachments`, read with every row,
  instead of a second read through `Repository.Attachments`: the submit, the withdrawal and the FHIR handler all need
  them with the response, and a Name view drops them like the answers.
- 2026-11-07, session (3.2): the FHIR handler takes the api base from the FHIR base it is given (`/fhir/R4` trimmed,
  the rule of `config.Config.APIBaseURL`), so `NewHandler` and the rest of `app.go` stay as they are.
- 2026-11-07, session (3.2): `valueAttachment` keys are written `contentType`, `url`, `title`, `hash`, `size` (the
  struct's order, which the linter's field alignment decides); JSON property order carries no meaning in FHIR.
- 2026-11-07, session (3.2): `binary_discard` goes on River's default queue (no `InsertOpts`): SPEC §2 names a queue
  only for the scan; 3.1 registers its worker.

Copy added by the session, not yet reviewed: none.

For later packages:
- 3.1: `binary_discard` jobs are enqueued from this package on (withdrawals, submits that hide files), on the default
  queue, args `{"questionnaire_response_id", "organization_id"}`; after a withdrawal `Holder.Attached` lists none.
- 4.1: the journeys' withdrawal and hidden-item submit now enqueue the discard; `valueAttachment.url` is
  `<FHIR base without /fhir/R4>/binaries/<id>/content`.

Dev stack: nothing changed. The tests ran on throwaway databases (`pgtest`); no service was rebuilt or restarted,
the dev database and the bucket were not touched.

## 4.1 End-to-end and privacy

Status: done, 2026-11-07. The acceptance walk (the builder, the patient app on a device or emulator, the
practitioner's viewer and player at desktop width and 390 px, the PDF viewer under its CSP in Chrome, O23) is not in
this package: the orchestrator runs the owner's acceptance walk after the PR, and it is recorded in
`docs/attachments/WALK.md` then.

Built:
- `test/e2e/attachments_test.go` (build tag `e2e`, synthetic data fresh per run):
  `TestAttachments_ThePatientAttachesFilesAndThePractitionerOpensThem`: the practitioner publishes a questionnaire
  with a photo-and-document item (up to 3 files) and a video item and makes it the role's default; a patient books;
  uploads a generated JPEG, a generated 50 MB MP4 and a generated PDF carrying the EICAR string, raw with
  `Content-Disposition` `filename*`, as the app does; the first two become `clean` and the PDF `rejected` (each waited
  for by polling `GET /binaries/{id}`, as the app does); the rejected file's download is 404 and the patient's
  response no longer lists it; the patient submits; the practitioner's FHIR search holds both `valueAttachment`s
  (`contentType`, `title`, `size`, the base64 SHA-1 `hash`, an absolute `url` ending in `/binaries/<id>/content`); the
  practitioner downloads the photo whole (bytes and the headers `Content-Type`, `Accept-Ranges`, `Cache-Control`,
  `X-Content-Type-Options`, `Cross-Origin-Resource-Policy`, `ETag` the SHA-1) and the video by three ranges (start,
  middle, end) whose bytes and `Content-Range` match the original; the audit (as the owner) holds, per file on
  `Binary/<id>`, one upload by the patient, one read per status or content request the patient made, one scan result
  by `binary_scan` (`U` clean, `D` rejected) and one read per practitioner download (1 for the photo, 3 for the video
  ranges). Then it runs the web journey. `TestAttachments_ACancelledAppointmentTakesItsFilesWithIt`: a file uploaded
  to the questionnaire of an appointment more than 24 h ahead, clean and downloadable (200); the patient cancels;
  once the `questionnaire_responses_discard` job has deleted the response (waited for), the `"binary"` row is already
  gone with it (no further wait: the foreign key) and the status and the download are 404.
- `test/e2e/attachments_media_test.go`: the generated files. The JPEG (64x64, `image/jpeg`), the EICAR PDF (the
  string assembled in the test, uncompressed in a content stream, as 3.1's tests) and a real H.264 MP4 written in the
  test: 128x96, 10 frames a second for 6 s, every frame an IDR picture of I_PCM macroblocks (Baseline, CAVLC; the
  pixels uncoded, so about 200 lines of Go without a codec), a bright bar that moves; `ftyp` `isom`, a `moov` with one
  `avc1` track and its `avcC`, the `mdat`, then a top-level `free` box of a counter up to 50,000,000 bytes, so a
  misplaced range shows. It passes `pkg/filetype`'s checker (an `avc1` track, no audio: O18 allows that) and plays and
  seeks in Firefox and WebKit (checked before it was used).
- `test/e2e/web/attachments.spec.ts`: the practitioner signs in, opens the appointment, "View answers" (both items
  "1 file", the rejected document detached), the photo in the viewer ("File 1 of 1", the `<img>` labelled "Photo:
  [Title]" decoded at 64 px), Close; the video in the viewer's player: its duration is the file's 6 s, it plays
  (`currentTime` past 0.5 s), a seek to 4 s lands there and playback moves on past 4.5 s, and the API answered the
  browser's content requests with at least one `206`.
- `docs/privacy/dpia.md`: the section "Questionnaire attachments (`docs/attachments`)", one subsection each for 0.1,
  1.7, 2.1, 3.1 and 3.2 from this file (3.1's: the ClamAV signature is logged next to the binary id only and not
  audited, O25), and one for 4.1.
- `docs/privacy/processors.md` (SPEC §5): no processor added while the store and ClamAV are self-hosted; the hosting
  provider's pending entry names the object store for the files and the KMS for `BINARY_KEK`; "Not processors" gains
  the compose store and ClamAV (its signature download sends no personal data).
- Folded in (owner): `.github/workflows/mobile.yml` job `apk`, `flutter build apk --debug` in `mobile/` on every
  change the workflow's filter catches (every path under `mobile/` among them), a separate job beside `mobile`.

Failing first (each seen failing before it passed):
- Both journeys with clamd stopped (the scan, the piece the journeys depend on, missing; the worker running):
  "the binary_scan job makes the file clean: not within 30s" in each. (A first try with the worker stopped failed
  earlier, at the slots, so it proved nothing about files.) clamd started again, healthy.
- With clamd back, the first journey failed on the test's own count: "the audit of the image/jpeg file: … 1 patient
  reads …; want … 2" (the practitioner's downloads were counted as the patient's reads); fixed in the test, the
  product unchanged.
- The web spec run in Chromium instead of Firefox: the photo step passed, the video step failed, "Expected: 6,
  Received: NaN" for the duration (Playwright's Chromium cannot open H.264, below). Back on Firefox it passes.
- The CI job cannot run locally; it is proven on the PR.

Checks (`checks.sh`, every command with `--no-deps`; the e2e's `up -d --build --wait` without `--remove-orphans`):
- lint (gofmt, `go vet`, golangci-lint both modules, check-migrations): pass; also golangci-lint with `--build-tags
  e2e` over `test/e2e`: 0 issues, and `go vet -tags e2e`.
- archcheck: pass. test (`go test -count=1`, both modules): pass. cover: pass, total 94.3%, every package at 80% or
  more.
- fhir-validate: pass. gen-clients: no diff. Web: lint, typecheck, test, build pass. Mobile: analyze, test pass.
- e2e twice in a row: pass both (39 s and 37 s for the whole suite); the LiveKit log check holds none of the run's
  synthetic values.
- The `apk` job: YAML parsed and read against `mobile`'s job; the actions pinned by commit like the others:
  `actions/setup-java` v6.1.0 (`b843ebf9…`, 2026-10-14; v6.2.0 was 3 days old) with Temurin 21, the JDK 1.7 built
  with; `subosito/flutter-action` v2.23.0 (`1a449444…`, 2026-03-25) with Flutter 3.47.7, the version of
  `build/package/flutter.Dockerfile`. Not run here; it will be proven on the PR.

Readings: the whole first journey takes about 10 s (three uploads, the 50 MB scan included, the ranges and the
browser); the second about 2 s.

### Decided while the owner was away

- 2026-11-07, session (4.1): the web spec runs in Playwright's Firefox, not its Chromium. Playwright's Chromium is
  built without the proprietary codecs (`canPlayType('video/mp4; codecs="avc1…"')` is empty in the image), and the
  SPEC allows H.264 MP4 only (O1), so no allowed video can play in it; Firefox decodes H.264 with the image's
  libavcodec, so "plays and seeks" is a real check. Installing Google Chrome into the Playwright image would have
  changed `build/package/playwright.Dockerfile` (not this package's file) and pulled an unpinned browser. Chrome
  itself, and its PDF viewer under the CSP (O23), are covered by the owner's acceptance walk after the PR. No SPEC
  row: the SPEC names no test browser.
- 2026-11-07, session (4.1): the generated video is a real, decodable H.264 stream written by the test (I_PCM
  frames), not 2.1's skeleton with zeros for media: a browser has to decode it for `currentTime` to move, and the
  repository holds no video file (SPEC §5 "Fixtures are synthetic … generated"). Its 50 MB come from a top-level
  `free` box after the media, so the three ranges still read different parts of a 50 MB object.
- 2026-11-07, session (4.1): the debug APK job uses the runner's Android SDK with `actions/setup-java` and
  `subosito/flutter-action` (pinned by commit), not "the pinned Flutter image" of the plan: that image is built with
  `precache --no-android` and has no Android SDK (1.7), and adding one to it would grow every `make mobile-*` run.
  Gradle installs the platform 37, NDK and CMake the build asks for, as it did in 1.7's container. The APK is not
  uploaded as an artifact: nothing installs it. Only the job's steps differ from the rest of the workflow, which
  keeps running through make.
- 2026-11-07, session (4.1): "an event per request" is checked per file as counts: the test counts its own status
  and content requests and the audit must hold exactly that many reads by the patient, and exactly 1 and 3 by the
  practitioner for the photo and the video. The rejected file's refused download counts as a read (it is audited,
  2.1).
- 2026-11-07, session (4.1): the DPIA also gets a short 4.1 subsection (no new processing, what the journeys prove).

Copy added by the session, not yet reviewed: none.

Dev stack: the e2e brought the stack up from this worktree (`up -d --build --wait`; no Go code changed in this package, so
the services run the wave 3 code); clamd was stopped for the failing run and started again (healthy), the worker was stopped and started for
the first try. The journeys' synthetic clinics, patients, appointments, responses and files stay on the dev database
and their encrypted objects in the bucket, as every e2e run's data does (the cancelled appointment's file row is
gone; its object goes with the purger after 24 h). salus-demo-* containers untouched.

## Review of PR #25

- 2026-11-07, Copilot (its note says it could not complete the review at the selected tier; two findings):
  - `pkg/filetype/limit.go:24`, "an int64 slice index does not compile": not a defect. The Go spec allows an index
    of any integer type (only a constant index must be representable as `int`); the package builds and its tests
    pass locally and in CI. Answered on the PR, no change.
  - `web/src/questionnaire.ts`, a blank "Number of files" field sent `maxFiles: NaN`, which `JSON.stringify` writes
    as `null`, so the API refused the whole body (400) instead of naming the item (422 `invalid_questionnaire`,
    W6). Fixed test first: `leaves maxFiles out when the field is blank` failed (`expected … to not have property
    "maxFiles"`), then `toJSONItem` sends `maxFiles` only when it is a finite number. Web checks: lint, typecheck,
    433 tests and build pass.
- 2026-11-07, CI: the first `e2e` run on the PR failed in the existing `TestQuestionnaire_TheBuilderPublishesAQuestionnaire`
  (the second palette item dragged in by the mouse did not land; the list had A and C only). It passed on every local
  run and on the CI re-run without any change, so it is recorded as a flaky drag timing in that journey, not caused
  by this feature. Fixed the same day (CLAUDE.md "TDD"): each drag-in now waits for the card before it to be closed
  and checks that its row was added before filling it in, so the drop place is measured on a settled list; the
  journey asserts more than before, nothing less. `make e2e` passed twice in a row after the change.
- 2026-11-07, independent review of PR #25, server findings. Each fixed test first; every new test was run and seen
  failing for the stated reason before the change:
  - Interrupted upload left no AuditEvent (real: net/http cancels the request's context when the connection fails
    mid-body, so the refusal's `WithTx` and the commit ran on a cancelled context and logged a 500). `Upload` now
    runs everything after the body (the refusal's audit, the commit, the deletion of a left-over object) on
    `context.WithTimeout(context.WithoutCancel(ctx), 30s)` (`afterBody`). The fake auditor now fails on a cancelled
    context as a pgx query does. Seen failing: `TestUpload_AClientThatGoesAwayMidBodyIsStillAudited`
    (`Upload() = binary: upload: context canceled, want ErrShortBody`) and
    `TestUpload_AClientThatGoesAwayOnceTheFileIsStoredStillGetsItCommitted` (`context canceled; want created`). A
    client gone after its last byte keeps its file; its retry gets the identical file back (200).
  - HEAD on `GET /binaries/{id}/content` (Go's mux matches HEAD on a GET route) opened, decrypted and audited the
    whole file to send none of it. Decision: answer HEAD with 405 `method_not_allowed` and `Allow: GET` before the
    caller or the file is looked at; simplest, nothing to audit, and no client uses HEAD (not in `api/openapi.yaml`,
    so no contract change). Seen failing: `TestHTTPHandler_RefusesHEADOnTheContent` (`HEAD = 200`, the file opened).
    `GET /binaries/{id}` (the status JSON) still answers HEAD as a GET, audited as a read; left as it is.
  - `Delete` removed the object under the URL's id; Postgres finds the row by an uppercase UUID, the object key is the
    lowercase canonical id, so the object stayed until the purger. Now the row's `b.ID`. The fake repository reads an
    id in any case, as Postgres does. Seen failing: `TestDelete_RemovesTheObjectOfTheCanonicalID` (`GetRange = <nil>,
    want ErrObjectNotFound`).
  - An untitled file rendered `"title":""` in the FHIR `valueAttachment` (R4 forbids empty strings): `title` is now
    `omitempty`. Seen failing: `TestHandler_ReadLeavesOutTheTitleOfAnUntitledFile` (`want no title`); new golden
    `response_attachment_untitled.json`, written with `-update` for that test only; `fhir-validate` passes (0 errors).
  - A file its response no longer lists (an answer of a hidden item dropped on submit, or a withdrawal) stayed
    readable until `binary_discard` ran. `Status` and `Download` now check `Holder.Attached(responseID)` in the read's
    transaction and answer `ErrNotFound` (404), audited like any not-found read: no entity, the caller's org. Seen
    failing: `TestStatusAndDownload_RefuseAFileTheResponseNoLongerLists` (Status `<nil>`, Download `scanning` / `<nil>`,
    each audited as a read of the file).
  - Recorded, no change: a `Range` past the end (416) is audited as a read of the file, because `Download` audits and
    commits before the handler parses the range. Over-auditing a refused read is the safe side; moving the range
    check into the service is not worth the change.
  - Checks: `go test -cover` of `internal/binary/...` (90.6%), `internal/questionnaireresponse/...` (93.6%),
    `internal/api/...` (91.8%, router 100%) pass; `scripts/lint.sh` (gofmt, vet, golangci-lint, check-migrations)
    0 issues; archcheck passes; `fhir-validate` 0 errors. Run with `--no-deps` on the shared stack; nothing left on
    it.
- 2026-11-07, independent review, the mobile findings and the web seek test (one commit; each fix test first, the
  failing test seen before the change):
  - `mobile/lib/files.dart`: the picked copy was deleted only when it lay under `Directory.systemTemp`, while on
    Android image_picker and file_picker copy into the cache directory (`<data>/cache`, `<data>/cache/<uuid>/`,
    `<data>/cache/file_picker/<stamp>/`, read in the plugins' sources) and `Directory.systemTemp` is often
    `<data>/code_cache` beside it, so the original with its EXIF and GPS stayed. New `mobile/lib/copies.dart`:
    a copy is deleted when its upload ends (sent, removed after failing, refused, or the screen left) wherever the
    picker put it among the app's own directories, the temporary directory and the cache directory beside it
    (`cache` on Android, `Library/Caches` on iOS), with the folders the picker made for it once empty; a path outside
    them (the gallery, the patient's documents, a `..` path) is never touched. `DeviceFiles` sweeps what an earlier
    run left at start (`salus-*`, `image_picker*`, `scaled_*`, `file_picker/`, the UUID-named files and folders) and
    a pick waits for the sweep. path_provider is only a transitive dependency, not one of the app's, so the
    directories are derived from `Directory.systemTemp`'s parent (decided, no new dependency). The intermediates
    (`salus-upload-*`, `salus-converted-*`) were already deleted by `prepareFile`'s discard wherever they lie.
    `test/copies_test.dart` (14 tests on the Android and iOS layouts in a throwaway directory) failed first (no
    `copies.dart`); the wiring in `DeviceFiles` reaches the plugins and is not unit-tested, as before.
  - `mobile/lib/booking.dart`: the streamed upload had no timeout, so a stalled socket showed "Uploading… n%"
    forever. An upload that goes 60 s (`uploadIdle`) without sending a byte or hearing its answer is aborted and
    fails as unreachable, which shows M5 "Not uploaded. Check your connection." with Try again and Remove (no new
    copy). Two widget tests, a body that stalls after its first chunk (still uploading at 59 s, M5 at 60 s, Try again
    sends it whole and it is added) and an answer that never comes, failed first (M5 not found).
  - `mobile/lib/metadata.dart`: the client read `count` stsd entries and ignored a mismatch, while the server takes
    the payload only when it is exactly `count` boxes (`pkg/filetype/mp4.go` `checkTrack`). The client now counts
    every entry in the payload and judges a mismatch as not playing in Chrome, so the file is converted instead of
    uploaded and refused. Two cases in `file_prep_test.dart` (more counted than held, more held than counted) failed
    first (`Expected: <false> Actual: <true>`).
  - `mobile/lib/uploads.dart`: the local `Attached.contentType` was `image/jpeg` for every photo, a PNG too; it is
    now the type the server detected (the upload's `Binary`, or the response's attachment). `test/uploads_test.dart`
    failed first for PNG (`Expected: 'image/png' Actual: 'image/jpeg'`).
  - `test/e2e/web/attachments.spec.ts`: the Range assertion counted any 206, which Firefox's first `bytes=0-`
    request already gives. It now records the start of each 206 Range request for the video and asserts, after the
    seek, that one started past 0. For that to be deterministic the generated video (`test/e2e/attachments_media_test.go`)
    now puts its 50 MB filler between `moov` and `mdat`, so the frames lie past it and the player has to ask for them
    with a ranged request (a seek in time within a small, already-buffered `mdat` would make no new request, and
    read-ahead on localhost would make a later-chunk layout flaky). The Go range checks and the audit counts are
    unchanged. Not run here (the stack is shared): the spec lists with Playwright (`--list`) and the e2e package
    passes `gofmt` and `go vet -tags e2e`; the orchestrator runs `make e2e` after merging.
  - Not fixed here: iPhone MOV timed-metadata tracks need a real iPhone (owner).
  - Checks: `make mobile-analyze` no issues; `make mobile-test` 314 tests pass; `dart format` clean. Not run: `make
    e2e` (shared stack, see above).
