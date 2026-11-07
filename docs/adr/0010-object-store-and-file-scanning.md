# 10. Garage as the local object store, aws-sdk-go-v2 as its client, ClamAV for scanning

Status: accepted, 2026-11-07 (docs/attachments, package 0.1)

## Context

Patients attach photos, documents and short videos to questionnaire answers (`docs/attachments/SPEC.md`). The
bytes go to an S3-compatible object store: in production the hosting provider's EU store (not chosen yet, SPEC
§8), locally a pinned container in compose. MinIO stopped publishing free Docker images in October 2025 and is
in maintenance mode, so the local store needs another choice. The api streams every file through itself in
constant memory (SPEC §3 O3), so the store must take a `PutObject` of known length whose body is not seekable,
serve a ranged `GetObject`, delete, and list page by page (the purger, package 3.1). Every file is scanned for
malware before anyone may open it (SPEC §3 O5).

The api needs exactly four operations: `PutObject` with a known `Content-Length` and an unsigned payload,
`GetObject` with one `Range`, `DeleteObject` and `ListObjectsV2` (with continuation tokens and each object's
`LastModified`).

## Options for the local store

| | Garage | SeaweedFS |
| --- | --- | --- |
| Licence | AGPL-3.0 (a separate dev-only container; nothing of it is linked into or shipped with Salus) | Apache-2.0 |
| Image | `dxflrs/garage`, one static binary, 20.6 MB compressed (amd64, v2.4.1), about 13 MB of memory idle | `chrislusf/seaweedfs`, about 197 MB compressed (amd64, `latest` of 2026-10-25); master, volume, filer and S3 gateway in one process |
| Maintenance | Active: Deuxfleurs, releases v2.0.0 (June 2025) to v2.5.0 (October 2026), commits this week | Active: frequent releases, commits this week |
| The four operations | All four, checked (below) | All four, by its documentation; not run, as Garage passed |
| Bucket and key | `garage server --single-node --default-bucket` lays the one-node cluster out at start and creates the bucket and the access key given in `GARAGE_DEFAULT_*`, with the key granted the bucket: no init service, idempotent on restart | An identities JSON file for the key; the bucket by an S3 `CreateBucket` or `weed shell` from an init service |
| Configuration | One TOML file, inline in compose (`configs:`), the RPC secret from the environment | Command-line flags and the identities file |

## Options for the Go client

CLAUDE.md: do not add a dependency for what the standard library does in under about 50 lines.

- **A hand-written SigV4 client**: canonical request, string to sign, the signing key chain, unsigned payload,
  the four requests and the XML of `ListObjectsV2` and of S3 errors. Well over 50 lines, and signing code we
  would own and have to get right.
- **`github.com/minio/minio-go/v7`** (v7.4.1, October 2026): still maintained (commits in November 2026), but its
  vendor's own server is in maintenance mode and the client's future is tied to it.
- **`github.com/aws/aws-sdk-go-v2/service/s3`** (v1.114.5, 2026-10-22): the reference client, maintained by AWS,
  speaks every S3-compatible store the production provider is likely to offer. Its transitive modules are all
  AWS's own, no third party: `aws-sdk-go-v2` (core: `aws`, the SigV4 signer), `aws/protocol/eventstream`,
  `internal/configsources`, `internal/endpoints/v2`, `internal/v4a`, `service/internal/accept-encoding`,
  `service/internal/checksum`, `service/internal/presigned-url`, `service/internal/s3shared`, and
  `github.com/aws/smithy-go`. The `config` and `credentials` modules (and with them STS, SSO and the INI parser)
  are not needed: `s3.New(s3.Options{…})` takes the endpoint, the region, an `aws.CredentialsProviderFunc` over
  the configured key pair, and the injected `*http.Client`.

## Decision

- **Local store: Garage** `dxflrs/garage:v2.4.1` (SPEC §3 O10's default), the compose service `objectstore` on
  the compose network only, port 3900, region `garage`, path-style addressing, bucket `salus-binaries`, a
  synthetic key pair, data in the `objectstore` volume. SeaweedFS stays the fallback.
- **Client: `github.com/aws/aws-sdk-go-v2/service/s3` v1.114.5**, pinned in `go.mod` (`// indirect` until
  package 1.4 imports it). The client is built with `RequestChecksumCalculation` and
  `ResponseChecksumValidation` set to `WhenRequired`, `UsePathStyle: true`, and each `PutObject` swaps the payload
  hash for `UNSIGNED-PAYLOAD` (`v4.SwapComputePayloadSHA256ForUnsignedPayloadMiddleware`): the body is a stream
  and AES-GCM authenticates the bytes (SPEC §2 "Encryption"). Production adds TLS to the store.
- **Scanning: ClamAV** `clamav/clamav:1.5.4`, the compose service `clamd` on TCP 3310, compose network only.
  `StreamMaxLength`, `MaxFileSize` and `MaxScanSize` are 250 MB, above the 200 MB video limit, set with the image's
  `CLAMD_CONF_*` variables (no bind mount); `ConcurrentDatabaseReload no`, so a signature reload does not double
  clamd's memory on a developer machine. The image carries a signature database, copied into the `clamav` volume
  on the first start; freshclam updates it once a day (it needs the internet and sends no personal data). The
  health check is the image's `clamdcheck.sh`, `PING` answered by `PONG`. The scan job streams the decrypted file
  to clamd's `INSTREAM` (`pkg/clamd`, package 1.3).

## The check (2026-11-07, from the `tools` container against the compose Garage)

A throwaway program (not committed) with aws-sdk-go-v2 v1.114.5 and the options above:

- `PutObject` of 3 MiB + 123 bytes from a non-seekable reader with `ContentLength` and `UNSIGNED-PAYLOAD`: OK,
  three objects.
- `GetObject` with `Range: bytes=65552-131103` (the second 64 KiB + 16 ciphertext segment): 65,552 bytes, equal to
  the source.
- `GetObject` of a missing key: `*types.NoSuchKey`.
- `ListObjectsV2` with `MaxKeys: 1` through the paginator: three pages, each object's size and `LastModified`.
- `DeleteObject` twice per key: OK both times.
- A body shorter than its `ContentLength`: the put fails, and no object is left (`HeadObject` finds none).
- With the SDK's default checksum settings (`WhenSupported`), the same put fails in the client before any
  request ("unseekable stream is not supported without TLS and trailing checksum"): `WhenRequired` is required.

ClamAV: healthy about 50 s after its first start on this machine with the bundled signatures, `PING` → `PONG`, the
three limits and the reload setting in `clamd.conf`, about 975 MiB resident idle; healthy again after a restart.

## Consequences

- No init service and no bind-mounted config file: every worktree's compose resolves the same `objectstore` and
  `clamd` services, so a run from another worktree does not recreate them.
- The store and clamd are self-hosted: no new processor while that holds (`docs/privacy/processors.md`, package
  4.1). The production store is the hosting provider's, with its DPA, private, versioning off and its own
  encryption at rest on top of ours (SPEC §8).
- One new Go dependency tree, AWS's own modules only. A provider whose store needs trailing checksums or TLS-only
  streaming is handled in the client's options, not in the domain.
- ClamAV needs about 1 GB of memory locally; its production placement and sizing come from package 3.1's
  measurement (SPEC §8).
