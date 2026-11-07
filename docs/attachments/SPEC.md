# Questionnaire attachments

Status: approved, 2026-11-07. The settled decisions (§2) are the owner's; the plan's own decisions in §2 (the
`Holder` port, the separate `attachments` column, `DELETE /binaries/{id}`, the `keep` → `delete` retention rule,
the route deadlines) and every default of §3 (O1 to O12) were approved by the owner the same day, with the EXIF
answer (§2 "Mobile", O13). `COPY.md` is approved as written. Rows of §3 marked "decided by the session, owner to
review" were added while the owner was away.
Branch `feat/attachments` from `main`, one commit per package (`attachments(<n>): <summary>`), packages of a wave
built in parallel worktrees and merged back in the wave's integration step, one PR at the end.

## 1. Goal

Patients attach photos, documents and short videos to questionnaire answers in the mobile app, and the
appointment's practitioner views them in the web app: a photo or a PDF in a viewer, a video in a player that
seeks. This enables the `attachment` item type and the `valueAttachment` answer that the questionnaire feature
reserved (`docs/questionnaire/SPEC.md` §2 "Item model", D1). A practitioner adds a "File upload" item to a
questionnaire in the builder, chooses which kinds of file it accepts and how many; the patient picks or records
the files, the app uploads them one by one, and the server checks each file's type from its bytes, encrypts it
with its own key, stores it in an S3-compatible object store and has it scanned for malware before anyone can
open it. Files are health data: the questionnaire response's access rule and consent decide who uploads and who
reads, every upload, download, scan result and deletion is audited, refusals included, and the files live and
die with the response they belong to. Files are FHIR R4 `Binary` resources, owned by a new domain `binary`; the
answer that references one is a `valueAttachment`.

## 2. Settled decisions

Given by the owner on 2026-11-07. Do not reopen them.

| Topic | Decision |
| --- | --- |
| What | Patients attach photos, documents and short videos to questionnaire answers in the mobile app. Practitioners view them in the web app. The reserved `attachment` item type and `valueAttachment` answers are enabled |
| Where the bytes live | An S3-compatible object store: a pinned container in compose locally, the hosting provider's EU object store in production (not chosen yet, §8) |
| The local store | MinIO no longer publishes free Docker images (October 2025, maintenance mode). ADR 0010 picks a maintained S3-compatible store for compose, comparing at least SeaweedFS and Garage, pinned, and the Go S3 client, justified against CLAUDE.md's dependency rule |
| Domain | A new domain `binary` (FHIR `Binary`) in the context `clinical`. It owns the object store, the `binary` table and the files' lifecycle. It is added to CLAUDE.md, `docs/architecture/archcheck/config.yaml` and `ownership.yaml` in the package that creates it (0.1) |
| Copy | Plain and professional (CLAUDE.md "Copy"). `COPY.md` holds the text, word for word, once the owner approves it; a change goes back to the owner first |

Decisions taken in this plan (reopen only with the owner):

