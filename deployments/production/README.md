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
| `.env.production.example` | Every variable, with where its value comes from and which are secrets |

The images (DEP-1): `ghcr.io/${GHCR_OWNER}/salus` (every Go binary, distroless), `salus-web`, `salus-landing`,
`salus-caddy` (Caddy with `github.com/mholt/caddy-l4`). Third-party images are pinned by tag and digest.

## Hostnames and ports

| Hostname | Served by |
| --- | --- |
| `SALUS_DOMAIN` | landing |
| `app.SALUS_DOMAIN` | web, and the API paths of `web/vite.config.ts` to the api (`make lint` checks they match) |
| `api.SALUS_DOMAIN` | api, everything (the mobile app) |
| `livekit.SALUS_DOMAIN` | LiveKit's signalling (`wss://`, port 7880 inside the network) |
| `turn.SALUS_DOMAIN` | TURN over TLS on 443: Caddy's layer4 wrapper terminates TLS and forwards plain TCP to `livekit:5349` |

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

Two things the local smoke test cannot show, to check on the server with a real browser behind a UDP-blocking
network:

- TURN relays to the SFU's public address (`NODE_IP`) from inside the LiveKit container, a hairpin through
  Docker's port publishing. If relayed calls fail, run LiveKit with `network_mode: host` instead (the shape of
  LiveKit's own guide), keeping 7880 and 5349 off the public interface.
- LiveKit sees Caddy's address, not the client's, on TURN/TLS. LiveKit can take the client's address from a PROXY
  protocol header (`turn.proxy_protocol`, and `proxy_protocol v2` on Caddy's side) if a browser rejects the mapped
  address; not needed with Chrome or Safari so far.

## Healthchecks

| Service | Check |
| --- | --- |
| api | `/usr/local/bin/healthcheck http://127.0.0.1:8080/readyz` (`cmd/healthcheck`, in the Go image: the distroless image has no shell or wget). `/readyz` pings Postgres and Redis, so the deploy waits for an api that can serve |
| worker, scheduler, notifier, agent | None. They listen on nothing; a fatal error (a lost database at start, a bad setting) ends the process and `restart: unless-stopped` starts it again, which `docker compose ps` shows as restarting. A liveness endpoint in four binaries would add code for no measured need |
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

Garage's data (the attachments) is not backed up yet (ADR 0011 open point 7).

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
| clamd | 2048 MB (the signatures in memory, about 1.5 GB; `ConcurrentDatabaseReload no` keeps a reload from doubling it) |
| postgres | 1024 MB |
| worker | 768 MB (a scan reads an upload of up to 200 MB) |
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
   `GHCR_OWNER=local`, `SALUS_TAG=local`, any `LIVEKIT_NODE_IP`.
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

## For DEP-3 and DEP-4

- The project is always `-p salus` with both env files (command above). `docker compose config -q` with them is
  a cheap check before `up`.
- `SALUS_TAG` must be a full SHA that exists for all four images; `GHCR_OWNER` must be lower case.
- Deploy order: `pull`, then `up -d --wait`: every app service depends on `migrate` completing, so `up` runs it
  first and stops if it fails (running `run --rm migrate` before is equivalent). A rollback without `migrate`
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
- Back up the `postgres` volume through `pg_dump` (DEP-3); the `objectstore-*` volumes hold the attachments and
  the `clamav` volume only signatures (re-downloaded if lost).
