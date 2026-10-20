# 6. Self-hosted LiveKit for video

Status: accepted, 2026-10-19

## Context

Consultations are live video between a patient and a practitioner. Video and audio are health data. A hosted
video service would be a processor holding special-category data, possibly outside the EU. Live transcription
is planned and needs access to the audio stream.

## Decision

Video runs on self-hosted LiveKit, in the EU, behind a provider port in the `consultation` domain. Live
transcription is a LiveKit agent (`cmd/agent`) with a pluggable speech-to-text engine behind its own port.
During bootstrap both are stubs: LiveKit runs in dev mode in compose and `cmd/agent` only starts and stops.

## Consequences

- No video processor DPA is needed while we host it ourselves; we own TLS, TURN and scaling.
- The port keeps LiveKit types out of the service layer, so a provider change touches one file.
- Choosing a speech-to-text engine is a later decision; a hosted engine would be a processor and needs a DPA,
  an entry in `docs/privacy/processors.md` and explicit SPEC approval before patient audio reaches it.
- No recordings are kept unless a SPEC says so.
