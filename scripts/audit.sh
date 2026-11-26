#!/usr/bin/env bash
# The dependency audit (docs/deploy/BEFORE-PRODUCTION.md "Dependencies"), all in Docker, read-only on the source:
#   govulncheck   the Go module and tools/ (reachability: a finding names the call path that reaches it)
#   npm           npm audit of web/, landing/ and test/e2e/web/ (their lock files)
#   mobile        osv-scanner over mobile/pubspec.lock (pub has no audit command of its own)
#   images        Trivy over the four production images built locally from this checkout (salus, salus-web,
#                 salus-landing, salus-caddy) and the third-party images the production compose file pins
# The scanners are pinned by version (and the images by digest); Trivy is the Images workflow's own pin. Output goes
# to the terminal and, per step, to $AUDIT_OUT (default ${TMPDIR:-/tmp}/salus-audit, outside the checkout; Trivy's
# database is cached there). The script reports; it fails (exit 1) when any step found something or could not run, so a run is
# read, not trusted blindly. The CI gate stays the Images workflow (CRITICAL fails a digest).
#
#   scripts/audit.sh [govulncheck|npm|mobile|images]...   (default: all four)
set -uo pipefail
cd "$(dirname "$0")/.." || exit 2

GOVULNCHECK=golang.org/x/vuln/cmd/govulncheck@v1.9.0
OSV_SCANNER=ghcr.io/google/osv-scanner:v2.6.0@sha256:afd838850ac1a0fcc15ff4a041dc9ba11123c3f0d2666217a5f0fcf9222b55fa
TRIVY_IMAGE=$(sed -n 's/^ *TRIVY_IMAGE: *//p' .github/workflows/images.yml)
TRIVY_DB_REPOSITORY=mirror.gcr.io/aquasec/trivy-db:2,ghcr.io/aquasecurity/trivy-db:2
OUT=$(realpath -m "${AUDIT_OUT:-${TMPDIR:-/tmp}/salus-audit}")
COMPOSE=(docker compose -f deployments/docker-compose.yml)
mkdir -p "$OUT"
status=0
found() {
	echo "audit: $1" >&2
	status=1
}

govulncheck_() {
	echo "== govulncheck ($GOVULNCHECK)"
	"${COMPOSE[@]}" run --rm -T --no-deps tools sh -c \
		"go run $GOVULNCHECK ./... && cd tools && go run $GOVULNCHECK ./..." 2>&1 | grep -v 'level=warning' |
		tee "$OUT/govulncheck.txt" || found "govulncheck reported a vulnerability (or failed): $OUT/govulncheck.txt"
}

npm_() {
	local d n
	for d in web landing test/e2e/web; do
		n=${d//\//_}
		echo "== npm audit $d"
		"${COMPOSE[@]}" run --rm -T --no-deps -w "/src/$d" node npm audit 2>&1 | grep -v 'level=warning' |
			tee "$OUT/npm-$n.txt" || found "npm audit of $d reported a vulnerability (or failed): $OUT/npm-$n.txt"
	done
}

mobile_() {
	echo "== osv-scanner mobile/pubspec.lock"
	docker run --rm -v "$PWD/mobile:/mobile:ro" "$OSV_SCANNER" scan source --lockfile /mobile/pubspec.lock 2>&1 |
		tee "$OUT/osv-mobile.txt" || found "osv-scanner reported a vulnerability (or failed): $OUT/osv-mobile.txt"
}

images_() {
	local spec name ctx file img third nocache
	mkdir -p "$OUT/trivy-cache"
	for spec in salus:.:build/package/Dockerfile salus-web:web:build/package/web.Dockerfile \
		salus-landing:landing:build/package/landing.Dockerfile salus-caddy:build/package:build/package/caddy.Dockerfile; do
		IFS=: read -r name ctx file <<<"$spec"
		echo "== build $name"
		# As the Images workflow: the nginx images' runtime stage (its apk upgrade) is rebuilt without the cache.
		nocache=()
		[[ $name == salus-web || $name == salus-landing ]] && nocache=(--no-cache-filter runtime)
		docker build -q --platform linux/amd64 "${nocache[@]}" -t "salus-audit/$name:local" -f "$file" "$ctx" >/dev/null ||
			{ found "building $name failed"; continue; }
	done
	third=$(sed -n 's/^ *image: *\([a-z][^$]*@sha256:[0-9a-f]*\)$/\1/p' deployments/production/docker-compose.prod.yml)
	for img in salus-audit/salus:local salus-audit/salus-web:local salus-audit/salus-landing:local \
		salus-audit/salus-caddy:local $third; do
		echo "== trivy $img"
		docker run --rm -e TRIVY_DB_REPOSITORY -v /var/run/docker.sock:/var/run/docker.sock:ro \
			-v "$OUT/trivy-cache:/root/.cache/trivy" -v "$PWD/.trivyignore.yaml:/trivyignore.yaml:ro" \
			"$TRIVY_IMAGE" image --scanners vuln --ignorefile /trivyignore.yaml --severity HIGH,CRITICAL \
			--exit-code 1 --no-progress "$img" 2>&1 | tee "$OUT/trivy-${img//[\/:@]/_}.txt" ||
			found "trivy found HIGH or CRITICAL in $img (or failed)"
	done
}
export TRIVY_DB_REPOSITORY

(($#)) || set -- govulncheck npm mobile images
for step in "$@"; do
	case $step in
	govulncheck) govulncheck_ ;;
	npm) npm_ ;;
	mobile) mobile_ ;;
	images) images_ ;;
	*)
		echo "unknown step: $step" >&2
		exit 2
		;;
	esac
done
exit "$status"
