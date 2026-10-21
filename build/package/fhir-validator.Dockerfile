# The HL7 FHIR validator for `make fhir-validate`. The jar is pinned by version and checksum; the R4 core
# package it downloads on first use is cached in a named volume.
FROM eclipse-temurin:25-jre@sha256:dbb43d81099eec91e0800226ed2296d03146ee7857165b95112372ad0d67e87f
ADD --checksum=sha256:e3e50f06b806437821b7aef8189a1b790415cb96c5036e0812ed62981452b377 \
    https://github.com/hapifhir/org.hl7.fhir.core/releases/download/7.0.2/validator_cli.jar /opt/validator_cli.jar
ENV JAVA_TOOL_OPTIONS=-Duser.home=/cache
WORKDIR /src
