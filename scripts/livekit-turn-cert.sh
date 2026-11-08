#!/usr/bin/env bash
# Writes the local TURN/TLS certificate of the compose LiveKit (deployments/livekit.yaml, "turn") into
# deployments/livekit-turn/: a server certificate and key for turn.salus.test (the livekit service's alias on the compose network; .test is reserved, RFC 6761), and the
# certificate of the throwaway CA that signed it (ca/ca.crt), which `make turn-check` trusts. The CA's key is
# never written to disk, so nothing else can be signed with it. Synthetic and local only: production's TURN
# certificate is for LiveKit's public domain, from the hosting provider (docs/consultation/SPEC.md §8).
# Run with `make livekit-turn-cert`; commit the three files.
set -euo pipefail

out=deployments/livekit-turn
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
mkdir -p "$out/ca"

openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes -days 3650 \
	-subj "/CN=Salus local TURN CA (synthetic)" -keyout "$work/ca.key" -out "$out/ca/ca.crt" 2>/dev/null
openssl req -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes \
	-subj "/CN=turn.salus.test" -keyout "$out/turn.key" -out "$work/turn.csr" 2>/dev/null
printf 'subjectAltName=DNS:turn.salus.test\nextendedKeyUsage=serverAuth\nkeyUsage=digitalSignature\nbasicConstraints=CA:FALSE\n' >"$work/ext"
openssl x509 -req -in "$work/turn.csr" -CA "$out/ca/ca.crt" -CAkey "$work/ca.key" -CAserial "$work/ca.srl" -CAcreateserial -days 3650 \
	-extfile "$work/ext" -out "$out/turn.crt" 2>/dev/null
chmod 0644 "$out/turn.key" # read by the livekit container's own user
echo "wrote $out/turn.crt, $out/turn.key and $out/ca/ca.crt"
