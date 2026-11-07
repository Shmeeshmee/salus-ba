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
