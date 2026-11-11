# Reverse proxy for production (deployments/production/): Caddy with the layer4 module (github.com/mholt/caddy-l4),
# which terminates TURN over TLS on 443 by SNI in front of the HTTP server (docs/adr/0011-single-server-deployment.md).
# The official recipe for a custom build (xcaddy in the builder image, the binary copied onto the plain image, which
# keeps its layout: config /etc/caddy/Caddyfile, state /data, config dir /config), except that Caddy runs as the
# non-root user caddy (uid and gid 10001), which owns /data and /config. The binary carries the file capability
# cap_net_bind_service so that user can bind 80 and 443; it is set in the build stage, so libcap is not in the
# runtime image. Docker also sets net.ipv4.ip_unprivileged_port_start=0 in a container by default, which is enough on
# its own (and the only thing that works under no-new-privileges, which ignores file capabilities); the capability
# keeps Caddy working where that sysctl is not 0. With `cap_drop: [ALL]`, add NET_BIND_SERVICE back.
# xcaddy builds the Caddy version the builder image names (CADDY_VERSION), so the two FROM lines move together
# (Dependabot's docker group). Check caddy-l4's go.mod when either moves: v0.1.2 requires Caddy v2.11.4 or later.
# golang.org/x/net is raised above the version Caddy's go.mod requires (v0.60.0 in v2.11.9) to v0.62.0, the first
# without CVE-2026-78661 (HIGH, in x/net/html). xcaddy's --with takes a package, not a bare module, so it names the
# affected package x/net/html; drop that --with once a Caddy release requires v0.62.0 or later. Trivy still reports
# github.com/go-jose/go-jose/v3 v3.0.5 (CVE-2026-116587, HIGH), which has no fixed release yet.
FROM caddy:2.11.9-builder@sha256:1b828679ae08622c534d8e83ed80c6c25d343202f0586fe3b52d8e0ce33413b9 AS build
RUN xcaddy build --with github.com/mholt/caddy-l4@v0.1.2 --with golang.org/x/net/html@v0.62.0 \
    && setcap cap_net_bind_service=+ep /usr/bin/caddy

FROM caddy:2.11.9-alpine@sha256:0bbb055bda6793c999bde78b1b5ce3e89f829fb031af2cf50056ea1246559fee
RUN addgroup -S -g 10001 caddy \
    && adduser -S -D -H -h /data -s /sbin/nologin -u 10001 -G caddy caddy \
    && chown -R caddy:caddy /data /config
COPY --from=build /usr/bin/caddy /usr/bin/caddy
USER caddy:caddy
