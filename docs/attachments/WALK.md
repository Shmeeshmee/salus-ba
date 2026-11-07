# Questionnaire attachments: acceptance walk

Date: 2026-11-07, 21:19 to 21:38 (Europe/Sarajevo). Branch `feat/attachments` at `4a0fe37` (PR #25). Run by a
Claude Code session for the owner. Screenshots are in `docs/attachments/walk/`, named by step. Every value below is
synthetic.

## Summary

| Step | What | Result |
| --- | --- | --- |
| 1 | Web builder: "First visit" revised with a "File upload" item (Photos and Documents, up to 3 files), published, still the role's default | Pass |
| 2 | Mobile on the Android emulator: Sara books, fills in "First visit", adds a camera photo and a PDF, removes and re-adds the PDF, tries a PDF over 20 MB, submits | Pass, with notes: the photo's "Uploading…" passed too fast for a screenshot; the over-limit PDF was refused by the app, not the server |
| 3 | EICAR PDF uploaded through the API as Sara: rejected, never downloadable, audited | Pass. M7 could not be shown by the app for a file uploaded outside it (expected, below) |
| 4 | Web: the GP opens the answers, the photo in the viewer, downloads the PDF (sha256 identical) | Pass, with a finding: the portrait photo is not fitted into the viewer at 1440x900 |
| 4 (O23) | The PDF in Chrome's own viewer under the content CSP | Not done: Playwright's headless Chromium has no PDF viewer; needs the owner in desktop Chrome |
| 5 | Stored objects encrypted, sizes, EXIF of the downloaded photo, logs free of file names | Pass |
| 6 | Existing e2e suite once | Pass (37 s) |

For the owner to look at:
1. **Photo viewer (web).** A portrait photo from the phone (1440x1920) is drawn at the dialog's full width
   (1040x1387 px) in a dialog 852 px high at 1440x900, so its lower third is cut off (`4-03-viewer-photo.png`).
   Whether the dialog scrolls to the rest was not checked. Expected: the photo fitted inside the viewer.
2. **Camera photo title.** A photo taken with the camera is uploaded under the picker's cache file name
   (`39f6ccbd-e0b7-47be-9a0c-6421932038c37154703606034263444.jpg`), which is what the practitioner then sees as its
   title (W8). The long name also pushes "Open" and "Download" onto a second line in the web list (`4-02-answers.png`),
   unlike the PDF's row. A plainer default title (for example the kind and a time) is a decision for the owner and
   would be a COPY change.
3. **Over-limit file refused on the device.** The app refused the 21 MB PDF with M8 before any request was sent
   (`mobile/lib/file_prep.dart`, `ready()` checks every kind's limit). PROGRESS 1.7 says a PDF's and a video's limits
   are left to the server; the code is stricter than that note. The server's own 413 `too_large` was checked
   separately through the API (step 3). Nothing wrong for the patient; the note in PROGRESS is out of date.
4. **Keyboard after "Remove".** After confirming "Remove" (M6) the focus went back to the first text answer and the
   keyboard opened again over the form. Minor.
5. **Chrome PDF viewer under `Content-Security-Policy: default-src 'none'; sandbox` (O23)** is still open: see step 4.
6. **The demo emulator container did not start** (below); the walk used a throwaway container from its image.

## Setup

### Accounts (fresh, synthetic)

The demo accounts were not used, read or changed (their credentials belong to another session). The walk made its
own accounts the way the e2e helpers do (`test/e2e/booking_test.go`, `accounts_test.go`, `questionnaire_test.go`),
through `salusctl` and the API, with a temporary test file that was deleted afterwards and is not committed. They are
named after the demo's people so the screenshots read like the demo, and nothing in them is real:

- Clinic "Walk synthetic clinic ba80ba63" (`salusctl create-clinic`, listed by its admin
  `walk-admin-ba80ba63@example.test`, who enrolled TOTP from the Mailpit invite).
- GP "Amar Mehmedović", `walk-amar-ba80ba63@example.test`, invited by the admin, TOTP enrolled; availability every day
  00:00 to 23:30, 30-minute slots.
- Questionnaire "First visit", two placeholder questions ("Placeholder question: what is the reason for your visit?",
  long answer, required; "Placeholder question: have you seen a practitioner about this before?", yes or no),
  published (version 1) and set as the GP role's default.
- Patient "Sara Synthetic", `walk-sara-ba80ba63@example.test`, registered with a synthetic profile (birth date
  1992-03-14, phone +38760000099) and consent to the clinic (`POST /consents`).
- All accounts use the e2e suite's synthetic password.

The walk's data stays in the dev database and its encrypted objects in the local bucket: the clinic, the accounts,
the appointment `732c2ee4-7474-409f-89e5-95c5140da9c4`, the submitted response
`08a5ac06-f66e-434d-acd6-3c00926286c8`, the two clean files and the rejected file's row (its object is gone).

### APK

`flutter build apk --debug --dart-define=API_BASE_URL=http://10.0.2.2:18080` from this worktree's `mobile/` (commit
`4a0fe37`), in a throwaway `docker run --rm` container from the local `salus-demo-android` image (Flutter 3.47.7), on a
copy of `mobile/` made inside the container (the worktree mounted read-only), with its own pub cache; only the APK
was copied out. Gradle 231 s; `app-debug.apk` 204,458,635 bytes, sha256
`d4aee42c4c4684e5216de6d79f5c50a407ab9e963e6823b0d3ec6e4da38e8fea`. The APK was kept in the session scratchpad only.

API base: the demo app was built with the README's `http://10.0.2.2:8080` (the string is in the demo APK's
`kernel_blob.bin`), the host's port through the emulator's alias `10.0.2.2`. On this machine 8080 belongs to another
project and the compose API is published on `SALUS_API_PORT=18080` (`deployments/.env`), so the walk's APK uses
`http://10.0.2.2:18080`. No `adb reverse` was set.

