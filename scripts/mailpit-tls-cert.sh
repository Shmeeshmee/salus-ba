#!/usr/bin/env bash
# Writes the local SMTP/TLS certificate of the compose Mailpits that the SMTP sender's tests use for STARTTLS and
# implicit TLS (deployments/docker-compose.yml, mailpit-starttls and mailpit-tls) into deployments/mailpit-tls/: a
# server certificate and key for both service names, and the certificate of the throwaway CA that signed it
# (ca/ca.crt), which the tools service trusts beside the system roots (SSL_CERT_DIR). The CA's key is never written
# to disk, so nothing else can be signed with it. Synthetic and local only: a relay's certificate is its own.
# Run with `make mailpit-tls-cert`; commit the three files.
set -euo pipefail

out=deployments/mailpit-tls
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
mkdir -p "$out/ca"

openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes -days 3650 \
	-subj "/CN=Salus local SMTP CA (synthetic)" -keyout "$work/ca.key" -out "$out/ca/ca.crt" 2>/dev/null
openssl req -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes \
	-subj "/CN=mailpit-tls" -keyout "$out/smtp.key" -out "$work/smtp.csr" 2>/dev/null
printf 'subjectAltName=DNS:mailpit-starttls,DNS:mailpit-tls\nextendedKeyUsage=serverAuth\nkeyUsage=digitalSignature\nbasicConstraints=CA:FALSE\n' >"$work/ext"
openssl x509 -req -in "$work/smtp.csr" -CA "$out/ca/ca.crt" -CAkey "$work/ca.key" -CAserial "$work/ca.srl" -CAcreateserial -days 3650 \
	-extfile "$work/ext" -out "$out/smtp.crt" 2>/dev/null
chmod 0644 "$out/smtp.key" # read by the mailpit containers' own user
echo "wrote $out/smtp.crt, $out/smtp.key and $out/ca/ca.crt"
