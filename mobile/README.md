# salus

Salus app for patients

## Getting Started

This project is a starting point for a Flutter application.

A few resources to get you started if this is your first Flutter project:

- [Learn Flutter](https://docs.flutter.dev/get-started/learn-flutter)
- [Write your first Flutter app](https://docs.flutter.dev/get-started/codelab)
- [Flutter learning resources](https://docs.flutter.dev/reference/learning-resources)

For help getting started with Flutter development, view the
[online documentation](https://docs.flutter.dev/), which offers tutorials,
samples, guidance on mobile development, and a full API reference.

## Questionnaire files: the device check

The pickers, the camera's permission and the on-device conversion (`lib/files.dart`, `MainActivity.kt`,
`AppDelegate.swift`) are platform code that the widget tests fake; check them by hand on a device or emulator
after a change (docs/attachments/SPEC.md §2 "Mobile"), with synthetic files only:

- a questionnaire with a file upload item for each kind: "Add file" offers only the item's sources;
- "Take a photo" on Android explains the camera first (M18), and a refused camera shows M19 with "Open settings";
- a HEIC photo uploads as a JPEG, a MOV or HEVC video as an H.264/AAC MP4, and both open in Chrome on the web app;
- the uploaded photo and video carry no location, device or time (`exiftool` on the downloaded file shows only
  the orientation), and a portrait photo and video still show upright;
- the app's cache holds no `salus-upload-*` or `salus-converted-*` file once the upload is done.