### Emulator

`docker start salus-demo-emu` started the container, which exited within seconds with code 1: `FATAL | Running
multiple emulators with the same AVD is an experimental feature.` Its command copies `/opt/android-home` into
`/tmp/ah` inside the container's own layer; after the crash earlier today (139) that copy, with the AVD's lock files,
was still there, so `cp -r` nested the fresh copy and the emulator found the old locks. The container cannot be fixed
without changing it, which the walk was not allowed to do. Instead the walk ran a throwaway container,
`salus-walk-emu`, from the same `salus-demo-emu` image with the same settings (host network, `/dev/kvm`, group 991, user
1000), the copy removed first, `-no-metrics` and `-camera-back virtualscene` (the virtual scene camera); it booted in
26 s (Android 15, API 35 image, 1080x2400) and was stopped and removed after step 4. `salus-demo-emu` and its image were
not changed and are left as they were (Exited (1)); `salus-demo-web` was not touched.

Driving: `adb shell input` (tap, text, keyevent), `uiautomator dump` to find elements by their text or accessibility
label, `screencap` for screenshots (about 0.3 s apart in a loop to catch the upload states). Synthetic files pushed to
`/sdcard/Download`: `walk-placeholder-referral.pdf` (15,000,709 bytes, a one-page placeholder letter padded with a
comment stream, sha256 `fc929bbce2cf9ba8fff9f7d12bfccd2470da8369568d184270593473587f0727`) and
`walk-placeholder-large.pdf` (21,000,694 bytes). Both generated by a script at run time.

## Step 1: the builder (web, Chromium 1440x900)

Playwright's Chromium in the `playwright` service (the Vite dev server on the container's own localhost, as the e2e
web journeys), signed in as the walk GP with password and TOTP.

| | |
| --- | --- |
| Expected | "Edit as new version" on "First visit" opens a draft; "File upload" can be dragged in; the opened card has "Accepted files" (Photos, Documents, Videos) with the W3 hint and "Number of files" with "From 1 to 10."; publishing makes version 2 active; the role's default still points at the series |
| Observed | As expected. The new item opened with Photos and Documents ticked (the 1.6 default) and Videos not; "Number of files" set to 3; the closed card shows "File upload" and "Up to 3 files" (W5). Publish asked "Publish this questionnaire?" and version 2 became active, version 1 retired. Working hours shows "Questionnaire for every booking: First visit" |
| Result | Pass |

Database (as the owner): `questionnaire` series `bc44a6c8-…` version 1 `retired`, version 2 `active`; version 2's
third item is `{"type": "attachment", "accept": ["photo", "document"], "maxFiles": 3, "linkId":
"1603b735-a93d-4d05-af02-a4ff159b1750"}`. `questionnaire_default` for the role: `series_id` `bc44a6c8-…`, unchanged
since the setup (it names the series, not a version).

