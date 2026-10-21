# The HL7 FHIR validator for `make fhir-validate`. The jar is pinned by version and checksum; the R4 core
# package it downloads on first use is cached in a named volume.
FROM eclipse-temurin:21-jre@sha256:227118764cccd3c3294617061aaa93a08309c63a395eefa83e6b35032992ea99
ADD --checksum=sha256:e3e50f06b806437821b7aef8189a1b790415cb96c5036e0812ed62981452b377 \
    https://github.com/hapifhir/org.hl7.fhir.core/releases/download/7.0.2/validator_cli.jar /opt/validator_cli.jar
ENV JAVA_TOOL_OPTIONS=-Duser.home=/cache
WORKDIR /src
