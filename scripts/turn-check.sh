#!/usr/bin/env bash
# `make turn-check` (docs/consultation/PROGRESS.md, package 10): blocks this container's UDP and LiveKit's ICE/TCP
# port, as a hotel Wi-Fi that lets only TCP 443 out would, then runs test/turn as the host user. The test proves
# that a call cannot connect without TURN and connects through LiveKit's TURN over TLS on 443. Runs as root only
# for iptables (the turn-check service has NET_ADMIN); the rules hold in this container's network namespace only.
set -euo pipefail

for ipt in iptables ip6tables; do
	"$ipt" -A OUTPUT -o lo -j ACCEPT # Docker's DNS on 127.0.0.11
	"$ipt" -A OUTPUT -p udp -j DROP
	"$ipt" -A OUTPUT -p tcp --dport 7881 -j DROP
done

exec setpriv --reuid="${HOST_UID:-1000}" --regid="${HOST_GID:-1000}" --clear-groups \
	go test -tags turn -count=1 -v ./test/turn/