Screenshots: `walk/1-01-signed-in.png`, `walk/1-02-first-visit-v1.png`, `walk/1-03-builder-file-upload-open.png` (the
builder with the item open), `walk/1-04-file-upload-card.png` (the card alone), `walk/1-05-publish-confirm.png`,
`walk/1-06-published-v2.png`, `walk/1-07-role-default.png`.

## Step 2: the patient app (Android emulator)

| Part | Expected | Observed | Result |
| --- | --- | --- | --- |
| Sign in, book | Sara finds the clinic and the GP, books a time | Signed in; "Choose a clinic" (the walk clinic at the end of a long list of e2e clinics), "Amar Mehmedović", Saturday 23:00 to 23:30, "Book"; "Your appointment is booked." with "Fill in" | Pass |
| The form | "First visit" v2 with the file item, M2 hint, M1 | The two placeholder questions, then the file item with "Photos up to 20 MB. Documents (PDF) up to 20 MB. Up to 3 files." and "Add file"; footer "Saved" | Pass |
| Source sheet (M3) | Only the sources the item accepts | "Take a photo", "Choose from photos", "Choose a document", "Cancel" (no video) | Pass |
| Camera | M18 before the system prompt, then the camera | "Allow camera access" / "Salus needs the camera…" with "Not now", "Continue"; then Android's prompt ("While using the app"); the emulator's camera app on the virtual scene; shutter, then "Done" | Pass |
| Photo states (M4) | "Uploading…", "Checking the file…", "Added" | The 64 KB photo uploaded in 75 ms (api log), scanned clean 100 ms later; the app's first status read came 2.4 s after the upload. "Uploading…" passed too fast for any screenshot or dump. "Checking the file…" was read from the accessibility tree (`Photo, 39f6…444.jpg, Checking the file…`) but had become "Added" by the next screenshot. "Added" captured | Pass (two states not captured, see text) |
| PDF states (M4) | The same three | Captured: "Uploading… 33%" with the progress bar, "Uploading… 77%", "Checking the file…" (with M10 in the footer: "Some files are still being checked. You can submit once they are ready."), "Added" | Pass |
| Remove (M6) and add again | Confirmation, the file gone, then added again | "Remove file" → "Remove this file?" / "It is deleted from your answers." → "Remove": the PDF row went (204 on `DELETE /binaries/{id}`, the object deleted, audited). Added again: "Uploading… 38%", "Uploading… 70%", "Checking the file…", "Added" | Pass |
| Over the limit (M8) | "This file is larger than 20 MB." | Shown under the item for the 21 MB PDF, no row added. The app refused it on the device: no request reached the API (see summary point 3). The server's 413 was checked through the API in step 3 | Pass |
| Submit | Confirmation, then the read-only answers (M12) | "Submit your answers?" → "Submit": "Answers submitted", the answers with "2 files", each file with "[Kind], [size]" | Pass |

The picker's copies: after the uploads the app's `cache` directory was empty (`run-as ba.salus.salus ls -laR cache`),
and the camera wrote nothing to `DCIM` or `Pictures`.

Screenshots: `walk/2-01-app-start.png`, `walk/2-02-sign-in.png`, `walk/2-03-signed-in.png`,
`walk/2-04-choose-clinic.png`, `walk/2-05-practitioner.png`, `walk/2-06-slots.png`, `walk/2-07-confirm-booking.png`,
`walk/2-08-booked.png`, `walk/2-09-first-visit-form.png`, `walk/2-10-add-file-sheet.png` (M3),
`walk/2-11-camera-rationale.png` (M18), `walk/2-12-system-camera-permission.png`, `walk/2-13-camera-app.png`,
`walk/2-14-photo-taken.png`, `walk/2-15-photo-added.png` (taken while the Submit button was still fading in),
`walk/2-16-document-picker.png`, `walk/2-17-pdf-uploading-33.png`, `walk/2-18-pdf-uploading-77.png`,
`walk/2-19-pdf-checking.png`, `walk/2-20-pdf-added.png`, `walk/2-21-remove-confirm.png` (M6),
`walk/2-22-pdf-removed.png`, `walk/2-23-pdf-readded-uploading.png`, `walk/2-24-pdf-readded-checking.png`,
`walk/2-25-pdf-readded-added.png`, `walk/2-26-too-large.png` (M8), `walk/2-27-appointment-continue.png`,
`walk/2-28-after-eicar-reopened.png` (step 3), `walk/2-29-submit.png`, `walk/2-30-submitted.png`.

