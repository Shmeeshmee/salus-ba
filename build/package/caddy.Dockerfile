# Reverse proxy for production (deployments/production/): Caddy with the layer4 module (github.com/mholt/caddy-l4),
# which terminates TURN over TLS on 443 by SNI in front of the HTTP server (docs/adr/0011-single-server-deployment.md).
# The official recipe for a custom build: xcaddy in the builder image, the binary copied onto the plain image, which
# keeps its layout (config /etc/caddy/Caddyfile, state /data, config dir /config) and its user. xcaddy builds the
# Caddy version the builder image names (CADDY_VERSION), so the two FROM lines move together (Dependabot's docker group).
# Check caddy-l4's go.mod when either moves: v0.1.2 requires Caddy v2.11.4 or later.
FROM caddy:2.11.9-builder@sha256:1b828679ae08622c534d8e83ed80c6c25d343202f0586fe3b52d8e0ce33413b9 AS build
RUN xcaddy build --with github.com/mholt/caddy-l4@v0.1.2

FROM caddy:2.11.9-alpine@sha256:0bbb055bda6793c999bde78b1b5ce3e89f829fb031af2cf50056ea1246559fee
COPY --from=build /usr/bin/caddy /usr/bin/caddy