- **Dependency direction.** `questionnaireresponse` → `binary`, never the other way. `binary` is the lower
  domain, like `retention`: it declares the port it needs from the file's owner, `binary.Holder`, in its own
  types, and `questionnaireresponse` implements it (`service_attachments.go`), as `auth` implements
  `retention.Purger`. `binary` imports no other domain but `auditevent`. `internal/api/app.go` builds
  `questionnaireresponse.Service` first and passes it to `binary.NewService` as the holder, so there is no
  construction cycle. `questionnaireresponse` never calls `binary.Service`: what it needs done to files (after a
  withdrawal, or when hidden items' answers are removed on submit) it enqueues as a `binary_discard` job in its
  own transaction, with the response id only.
- **`binary.Holder`** (implemented by `questionnaireresponse.Service`; unaudited checks, because `binary` audits
  the call they serve):
  - `UploadTarget(ctx, caller, responseID, linkID) (Target, error)`: the patient's own `in-progress` response,
    active consent with the clinic, appointment `booked` and before `due_at`, and `linkID` an `attachment` item
    of the response's version. Returns the clinic, the patient, the appointment, the item's accepted kinds and
    its maximum number of files. Refusals use the response's existing codes (404, 409 `submitted`, `withdrawn`,
    `past_due`, `appointment_closed`, `consent_required`) and 422 `not_attachment_item`.
  - `Editable(ctx, caller, responseID) error`: the same checks without the item, for a deletion by the patient.
  - `Readable(ctx, caller, responseID) (Grant, error)`: `access.QuestionnaireResponse` at `Full` (the patient
    on their own response; the appointment's practitioner on a `completed` one, D9 included); anything else
    404. The grant carries the purpose of use (`PATRQT` for the patient, `TREAT` for staff).
  - `Attach(ctx, responseID, linkID, Attachment)`, `Detach(ctx, responseID, binaryID)`,
    `Cleared(ctx, responseID, binaryID)` and `Attached(ctx, responseID) ([]string, error)`: keep the
    response's list of attachments in step with the files, inside `binary`'s transaction.
- **Where an attachment answer lives.** `questionnaire_response` gains a column `attachments` (`jsonb`, default
  `[]`): `{linkId, binaryId, contentType, size, title, hash, pending}` per file, written only through the four
  holder methods above. It is not part of `answers`, and changing it does not bump `version_id`: `version_id`
  (the `If-Match` of `PUT /answers`) guards the patient's typed answers, so an upload that finishes while the
  patient types never makes their next save stale. A change to `attachments` updates `updated_at`
  (`meta.lastUpdated`), not `meta.versionId`; the FHIR API has no `vread` or history, so `meta.versionId` is the
  lock token and nothing else. `PUT /answers`
  refuses an answer to an `attachment` item (422 `answers_invalid` with its `linkId`); files are added and
  removed only through the binary endpoints.
- **Tables.** One new, one column added:
  - `binary` (owner `binary`, `personal_data: true`): `id` (the `Binary` id and the object key), `organization_id`,
    `patient_id`, `appointment_id`, `questionnaire_response_id` (`REFERENCES questionnaire_response (id) ON DELETE
    CASCADE`), `link_id`, `content_type` (`application/pdf`, `image/jpeg`, `image/png`, `video/mp4`), `size`
    (plaintext bytes), `title`, `sha1` (20 bytes), `status` (`quarantined` | `clean` | `rejected`), `kek_id`,
    `wrapped_key`, `segment_size`, `scanned_at`, `created_at`, `updated_at`. The clinic, patient and appointment
    are copied once from the `Target`, for row-level security and the audit, as `questionnaire_response` copies
    them from the appointment. RLS: the patient scope sees and deletes its own rows and inserts rows for itself;
    the clinic scope sees the rows of its clinic whose `questionnaire_response` it can see (an `EXISTS` on that
    table, which applies that table's own policies: consent and D9 follow without being written twice); the job
    scope (`app.questionnaire_response_id` with `app.organization_id`, set by `binary`'s jobs) sees, updates and
    deletes the rows of that one response. Grants to `salus_app`: `SELECT, INSERT, DELETE`, `UPDATE` of `status`,
    `scanned_at`, `wrapped_key`, `updated_at` only. `SELECT, DELETE` to `salus_retention` (the purger,
    §5).
  - `questionnaire_response.attachments` (owner `questionnaireresponse`): the column above, `UPDATE` granted to
    `salus_app`; a new policy lets the job scope (`app.questionnaire_response_id` with `app.organization_id`)
    read and update that one row, for the scan and discard jobs' holder calls.
  - Deleting a response, by any path (the cancellation's discard job, the retention purger as
    `salus_retention`, a later erasure), deletes its `binary` rows in the same statement, through the foreign key.
    The wrapped data key goes with the row, so the stored ciphertext can no longer be read by anyone from that
    moment (crypto-shredding); the purger removes the object itself later (§5). Package 0.1 tests that the
    cascade works for `salus_retention`, which holds no privilege of its own on the rows it cascades to.
- **Item model.** An `attachment` item has `text`, `required`, help text and a condition like any question, and
  two settings of its own: `accept`, a non-empty set of `photo` (JPEG, PNG), `document` (PDF) and `video`
  (MP4), and `maxFiles`, 1 to 10. In FHIR: `type: attachment`, `repeats: true` when `maxFiles` is more than 1,
  one R4 core extension `http://hl7.org/fhir/StructureDefinition/mimeType` (`valueCode`) per accepted
  content type, and the R4 core extension `http://hl7.org/fhir/StructureDefinition/questionnaire-maxOccurs`
  (`valueInteger`) with `maxFiles`. No `maxSize` extension: the size limits depend on the kind (§3 O2) and are
  the server's. An `attachment` item may not be the `question` of another item's condition (it is neither
  boolean nor choice, as today). The builder offers it as "File upload" (COPY W1).
- **Upload pipeline** (`POST /questionnaire-responses/{id}/items/{linkId}/attachments`, raw body, served by
  `binary`): (1) `Holder.UploadTarget`; `Content-Length` required (411 `length_required`), at most the largest
  limit among the item's accepted kinds and the response's remaining budget (413 `too_large`,
  `response_full`); the item not yet at `maxFiles` (409 `too_many_files`); (2) read the first 4 KiB and detect
  the type from the bytes (`pkg/filetype`), not from `Content-Type` or the file name; a type the item does not
  accept is 415 `unsupported_type`, and a declared length past the detected kind's limit 413 `too_large`;
  (3) stream the rest through a `filetype` limit reader (exactly `Content-Length` bytes), the MP4 checker (the
  codecs, §3 O1) and SHA-1, into the envelope encrypter (`pkg/envelope`), into one `PutObject` of known length
  (the ciphertext length follows from the plaintext's), outside any transaction; (4) one transaction: an
  advisory lock on the response, the item's count and the response's bytes checked again, the `binary` row
  `quarantined`, `Holder.Attach` with `pending: true`, the `binary_scan` job (`InsertTx`, the binary id only), the
  `AuditEvent`. A failure after the object is stored deletes it (best effort; the purger removes what is left,
  §5). Memory per upload is constant: one segment buffer and the S3 client's, whatever the file's size. The
  title comes from the request's `Content-Disposition` `filename*` (RFC 6266), cut to 200 characters, the path
  stripped; never from the query string, which access logs may hold.
- **Retried uploads.** The app retries a failed upload with the whole file (§3 O3). When the response already
  has a file on the same item with the same SHA-1 and size, the upload returns that file (200) and deletes the
  new object, so an upload that succeeded but whose answer was lost does not attach the file twice.
- **Download** (`GET /binaries/{id}/content`): one transaction reads the row (RLS), calls
  `Holder.Readable`, requires `clean` (409 `scanning` while `quarantined`; 404 when `rejected` or gone) and
  writes the `AuditEvent`; it commits before a byte is sent. Then the api maps the requested plaintext range to
  whole ciphertext segments, fetches them with one ranged `GetObject`, decrypts and writes the slice: 200 with
  the whole file, 206 for one `Range` (`bytes=a-b`, `a-`, `-n`), 416 outside the file; a request with several
  ranges gets the whole file (200). Headers: the detected `Content-Type`, `Content-Length`, `Accept-Ranges:
  bytes`, a strong `ETag` (the SHA-1), `Cache-Control: private, no-store`, `X-Content-Type-Options: nosniff`,
  `Cross-Origin-Resource-Policy: same-origin`, `Content-Disposition: inline` with the title as `filename*`, and
  `Content-Security-Policy: default-src 'none'` (plus `sandbox` for every type the browser still shows with
  it; package 1.6's walk checks the PDF viewer in Chrome and 2.1 records the header per type). Every request is
  audited, ranges included: a seek in a video is a read of the file.
- **Status and deletion.** `GET /binaries/{id}`: `{id, contentType, size, title, status}`, for whoever may
  read the response at `Full`, so the app can show "Checking the file" until the scan is done. `DELETE
  /binaries/{id}`: the patient, through `Holder.Editable`; one transaction deletes the row and calls
  `Holder.Detach`, audited; the object is deleted after the commit. Deleting a file the patient has submitted is
  refused like any change to a submitted response (409 `submitted`).
- **Scan** (`binary_scan` job, River, `internal/worker`): the job scope of the file's response; it streams the
  object through the envelope decrypter into clamd's `INSTREAM` (`pkg/clamd`); `OK` → `clean`, `scanned_at`,
  `Holder.Cleared`, audited; `FOUND` → `rejected`, the wrapped key cleared, `Holder.Detach`, audited with the
  signature name (a ClamAV name, not personal data), the object deleted after the commit; clamd unreachable or
  any other error → River retries with its backoff, and the file stays `quarantined` (nobody downloads it).
  The worker's scan queue runs 2 jobs at a time (ClamAV's memory, measured in 3.1). The job's log line carries
  the binary id, the result and the timing; never the title or the signature together with a patient id.
- **Discard** (`binary_discard` job): enqueued by `questionnaireresponse` in its transaction when a withdrawal
  clears a response and when a submit removes the answers of hidden attachment items. The worker calls
  `binary.Service.Discard(ctx, agent, org, responseID)`: in the job scope, every file of the response that
  `Holder.Attached` no longer lists is deleted (row, then object after the commit), each audited with the job
  as agent. Repeating it changes nothing.
- **Submit** (`questionnaireresponse`): a required `attachment` item needs at least one file; a submit while
  any attached file is `pending` is refused (409 `attachments_pending`, COPY M10), so a completed response holds
  only clean files and the practitioner never sees a file that is still being checked or was rejected. Files of
  items hidden by a condition are detached on submit and discarded by the job.
- **Encryption.** Envelope encryption (§3 O4): a random 256-bit data key per file; the plaintext in 64 KiB
  segments, each sealed with AES-256-GCM under the data key, the nonce the 96-bit big-endian segment index (safe
  because every key encrypts one file only), the additional data the binary id, the segment index and a
  final-segment flag, so a segment cannot be moved, swapped between files or the file truncated without the open
  failing. Ciphertext segments are 64 KiB + 16 bytes; segment `i` starts at `i × 65,552`, which is what lets a
  range be served without reading the file from the start. The data key is wrapped with AES-256-GCM by the
  key-encryption key, the binary id as additional data, and stored as `wrapped_key` with the key's `kek_id`.
  Locally and in CI the key-encryption key comes from configuration, `BINARY_KEK` (base64, 32 bytes) and
  `BINARY_KEK_ID`, read only in `internal/config`, like `TOTP_KEY`; production takes it from the KMS (§8). The
  object store's own encryption at rest comes on top in production, not instead.
- **Configuration** (`internal/config`): `config.LoadObjectStore()` (`S3_ENDPOINT`, `S3_REGION`, `S3_BUCKET`,
  `S3_ACCESS_KEY_ID`, `S3_SECRET_ACCESS_KEY`, `BINARY_KEK`, `BINARY_KEK_ID`), loaded by the api and the worker
  only; `config.LoadClamd()` (`CLAMD_ADDR`), the worker only. Errors name the variable, never a value. The API's
  public base for `valueAttachment.url` is derived from `FHIR_BASE_URL` (its origin and path without `/fhir/R4`),
  which `config.Load` already requires.
- **Compose.** The object store chosen by ADR 0010 (pinned; default Garage, §3 O10) with a one-shot init service
  that creates the bucket `salus-binaries` and the synthetic local key, and `clamd` (`clamav/clamav`, pinned) with
  `StreamMaxLength`, `MaxFileSize` and `MaxScanSize` at 250 MB (above the 200 MB video limit), a volume for the
  signature database and a health check on `PING`. Both on the compose network only; no host port (the api
  serves the files). The `tools` service gains `TEST_S3_*` and `TEST_CLAMD_ADDR`. ClamAV downloads its
  signatures (freshclam) on its first start; that needs the internet once and takes about a minute, and the
  health check waits for it.
- **Timeouts and body limits.** The api's server `ReadTimeout` (15 s) and the request timeout middleware would
  cut a 200 MB upload on a phone. The upload and content routes set their own read and write deadlines through
  `http.ResponseController` (upload: 30 minutes; content: 10 minutes per request) and are exempt from the
  256 KiB body limit; every other route keeps its limits.
- **JSON API** (`api/openapi.yaml`, `make gen-clients`, all in package 0.1):
  `POST /questionnaire-responses/{id}/items/{linkId}/attachments` (`application/octet-stream`; 201, or 200 for
  a retried upload; `Binary` JSON), `GET /binaries/{id}`, `GET /binaries/{id}/content` (bytes, `Range`),
  `DELETE /binaries/{id}` (204). The questionnaire item schema gains `attachment`, `accept` and `maxFiles`; the
  submit gains 409 `attachments_pending`. Errors keep `{"error": "<code>"}`.
- **FHIR.** No new interaction: files are not served as FHIR `Binary` reads (a later feature may, with
  `Binary.securityContext` the response), and `CapabilityStatement.json` does not change. In a
  `QuestionnaireResponse` an attachment answer is `valueAttachment` with `contentType`, `size`, `title`, `hash`
  (§3 O8) and `url`, the absolute download endpoint `<api base>/binaries/<id>/content`, never the bucket. The
  apps build the URL from the binary id through their own client base, not from `url`. The patient's view lists
  `pending` files too (the app shows their status); a practitioner only ever sees completed responses, whose
  files are all clean.
- **Audit.** Every upload (created, `PATRQT`), status read and download (read; `PATRQT` for the patient,
  `TREAT` for staff), deletion by the patient, scan result (`binary_scan` as agent, `HOPERAT`) and discard
  (`binary_discard` as agent) writes an `AuditEvent` in the same transaction, refusals included, with
  `Binary/<id>` and `QuestionnaireResponse/<id>` as entities (one entity when no file exists yet: a refused
  upload names the response). Codes as in booking SPEC §2 "Audit codes".
- **Web.** The builder: "File upload" in the palette; on the opened card the accepted kinds (checkboxes) and the
  number of files, with the limits in the hint. Appointment detail, answers of a submitted response: a file
  list per item (title, kind, size); a photo or PDF opens in a viewer overlay with previous and next, a video in
  the browser's `<video>` element pointed at the content URL, so the browser makes the `Range` requests and seeks.
  PDFs are shown by the browser's own viewer, no PDF library. No new npm dependency.
- **Image and video metadata** (owner, 2026-11-07). The mobile app strips location and every other metadata
  from photos (EXIF, XMP, IPTC) and videos (the container's metadata atoms: location, device, creation time) before
  upload, and keeps the orientation: a photo is written upright with no orientation tag left to need, or keeps
  only its orientation tag. The server does not alter the bytes it receives.
- **Mobile.** On an `attachment` item: "Add file" opens a sheet with only the sources the item accepts (take a
  photo, record a video, choose from photos, choose a document). Before upload the app converts HEIC and HEIF to
  JPEG and a MOV or a non-H.264/AAC video to H.264/AAC MP4, on the device, so everything plays in Chrome; then
  uploads with progress, one file at a time, retrying the whole file on "Try again"; then polls `GET
  /binaries/{id}` until `clean` or `rejected` (every 2 s, then backing off to 10 s). Permission text is in
  `COPY.md` (iOS purpose strings and the Android rationale). Package 1.7 adds and justifies its dependencies
  (`image_picker`, `file_picker`, and the platform conversion: `AVAssetExportSession` on iOS, Media3
  Transformer on Android), pinned exact, as the questionnaire SPEC justified `@dnd-kit`. No viewer of the
  patient's own files on mobile in this feature: the read-only view lists them.
- **Performance.** Measured, never assumed, numbers in `PROGRESS.md`: `BenchmarkSeal`, `BenchmarkOpen` and
  `BenchmarkOpenRange` (envelope, MB/s, 1.1); the upload of a 200 MB synthetic MP4 through the api handler, with
  the real Postgres and object store (throughput in MB/s and the api's peak heap, sampled with `runtime/metrics`
  while it runs, 2.1); 1,000 random 1 MiB `Range` requests on that file, p50, p95, p99 (2.1); the scan time of
  the 200 MB file and clamd's resident memory (3.1). A reading that misses what a phone needs (an upload slower
  than the network, memory that grows with the file) is recorded and fixed before the package is done.

## 3. Open points

Each is written with its default applied, and the plan builds to the default. O1 to O12 were approved by the
owner on 2026-11-07. Rows marked "decided by the session, owner to review" were decided while the owner was away
(`PROGRESS.md` "Decided while the owner was away" gives the reasons).

| # | Question | Default applied | Built in |
| --- | --- | --- | --- |
| O1 | Which file types? | PDF, JPEG, PNG and MP4 (H.264 video, AAC audio), detected from the bytes. The mobile app converts HEIC/HEIF to JPEG and MOV (and any non-H.264/AAC video) to MP4 before upload, so everything plays in Chrome. No transcoding or thumbnails on the server | 1.2, 1.7, 2.1 |
| O2 | Which limits? | 20 MB per photo or document, 200 MB per video, 10 files per item, 500 MB per response. Video length is not checked on the server | 1.2, 1.5, 2.1 |
| O3 | How do bytes travel? | Upload and download streamed through the api in constant memory. No presigned URLs. No resumable upload: the app retries the whole file, and the server returns the earlier file for an identical retry | 2.1, 1.7 |
| O4 | How are files encrypted? | Envelope encryption: a data key per file, AES-256-GCM in 64 KiB segments so the api serves `Range` requests, the data key wrapped by a key-encryption key from configuration locally (`internal/config` only, like `TOTP_KEY`) and from the KMS in production (§8) | 1.1, 2.1 |
| O5 | How is malware handled? | ClamAV (clamd, pinned) scans each file as a River job. `quarantined` → `clean` or `rejected`. Nobody downloads before `clean`. A rejected file is deleted (key at once, object after the commit) and audited | 1.3, 3.1 |
| O6 | How long are files kept? | Files follow their response: kept with a completed one, deleted with it on cancellation, revocation of a future appointment, the retention purge of an unsubmitted one, and withdrawal. An object with no row (an upload that never reached its answer) and a rejected file's row are removed after 24 h | 0.1, 3.1, 3.2 |
| O7 | What is audited? | Every upload, status read, download (each range request), deletion, scan result and discard writes an `AuditEvent` in the same transaction, refusals included. Access goes through the existing questionnaire-response access rule and consent checks | 2.1, 3.1 |
| O8 | Compute the R4 `Attachment.hash` (SHA-1, base64)? | Yes. It costs one pass that the pipeline makes anyway, R4 names it, it lets a FHIR consumer check a download end to end, and it makes a retried upload idempotent (O3). It is not a security control: GCM authenticates the bytes | 2.1, 3.2 |
| O9 | Submit while a file is still being checked? | Refused, 409 `attachments_pending`; the app shows M10 and enables Submit once every file is clean. So a practitioner never sees a file that is pending or rejected | 3.2, 1.7 |
| O10 | Which local object store and which Go S3 client? | Garage (single binary, small, S3 operations we need), pinned, SeaweedFS the fallback if 0.1's check of the four operations fails on Garage; `github.com/aws/aws-sdk-go-v2/service/s3`, pinned, over a hand-written SigV4 client (well over 50 lines with streaming and signing) and `minio-go` (its maintenance after MinIO's change of course is to be checked in 0.1). ADR 0010 records the comparison | 0.1, 1.4 |
| O11 | Keep the file name? | Yes, as `title`, cut to 200 characters, so the practitioner can tell files apart. It is personal data (a name can say anything), never logged, deleted with the file | 2.1 |
| O12 | Can the patient open their own files on mobile? | Not in this feature: the read-only view lists title, kind and size. They have the original on their device | 1.7 |
| O13 | Image and video metadata? | Decided by the owner, 2026-11-07: the app strips location and all other metadata, keeps orientation (§2 "Image and video metadata") | 1.7 |
| O14 | The init service of §2 "Compose"? | None: Garage's `server --single-node --default-bucket` lays out the one node and creates the bucket `salus-binaries` and the synthetic key from its environment at start, idempotently, with no bind-mounted file (ADR 0010). Decided by the session, owner to review | 0.1 |
| O15 | The table name `binary` is a reserved word in Postgres | Kept as named, quoted `"binary"` in every statement (archcheck and sqlc accept the quotes). Decided by the session, owner to review | 0.1, 2.1, 3.1 |
| O16 | `salus_retention`'s `SELECT, DELETE` on `binary` (§2 "Tables") | Granted with the `delete` rule, not before: a migration test requires the role's grants to be exactly the tables whose rule is `delete`, and the cascade from `questionnaire_response` needs no privilege (tested). Package 3.1 adds a new migration with the grant, with its purger and rule. Decided by the session, owner to review | 0.1, 3.1 |
| O17 | Error codes the §2 "JSON API" does not name | `range_not_satisfiable` (416 on content); `invalid_request` (400) for an upload body that ends before `Content-Length`; `forbidden` (403) for staff on the upload and the deletion, as for the answers; the deletion's 409 are the answers' codes. Decided by the session, owner to review | 0.1, 2.1 |
| O18 | Which MP4 files exactly? (decided by the session, owner to review) | The major brand decides: `isom`, `iso2` to `iso6`, `mp41`, `mp42`, `avc1`, so a HEIF or MOV listing `isom` only as compatible is refused. At least one `avc1` track; every sample entry `avc1` or `mp4a`, so HEVC, Opus, encrypted, timecode and metadata tracks are refused (audio-only is out of scope, §6). `moov` at most 16 MiB. Reason: only what Chrome plays everywhere, nothing the checker cannot vouch for (1.2) | 1.2 |
| O19 | `questionnaire-maxOccurs` on a one-file item? (decided by the session, owner to review) | Written only when `maxFiles` is 2 or more: the R4 extension is allowed only on a repeating item with a value of 2 or more, and `fhir-validate` refused it otherwise. A one-file item has no `repeats`, which already means one; the JSON API and the stored item always carry `maxFiles` (1.5) | 1.5 |
| O20 | Which image metadata stays? (decided by the session, owner to review) | Removed: EXIF (orientation kept), XMP, IPTC, comments, data after the end of the image, PNG text chunks, MP4 `udta`, `meta` and `uuid` boxes and creation times. Kept: the JPEG ICC profile and Adobe APP14 and the PNG colour chunks and `pHYs`, which say how to show the colours and carry no location, device or time. Reason: O13 forbids location and identifying metadata, and dropping colour data would change what the practitioner sees (1.7) | 1.7 |
| O21 | What does the `binary_scan` job carry? (decided by the session, owner to review) | The binary id, its response's id and its clinic's id, all opaque. §2 "Scan" runs the job in its response's job scope (`app.questionnaire_response_id` with `app.organization_id`), which row-level security needs before the job can read the file's row; the binary id alone cannot reach it. No personal data, as `binary_discard`'s ids (§5 "River jobs") | 2.1, 3.1 |
| O22 | Two audit entities in one `AuditEvent`? (decided by the session, owner to review) | `audit_event` holds one entity, so each audited call writes one event per entity, in the same transaction and with the same request id, outcome and purpose: `Binary/<id>` and `QuestionnaireResponse/<id>`; one event without an entity when none is disclosed (a refused upload names the response when the caller sees it). No migration of the audit table | 2.1, 3.1 |
| O23 | The download's `Content-Security-Policy` per type (decided by the session, owner to review) | `default-src 'none'` for `application/pdf`; `default-src 'none'; sandbox` for `image/jpeg`, `image/png` and `video/mp4`. Chrome's PDF viewer refuses to render a sandboxed document, in a tab or in the web app's `<object>`; images and video still show sandboxed. Not walked in Chrome in 2.1 (no browser in the session); 4.1's walk checks the PDF viewer | 2.1 |
| O24 | An identical retry on a full item (decided by the session, owner to review) | Before the bytes are read, `too_many_files` and `response_full` are not answered when the item already holds a file of exactly the declared size: that upload may be the retry of it, whose SHA-1 is known only once read. The commit checks the limits again under the lock, after the identical-retry check, so a different file of the same size is refused there (and its object deleted) | 2.1 |

## 4. Lawful basis

- Attached files are health data (Art. 9), and often more than the patient meant to show (a photo's
  background, a document's other pages). Care: Art. 9(2)(h), with the consent for sharing before care (booking
  SPEC §3), exactly as the questionnaire's answers: the patient attaches them to prepare an appointment, and they
  are shared with the clinic once submitted. Uploading needs the active consent. **Needs legal confirmation
  before production**, with the questionnaire's.
- Purpose limitation: files are used for the patient's care only. No analysis, thumbnails, AI processing,
  reports or training on them without a separate basis, SPEC and DPA.
- Malware scanning is security of processing (Art. 32) and the clinic's legitimate interest in not opening
  malicious files; it runs on our own infrastructure, and the file never leaves it.
- Data minimisation: the mobile app strips location and all other metadata from photos and videos before upload,
  keeping only the orientation (owner, 2026-11-07, §2 "Image and video metadata"); the server does not alter the
  bytes it receives.
- Audit and retention: as in accounts (Art. 5(2), 32).

## 5. Personal data

| Where | Field | Why it is needed |
| --- | --- | --- |
| object store | the encrypted bytes | The file the patient attached: the purpose of the feature. Unreadable without the row's wrapped key |
| `binary` | `questionnaire_response_id`, `link_id` | Which answer the file belongs to, so it is read under that response's access rule and deleted with it |
| `binary` | `organization_id`, `patient_id`, `appointment_id` | Copied once from the response, for row-level security and the audit entities |
| `binary` | `content_type`, `size`, `sha1` | What the viewer and player need, the limits per item and response, the identical-retry check and the R4 `hash` |
| `binary` | `title` | The file's name, so the practitioner can tell files apart (O11) |
| `binary` | `status`, `scanned_at` | Whether the file may be opened, and when the scan said so |
| `binary` | `kek_id`, `wrapped_key`, `segment_size` | To decrypt the file; deleting them makes the bytes unreadable |
| `binary` | `created_at`, `updated_at` | Operations and the 24 h rule for rejected rows |
| `questionnaire_response` | `attachments` | The answer: which files an item holds, with what the FHIR `valueAttachment` shows, and whether each is still pending |
| River jobs | `binary_scan`: binary id, response id and clinic id (§3 O21); `binary_discard`: response id and clinic id | To scan and to discard; opaque ids only |
| AuditEvent | `Binary/<id>`, `QuestionnaireResponse/<id>` | Who uploaded, read, deleted or scanned which file, when, and why |

Logs carry the request id, route, status, duration, opaque ids, error codes and, for the scan, the result:
never a title, a file's bytes or any part of them, a content type together with a patient id, or a signature
name together with a patient id. Fixtures are synthetic: generated images, a generated PDF, a generated MP4 (a
valid `ftyp` and `moov` with `avc1` and `mp4a` entries and padding), never a real photo; the EICAR test string is
assembled in the test, never stored as a file in the repository.

Retention (defaults, **need a legal decision before production**):

| Table | Rule |
| --- | --- |
| `binary` | `delete`, 24 h: a `rejected` row once 24 h have passed since its scan; each row otherwise lives exactly as long as its response, and goes with it (foreign key) or with its answer (withdrawal, the patient's deletion, a hidden item on submit). The purger also deletes every object in the bucket older than 24 h that has no row: the ciphertext of a deleted file, or an upload that never reached its answer. In 0.1 the rule is `keep` with this basis, because nothing removes rows before the scan job exists; 3.1 makes it `delete` with its purger (`retention.NewService` refuses a delete rule without one) |
| `questionnaire_response` | Unchanged (questionnaire SPEC §5); its deletion now takes its files' rows with it |

DPIA notes: a new section "Questionnaire attachments (`docs/attachments`)" in `docs/privacy/dpia.md`, one
subsection per package that touches personal data (0.1, 1.7, 2.1, 3.1, 3.2), written in 4.1 from `PROGRESS.md`.
Processors: none new while the store and ClamAV are self-hosted; `processors.md` gains that line, the hosting
provider's pending entry names the object store and the `BINARY_KEK` KMS, and "Not processors" gains the
compose store and ClamAV (its signature download sends no personal data), all in 4.1.

## 6. Out of scope

Attachments anywhere but questionnaire answers (messages, consultations, documents a practitioner uploads), FHIR
`Binary` read and `DocumentReference`, presigned or direct-to-bucket URLs, resumable or chunked upload, server-side
transcoding, thumbnails or previews generated on the server, image editing or cropping, other file types (Word,
HEIC on the server, audio), checking a video's length, rescanning stored files when signatures change, a viewer
of the patient's own files on mobile, downloads of several files at once (a zip), stripping image metadata on
the server, and localisation (before production, with the rest of the app).

## 7. Up next

- **Export and erasure.** `Patient/$everything` (subject-rights feature) includes the patient's files as
  `Binary` resources, with `securityContext` their response.
- **Rescanning.** A periodic rescan of `clean` files with new signatures, if the owner wants it.
- **Attachments elsewhere.** Practitioner-uploaded documents (`DocumentReference`) reuse `binary` with a second
  holder; that second use is when `Holder` becomes a set of holders by resource type, not before.

## 8. Before production

- The hosting provider's EU object store chosen, its DPA signed and its entry in `processors.md`; the bucket
  private, versioning off (a deleted file must be gone), its own encryption at rest on.
- The key-encryption key from the KMS instead of `BINARY_KEK`, with rotation: a new `kek_id` for new files, and
  a job that rewraps the data keys of old ones (the bytes are not re-encrypted).
- ClamAV in production: self-hosted beside the worker, signature updates through a mirror or the official CDN,
  its memory sized from 3.1's reading.
- Legal decision on the retention of `binary` with `questionnaire_response`, and legal confirmation of the lawful
  basis (§4).
- The new web and mobile text, and the iOS and Android permission text, localised (bs/hr/sr).
- The camera, photo library, document picker and conversion checked on a real Android device and an iOS build,
  with a screen reader; a 200 MB video uploaded over a mobile network.
- The download headers checked again in the production browser matrix (the PDF viewer under the CSP).

## 9. Done when

All packages are committed on `feat/attachments`; `make lint`, `make test`, `make archcheck`, `make cover` (every
package at 80% or more), `make fhir-validate` and `make e2e` pass locally and in CI; `make gen-clients` leaves no
diff; the web and mobile checks pass; `docs/privacy/` describes what exists; `PROGRESS.md` records every package
with its failing tests and the measurements of §2 "Performance".
