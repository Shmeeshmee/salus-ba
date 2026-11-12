# Production stack

One server, one compose project, `salus` (docs/adr/0011-single-server-deployment.md, docs/plans/deploy.md DEP-2).
The deploy (DEP-4) copies this directory to `/opt/salus/deployments/` and runs every command as:

```
docker compose -p salus --env-file /opt/salus/.env --env-file /opt/salus/release.env \
  -f /opt/salus/deployments/docker-compose.prod.yml <command>
```

`/opt/salus/.env` holds every variable of `.env.production.example`; `/opt/salus/release.env` holds
`SALUS_TAG=<full commit SHA>`, the tag of all four images. Relative paths in the compose file resolve from its own
directory, so the Caddyfile, `livekit.yaml` and `garage.toml` must sit beside it.

## Files

| File | What it is |
| --- | --- |
| `docker-compose.prod.yml` | The stack: caddy, migrate (one-shot), api, worker, scheduler, notifier, agent, web, landing, postgres, redis, livekit, objectstore (Garage), clamd |
| `Caddyfile` | TLS (Let's Encrypt) for every hostname, the routes, HSTS, the runtime log without client addresses, and TURN over TLS on 443 (the `layer4` listener wrapper) |
| `livekit.yaml` | LiveKit: TURN behind Caddy's TLS, log level `warn`, rooms created by the api only, the webhook to the api; keys, node IP, TURN domain and the webhook's key come from the environment |
| `garage.toml` | Garage, one node, fsync on |
| `.env.production.example` | Every variable, with where its value comes from and which are secrets (not `BINARY_KEK` or `TOTP_KEY`: "Keys") |
| `bootstrap/volume.sh` | `salus-volume` on the server: the one-time setup of the encrypted volume, the unlock after a reboot, the key files ("Encrypted volume") |

The images (DEP-1): `ghcr.io/${GHCR_OWNER}/salus` (every Go binary, distroless), `salus-web`, `salus-landing`,
`salus-caddy` (Caddy with `github.com/mholt/caddy-l4`). Third-party images are pinned by tag and digest.

## Hostnames and ports

| Hostname | Served by |
| --- | --- |
| `SALUS_DOMAIN` | landing |
| `app.SALUS_DOMAIN` | web, and the API paths of `web/vite.config.ts` to the api (`make lint` checks they match) |
| `api.SALUS_DOMAIN` | api, everything (the mobile app) |
| `livekit.SALUS_DOMAIN` | LiveKit's signalling (`wss://`, port 7880 inside the network); its server API (`/twirp/*`) answers 404 and stays on the compose network |
| `turn.SALUS_DOMAIN` | TURN over TLS on 443: Caddy's layer4 wrapper terminates TLS and forwards plain TCP to `livekit:5349` |

`/readyz` answers 404 on every public host: it names the api's failing dependencies, and only Docker's healthcheck
needs it (on 127.0.0.1). `/healthz` stays public.

Each hostname needs a DNS A record to the server (and AAAA if IPv6 is used, see below).

Published ports, and only these: 80/tcp, 443/tcp, 443/udp (Caddy: HTTP, HTTPS, HTTP/3, TURN/TLS), 3478/udp (TURN/UDP),
7881/tcp and 7882/udp (LiveKit's media). Postgres, Redis, Garage, clamd, the api, LiveKit's signalling and its TURN
listener are on the compose network only. Docker's published ports bypass `ufw`, so the compose file is the real
boundary; DEP-3's bootstrap opens the same list in `ufw` (plus SSH).

The FHIR base (`FHIR_BASE_URL`) and the attachments' download links are on `app.`: the web app opens attachments
same-origin only. The api answers the same paths on `api.`.

## TURN over TLS

LiveKit's own single-server layout: TURN clients connect to `turns:turn.SALUS_DOMAIN:443`. Caddy's `layer4`
listener wrapper on `:443` matches a TLS ClientHello with that SNI (and not ACME's `acme-tls/1` ALPN, so either
ACME challenge can renew the name), terminates TLS with the certificate Caddy manages for `turn.` (the site block
`turn.SALUS_DOMAIN` exists only for that), and proxies the plain stream to LiveKit's TURN listener `livekit:5349`.
LiveKit runs with `turn.external_tls: true`, holds no certificate, and always offers `turns:<domain>:443`. Every
other connection on 443 goes on to Caddy's HTTP server.

LiveKit stays on the compose network (bridge), with node IP `NODE_IP` (the server's public IPv4): it offers only
that address in its ICE candidates, so every participant, including those inside the stack, sends media to
`NODE_IP:7882/udp` or `:7881/tcp`. Two paths hairpin from the compose network to the host's own address and back
through Docker's port publishing; both were tested here (Docker 29.9.3, default `userland-proxy`, `NODE_IP` set to
the machine's LAN address and LiveKit's ports published on it only):

- **The agent, or any participant on the compose network.** It signals on `ws://livekit:7880` and sends media to
  `NODE_IP`. `lk load-test` with a publisher on the compose network and a subscriber on the host network, and the
  other way round: audio and video at 1.3 to 1.4 Mbit/s with no packet loss both ways. A capture on LiveKit's
  interface showed the bridge client's packets arriving DNATed to LiveKit's container address.
- **TURN relays.** LiveKit's TURN relays from inside its container to the SFU at `NODE_IP:7882`. `make turn-check`'s
  test (`test/turn`, UDP and 7881/tcp blocked in the client's container) against this stack, through Caddy's layer4
  route with SNI `turn.salus.test`: no connection without TURN, a call through TURN over TLS on 443. The smoke test
  needed `allow_restricted_peer_cidrs` for the private LAN address; production's public `NODE_IP` needs none.

Not tested: TURN/UDP (3478) and TURN/TLS from a real browser on the internet. LiveKit sees Caddy's address, not the
client's, on TURN/TLS; if a browser rejects the mapped address, LiveKit can take the client's from a PROXY protocol
header (`turn.proxy_protocol`, and `proxy_protocol v2` on Caddy's side).

If a later Docker version stops hairpinning published ports, `network_mode: host` for LiveKit (LiveKit's own
layout) is the fallback: then 7880 and 5349 listen on the host and only `ufw` keeps them off the internet, Caddy,
the api, the worker and the agent reach LiveKit through the host gateway, and LiveKit reaches the api's webhook
through a port published on the gateway.

## Encrypted volume

This layout is for a **fresh server**: no server has run an earlier release yet. A server that already ran a
release with the named volumes (`salus_postgres`, `salus_objectstore-meta`, `salus_objectstore-data`) and the keys in
its env file needs a one-time move first, by hand, before the first deploy of this layout:

1. Stop the stack (`docker compose -p salus ... down`, no `-v`), set up the volume (`salus-volume setup`).
2. Copy the data, as root, preserving owners: `cp -a /var/lib/docker/volumes/salus_postgres/_data/. /mnt/salus-data/postgres/`,
   and the same for `salus_objectstore-meta` to `garage/meta/` and `salus_objectstore-data` to `garage/data/`. Then
   `salus-volume check`, and start the stack with `salus-volume unlock`, which checks the layout again.
3. Once the stack works on the volume, remove the old named volumes (`docker volume rm salus_postgres
   salus_objectstore-meta salus_objectstore-data`).
4. Delete `BINARY_KEK` and `TOTP_KEY` from `/opt/salus/.env` and from every `/opt/salus/releases/*/.env` (the deploy
   refuses a rollback to a release whose env file has them), and from `SALUS_ENV_FILE` and Bitwarden's env note.
5. Treat the keys as exposed: they were in GitHub and on the unencrypted disk. Rotating `BINARY_KEK` needs a new
   `BINARY_KEK_ID` and re-wrapping every stored data key, and `TOTP_KEY` re-sealing every MFA secret; neither has a
   tool yet. If they are not rotated, record the exposure and the owner's decision in docs/privacy/.

Postgres's and Garage's data and the key files sit on a Hetzner Volume encrypted with LUKS2 (ADR 0011, "Decided
before real patient data"), mounted at `SALUS_DATA_DIR` (default `/mnt/salus-data`). The server's own disk holds
no personal data: no swap (the bootstrap fails while any is active), clamd's temporary files on the volume, container
logs without personal data (CLAUDE.md).

- **Set up once** with `salus-volume setup /dev/disk/by-id/scsi-0HC_Volume_<id>` (as root; docs/deploy/README.md):
  `cryptsetup luksFormat --type luks2` asks for the passphrase (Bitwarden only, never on the server or in GitHub),
  then the filesystem (ext4, mounted `nodev,nosuid,noexec`), the layout below and the key files, pasted from
  Bitwarden at a prompt that does not echo. It refuses a device that holds any signature (`blkid -p`), is mounted,
  or that `/etc/fstab` mounts (Hetzner's automatic mount).
- **Unlocked by hand after every boot**: `salus-volume unlock` (as root) opens the volume (the passphrase), mounts
  it, checks the layout and both key files, and starts the stack with the running release (`--no-start` prints the
  command instead). Running it again changes nothing. Nothing opens it at boot: no `/etc/crypttab` line (so no key
  file for it anywhere), no `/etc/fstab` line (so a boot never waits for a passphrase). Until it is unlocked the
  bind mounts below do not exist, and Postgres, Garage, the api and the worker do not start: their mounts use
  `create_host_path: false`, so Docker never creates an empty directory on the unencrypted disk in their place.
  Automatic reboots stay off (the bootstrap refuses `UPGRADE_AUTO_REBOOT=yes`).
- **The deploy refuses** (`deploy.sh`, a deploy and a rollback alike, before anything is pulled, migrated or
  switched) when `SALUS_DATA_DIR` is not the mount point of a `crypt` device (`findmnt`, `lsblk`), when a key file is
  missing or not `65532`'s with mode 400, and when the env file sets `BINARY_KEK` or `TOTP_KEY` (the deploy workflow
  refuses such a `SALUS_ENV_FILE` before uploading it).

| Path on the volume | Owner, mode | What |
| --- | --- | --- |
| `.` | `root:deploy` 0710 | The deploy user may pass (to check the key files), no other account may |
| `postgres/` | `999:999` 0700 | Postgres's data (`/var/lib/postgresql`; the image's `postgres` uid) |
| `garage/meta/`, `garage/data/` | `root:root` 0700 | Garage's metadata and blocks (Garage runs as root) |
| `clamd/` | `100:101` 0700 | clamd's `TemporaryDirectory` (the clamav image's `clamav` user): the uploads it scans and what it unpacks |
| `keys/` | `root:deploy` 0710 | The key files |
| `keys/binary_kek`, `keys/totp_key` | `65532:65532` 0400 | The keys, base64 and a newline |

`salus-volume check` (as root, the only account that reads the key files) reports every difference from this table,
and a host account with uid 65532. `salus-volume unlock` sets the directories' owners and modes to the table before it
checks (a volume moved from another server may carry its `deploy` gid); it never changes the key files. Unlock starts
the stack holding `deploy.sh`'s lock (`/opt/salus/.deploy.lock`) and refuses while a deploy or rollback holds it.

## Keys

`BINARY_KEK` (wraps every attachment's data key) and `TOTP_KEY` (seals the MFA secrets) are in the key files on the
encrypted volume and in Bitwarden, nowhere else: not in `/opt/salus/.env`, not in the `SALUS_ENV_FILE` GitHub
secret, not in the release directories. They reach only the containers that read them: the api (both) and the worker
(`BINARY_KEK`; `cmd/worker` loads the object store). migrate, scheduler, notifier and agent get neither.

How: compose file-based `secrets:`. The Docker daemon, which runs as root, bind-mounts each file read-only at
`/run/secrets/binary_kek` and `/run/secrets/totp_key`, and the binaries read them through `BINARY_KEK_FILE` and
`TOTP_KEY_FILE` (`internal/config`, which also takes the plain variables for local runs and CI; setting both a
variable and its `_FILE` is an error). Why this design:

- The compose CLI runs as `deploy`, so an `env_file:` of a root-only file cannot be read by it; and anything that
  goes through compose's interpolation ends up in `docker inspect`'s environment. A secret is a bind mount the daemon
  makes, and the value never passes through the CLI or the container's environment.
- Compose ignores `uid`/`gid`/`mode` for file secrets (it is a plain bind mount), so the file on the volume carries
  the owner the container needs: `65532`, the distroless `nonroot` uid the Go image runs as. No host account has that
  uid (`salus-volume check` fails if one does), so on the host only root reads the keys; the directories are
  `root:deploy` 0710, so `deploy` can see that the files exist and what their owner and mode are, and cannot list or
  read them.
- Two files rather than ADR 0011's "one key file": a compose secret is one whole file, and a `_FILE` variable reads
  one value. They sit together in one root-only directory and are written together by `salus-volume`.

The limits are ADR 0011's: `deploy` is in the `docker` group and so root-equivalent; the files protect the keys from
the env file's copies, from a copy of the server's disk and from a detached volume, not from someone with `deploy` or
root on the running server.

## Healthchecks

| Service | Check |
| --- | --- |
| api | `/usr/local/bin/healthcheck http://127.0.0.1:8080/readyz` (`cmd/healthcheck`, in the Go image: the distroless image has no shell or wget). `/readyz` pings Postgres and Redis, so the deploy waits for an api that can serve |
| worker, scheduler, notifier, agent | None (ADR 0011, as amended). They listen on nothing; a fatal error (a lost database at start, a bad setting) ends the process and `restart: unless-stopped` starts it again, which `docker compose ps` shows as restarting. A liveness endpoint in four binaries would add code for no measured need |
| migrate | None: one-shot. The app services wait for it to complete successfully |
| caddy | `wget` to its admin API on the container's loopback (2019) |
| web, landing | `wget` to nginx's `/healthz` |
| postgres, redis | `pg_isready` over TCP, `redis-cli ping` |
| livekit | `wget` to its HTTP port |
| objectstore | `garage bucket info <bucket>`: healthy once the bucket exists |
| clamd | `clamdcheck.sh` (PING/PONG); 10 minutes' start period, as the first start downloads the signatures |

`docker compose up -d --wait` waits for every check above and for the others to be running.

## Garage

`garage server --single-node --default-bucket`: on the first start Garage lays out a one-node cluster, then creates
the bucket `S3_BUCKET` and the access key `S3_ACCESS_KEY_ID`/`S3_SECRET_ACCESS_KEY` from the environment and grants
the key the bucket. On every later start it finds them and changes nothing, so no init job is needed. Changing the
key variables makes the next start create and grant the new key beside the old one (checked in the smoke test), so
a rotation is: new values in `.env`, deploy, then `garage key delete <old id>` inside the container.

Garage's data (the attachments) is backed up every night right after the Postgres dump, read through its S3 API
(docs/deploy/backup-restore.md); how long the backups are kept is ADR 0011 point 7.

## Redis

No persistence: no RDB snapshots, no AOF, no volume. Redis holds temporary state only (ADR 0005): slot holds, rate
limits, MFA challenges, the consultations' event streams, each key with a TTL and none the only copy of anything.
A restart costs the holds and challenges in flight, never a record. `maxmemory` (192 MB) sits below the container's
limit with `volatile-ttl`, so Redis evicts the keys nearest their end rather than being killed.

## Memory budget

Defaults for a 4 vCPU / 8 GB server (`SALUS_MEM_*` in `.env` change them). Limits, not reservations; not measured
under load yet (docs/consultation/SPEC.md §8).

| Service | Limit |
| --- | --- |
| clamd | 2048 MB (the signatures in memory, about 1.5 GB; `ConcurrentDatabaseReload no` keeps a reload from doubling it). The scans' temporary files are not in it: clamd spools each stream and what it unpacks to `SALUS_DATA_DIR/clamd` on the encrypted volume, not a tmpfs, because the worker runs two scans at once, each up to 250 MB streamed plus 250 MB unpacked: a tmpfs for that (1 GB) would count towards this limit and take the budget to about 7.7 GB of 8. The cost is disk I/O on the volume during a scan |
| postgres | 1024 MB |
| worker | 768 MB (a scan reads an upload of up to 200 MB; a chat image decodes in at most 192 MiB, one at a time), with `GOMEMLIMIT` at 640 MiB (`SALUS_WORKER_GOMEMLIMIT`) so the collector works harder before the kernel kills it |
| livekit | 768 MB |
| api | 512 MB |
| agent | 384 MB |
| objectstore | 384 MB |
| redis | 256 MB |
| caddy | 192 MB |
| scheduler, notifier | 128 MB each |
| web, landing | 64 MB each |
| **Long-running total** | **6720 MB** (6.6 GB), leaving about 1.3 GB for the OS, Docker and the page cache |
| migrate (one-shot, before the others start) | 256 MB |

## Smoke test on a developer machine

No ports on the host, Caddy on its internal CA instead of Let's Encrypt, the images built locally:

1. Build the images and tag them as the compose file expects, e.g. `ghcr.io/local/salus:local` (the Go image from
   `build/package/Dockerfile`, the web and Caddy images from DEP-1's Dockerfiles, the landing image from
   `build/package/landing.Dockerfile`).
2. Write an env file from `.env.production.example` with generated values, `SALUS_DOMAIN=salus.test`,
   `GHCR_OWNER=local`, `SALUS_TAG=local`, any `LIVEKIT_NODE_IP`, and `SALUS_DATA_DIR` set to a scratch directory
   holding `postgres/`, `garage/meta/`, `garage/data/` and `keys/binary_kek`, `keys/totp_key` (base64 of 32
   synthetic bytes each, owned by 65532 with mode 400: write them from a container, which may chown). The smoke
   test's directory is not encrypted; only the server's must be.
3. Copy the Caddyfile, replace the `email` and `acme_ca` lines with `local_certs`, and write an override that mounts
   it and removes the host ports:
   ```yaml
   services:
     caddy:
       ports: !reset []
       volumes: !override
         - /path/to/Caddyfile.smoke:/etc/caddy/Caddyfile:ro
         - caddy-data:/data
         - caddy-config:/config
     livekit:
       ports: !reset []
   ```
4. `docker compose -p salus-smoke-prod --env-file smoke.env -f deployments/production/docker-compose.prod.yml -f override.yml up -d --wait`
5. From a container on the network (`docker run --rm --network salus-smoke-prod_default curlimages/curl ...`),
   with `--resolve <host>:443:<caddy's address>` and `-k`: the landing, `app.` (index and a deep link such as
   `/appointments/x`), `/healthz` on `app.` and `api.`, `livekit.`; and TLS with SNI `turn.salus.test` answering a
   STUN Binding request.
6. `docker compose -p salus-smoke-prod down -v`.

Never use the project name `salus` on a machine that runs another Salus stack.

## Blockers before go-live

E-mail needs these, set to real values in `SALUS_ENV_FILE` (`.env.production.example` has placeholders):

- `SMTP_*`: decided, an EU SMTP relay, Brevo or Mailjet (ADR 0011 point 5); its DPA pending (point 1). The
  notifier sends with STARTTLS (587) or implicit TLS (465), the certificate verified, and AUTH PLAIN with the
  relay's SMTP login, from `SMTP_FROM` on a domain verified at the relay with SPF, DKIM and DMARC
  (docs/deploy/README.md, "E-mail relay"). Without a working login the email jobs fail and River retries them:
  staff invitations do not reach the invitee, and patients get no appointment e-mails. No e-mail to a real person
  before the relay's DPA is signed and recorded in docs/privacy/processors.md.
- `PATIENT_APP_URL`: required; every appointment e-mail links to `<PATIENT_APP_URL>/appointments`. The patient
  app's link domain is not decided yet.

## For DEP-3 and DEP-4

- The project is always `-p salus` with both env files (command above). `docker compose config -q` with them is
  a cheap check before `up`.
- `SALUS_TAG` must be a full SHA that exists for all four images; `GHCR_OWNER` must be lower case.
- Deploy order: `pull`, then `up -d --wait`: every app service depends on `migrate` completing, so `up` runs it
  first and stops if it fails. The old containers are already gone by then (every image changed, so compose
  recreates every service): a deploy is a short outage, and a failed migrate lasts until the rollback is up (running `run --rm migrate` before is equivalent). A rollback without `migrate`
  writes the previous tag and runs `up -d --wait --no-deps` with every service but `migrate` named (forward-only
  migrations, ADR 0011).
- `ufw` opens 80/tcp, 443/tcp, 443/udp, 3478/udp, 7881/tcp, 7882/udp and SSH; nothing else is published.
- IPv6: with Docker's default bridge (no `enable_ipv6`), connections over IPv6 reach Caddy through Docker's
  userland proxy and arrive from the bridge's gateway, so the api's rate limits would see one address for all
  IPv6 clients. Either publish only IPv4 records (A, no AAAA) or enable IPv6 on the network in a later change.
- The Caddy image runs as a non-root user (DEP-1): a `caddy-data` or `caddy-config` volume first created by a
  root-run Caddy keeps root's ownership and must be chowned to that user once.
- Caddy's state is the `caddy-data` volume (certificates, ACME account). Keep it across deploys; deleting it
  re-issues every certificate and can hit Let's Encrypt's rate limits. Start a new server on the staging CA
  (`SALUS_ACME_CA`), then switch and remove the staging certificates from `caddy-data`.
- Back up Postgres through `pg_dump` (DEP-3); `SALUS_DATA_DIR/garage/` on the encrypted volume holds the
  attachments and the `clamav` volume only signatures (re-downloaded if lost).
