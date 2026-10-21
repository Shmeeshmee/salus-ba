#!/usr/bin/env bash
# make fhir-validate, inside the fhir-validator container: validate every golden FHIR response
# (testdata/*.json with a resourceType) against FHIR R4 and the profiles in api/fhir/ once there are any.
set -euo pipefail

mapfile -t files < <(find cmd internal pkg -path '*/testdata/*.json' -exec grep -l '"resourceType"' {} + | sort)
if [ "${#files[@]}" -eq 0 ]; then
  echo "fhir-validate: no golden FHIR files" >&2
  exit 1
fi
igs=()
if compgen -G 'api/fhir/StructureDefinition-*.json' >/dev/null; then
  igs=(-ig api/fhir)
fi
# -tx n/a: no terminology server; codes are checked against the core package only.
java -jar /opt/validator_cli.jar -version 4.0.1 -tx n/a "${igs[@]}" "${files[@]}"