## Step 3: EICAR through the API

Done before Sara submitted (an upload needs an `in-progress` response), with `curl` as Sara (bearer token from
`POST /auth/login`). The file: a 328-byte PDF whose one content stream is the EICAR test string, uncompressed,
assembled from three pieces by a script at run time in the scratchpad (sha256
`de1333ec639d867f5109146f8fd72ade31e3cca614675da83f3bb474ceda7d54`); never written into the repository.

| | |
| --- | --- |
| Expected | 201 `quarantined`; the scan rejects it; `GET /binaries/{id}` says `rejected`; `GET …/content` 404; audited: the upload, the rejection by `binary_scan`, the refused downloads |
| Observed | `POST …/items/1603b735-…/attachments` = 201 `{"status":"quarantined","size":328,…}`; about 30 ms later the worker logged `binary scan done … "result":"rejected","signature":"Eicar-Signature"` (binary id only, no title or patient). Five status reads, all 200 `"status":"rejected"`; two downloads, both 404 `{"error":"not_found"}`. Row: `status rejected`, `wrapped_key` null, `scanned_at` set. Object `12a2c998-…` no longer in the bucket. The response's `attachments` lists only the photo and the PDF |
| Result | Pass |

Audit (`audit_event` as the owner role; the agent shown as a role, ids opaque):

```
 recorded     | action | outcome | subtype | purpose | agent          | entity_reference
 20:33:07.164 | C      | 0       | create  | PATRQT  | Sara (patient) | Binary/12a2c998-7ad0-421a-956b-6ac2f2c83e01
 20:33:07.190 | D      | 0       | delete  | HOPERAT | binary_scan    | Binary/12a2c998-7ad0-421a-956b-6ac2f2c83e01
 20:33:07.191 | D      | 0       | delete  | HOPERAT | binary_scan    | QuestionnaireResponse/08a5ac06-f66e-434d-acd6-3c00926286c8
 20:33:12.313 | R      | 0       | read    | PATRQT  | Sara (patient) | Binary/12a2c998-…  (status read, 5 rows to 20:33:16.467)
 20:33:17.499 | R      | 4       | read    | PATRQT  | Sara (patient) | Binary/12a2c998-…  (refused download)
 20:33:17.514 | R      | 4       | read    | PATRQT  | Sara (patient) | Binary/12a2c998-…  (refused download)
```

Each of those also has its `QuestionnaireResponse/08a5ac06-…` row. The server's size limit, checked here too: the
21,000,694-byte PDF sent through the API = 413 `{"error":"too_large"}`, audited on the response (`C`, outcome 4,
20:33:34.470).

M7 in the app: not shown, and it cannot be for this file. The app shows M7 when a file it uploaded itself turns
`rejected` while it polls that file. A file uploaded outside the app is never in the app's list: the scan detached
it, so the response's attachments (which the app reads when the form opens) no longer name it. Reopened after the
rejection, the form showed only the photo and the PDF, both "Added", and no message
(`walk/2-28-after-eicar-reopened.png`). M7 itself is covered by the widget tests (`mobile/test/attachments_test.dart`,
"a rejected file"); seeing it on a device needs a file the app uploads that the scan rejects, which needs an EICAR file
in the emulator's storage and was not done.

## Step 4: answers and viewer (web, Chromium 1440x900)

| | |
| --- | --- |
| Expected | The appointment's "View answers" lists "2 files" with title, kind and size; the photo opens in the viewer; the PDF downloads byte-identical |
| Observed | "First visit" "Submitted Saturday 7 November 2026, 21:34"; the file item "2 files": the photo "Photo, 64 KB" and "walk-placeholder-referral.pdf" "Document, 15.0 MB", each with "Open" and "Download". The viewer: "File 1 of 2", the image decoded (1440x1920), content response 200 with `Content-Security-Policy: default-src 'none'; sandbox`. "Next file": "File 2 of 2" for the PDF shows W12 "This file cannot be shown here." with "Download" (headless Chromium has no PDF viewer; the `<object>` made no request). "Download" saved `walk-placeholder-referral.pdf`, 15,000,709 bytes, sha256 `fc929bbce2cf9ba8fff9f7d12bfccd2470da8369568d184270593473587f0727`: identical to the file pushed to the emulator |
| Result | Pass, with finding 1 (the photo is cut off in the viewer) |

