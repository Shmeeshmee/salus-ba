# Flutter SDK for `flutter analyze`, `flutter test` and code generation in mobile/. Built from the official
# SDK archive because no maintained image publishes this version. Running on a device or emulator is a
# manual step on the host (see README.md), not part of any check.
FROM debian:trixie-slim

ARG FLUTTER_VERSION=3.47.7
ENV PATH=/opt/flutter/bin:$PATH \
    PUB_CACHE=/cache/pub \
    HOME=/cache/home \
    FLUTTER_SUPPRESS_ANALYTICS=true

# One layer: a chmod in a later layer would store a second copy of the whole SDK.
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates curl git unzip xz-utils \
 && rm -rf /var/lib/apt/lists/* \
 && curl -fsSL "https://storage.googleapis.com/flutter_infra_release/releases/stable/linux/flutter_linux_${FLUTTER_VERSION}-stable.tar.xz" \
    | tar -xJ -C /opt \
 && git config --system --add safe.directory '*' \
 && mkdir -p /cache/pub /cache/home \
 && flutter config --no-analytics >/dev/null \
 && flutter precache --no-android --no-ios --universal \
 && chmod -R a+rwX /opt/flutter /cache
WORKDIR /src/mobile