O23 (the PDF in Chrome's own viewer inside the `<object>` under the CSP with `sandbox`): not done. Playwright's
Chromium in the container is headless and has no PDF viewer, so it can only show the W12 fallback; Google Chrome is
not in the Playwright image (4.1). This needs the owner in desktop Chrome: open the appointment, "View answers", open
the PDF.

Audit: each GP content read is a `R` row by "Amar (GP)" with purpose `TREAT` on the file (20:34:58 to 20:35:25, one per
request).

Screenshots: `walk/4-01-appointment.png`, `walk/4-02-answers.png`, `walk/4-03-viewer-photo.png`,
`walk/4-04-viewer-pdf-chromium.png`.

## Step 5: object store, metadata, logs

Objects read straight from Garage (`rclone` in a container on the compose network, the `TEST_S3_*` credentials,
bucket `salus-binaries`); the plaintext downloaded through the API as Sara. Checked with a Python script (`python3 -I`):

| File | Plaintext | Stored object | Expected size (+16 B per 64 KiB segment) | Equal to plaintext | Magic at start | Plaintext's first 4 KiB inside |
| --- | --- | --- | --- | --- | --- | --- |
| Photo `6c7db06c-…` | 63,531 B, sha256 `e5ee51b9…69443`, SHA-1 `c03c1ff7…ac41` (= the row's `sha1`) | 63,547 B, starts `61cf06c07c4dbb6f` | 63,531 + 16 x 1 = 63,547 | no | no JPEG `ffd8ff`, no "Exif" anywhere | no |
| PDF `edcfe88c-…` | 15,000,709 B, sha256 `fc929bbc…f0727` | 15,004,373 B, starts `9dda2784ee8d8952` | 15,000,709 + 16 x 229 = 15,004,373 | no | no `%PDF` anywhere | no |

The removed PDF's object (`a5a1aa9c-…`) and the rejected file's (`12a2c998-…`) are not in the bucket.

The downloaded photo's metadata: its JPEG segments before the scan are only `DQT, DQT, SOF0, DHT x4`: no APP1 (no
Exif at all, so no GPS and no orientation tag), no XMP, no IPTC/Photoshop, no JFIF. The picture is 1440x1920, upright.
Pass: no EXIF GPS and nothing but what is needed to show it.

Logs: `docker compose logs api livekit worker` (1,199 lines covering the whole walk, taken before step 6 restarted the
services) hold none of: `walk-placeholder-referral`, `walk-placeholder-large`, `walk-eicar-test`, the photo's title
(`39f6ccbd-e0b7`), "First visit", the accounts' e-mails (`walk-sara`, `walk-amar`), "Sara", "Mehmedovi", the typed
answer. The api lines carry method, route template, status, duration and request id; the worker's scan lines the
binary id, result, timing, and on the rejection the signature name. Pass.

## Step 6: regression

`E2E=1 ONLY=e2e bash checks.sh <log dir>` (the session's script: `up -d --build --wait` without `--remove-orphans`,
then the e2e commands with `--no-deps` in the `playwright` service):

```
PASS e2e-up       21:36:59 → 21:37:06
PASS e2e-install  21:37:06 → 21:37:08
PASS e2e          21:37:08 → 21:37:45
ok  github.com/Shmeeshmee/salus/test/e2e  33.349s
e2e-livekit-log: LiveKit's log (523 lines) holds none of the run's 22 synthetic values and each of its 2 rooms
```

Result: pass. Note: its `up -d --build --wait` recreated `migrate`, `agent`, `api`, `scheduler`, `worker` and
`notifier` (same code, rebuilt images); `postgres`, `redis`, `objectstore`, `clamd`, `livekit`, `mailpit` and `landing`
kept running.

## What the walk changed outside this file

- Dev database and bucket: the walk's synthetic data (above), left in place.
- Docker: `salus-demo-emu` started once (it exited by itself, as described); the throwaway `salus-walk-emu` and
  `salus-walk-apk` containers removed. The step 6 recreation of the app services.
- Repository: only this file and `docs/attachments/walk/`. The temporary setup test and Playwright specs were deleted
  before step 6.
