# Questionnaire attachments: text for review

Status: approved by the owner as written, 2026-11-07. Packages 1.6 and 1.7 put these strings into
`web/src/messages.ts`, `mobile/lib/messages.dart`, `mobile/ios/Runner/Info.plist` and the Android rationale
exactly as approved here. A change to any string goes back to the owner first; a line marked "added by the
session, not yet reviewed" was added while the owner was away and waits for that review. W is the web app (practitioners and admins), M the
mobile app (patients). [Brackets] are filled in by the app. Existing strings that are reused are named, not
repeated (`docs/questionnaire/COPY.md`, `docs/consultation/COPY.md`).

## Builder (web)

- W1 (item type, in the palette and in the type dropdown of an opened card): "File upload"
- W2 (on the opened card): "Accepted files", with the choices "Photos", "Documents", "Videos"
- W3 (hint under W2): "Photos: JPEG or PNG, up to 20 MB. Documents: PDF, up to 20 MB. Videos: MP4, up to 200 MB."
- W4 (on the opened card): "Number of files", with the hint "From 1 to 10."
- W5 (on the closed card, after the type): "1 file"; "Up to [n] files"
- W6 (item errors, on the item): "Choose at least one kind of file."; "The number of files must be from 1 to 10."

## Appointment detail, submitted answers (web)

- W7 (the answer of a file upload item): "1 file"; "[n] files". With no file, "Not answered" (questionnaire W38).
- W8 (a file in the list): "[Title]", and under it "[Kind], [size]", the kind one of "Photo", "Document", "Video",
  the size "[n] KB" or "[n.n] MB"
- W9 (actions on a file): "Open", "Download"
- W10 (viewer): "Close", "Previous file", "Next file", "File [n] of [total]"
- W11 (viewer, labels for screen readers): "Photo: [Title]"; "Document: [Title]"; "Video: [Title]"
- W12 (viewer, when the browser cannot show the file): "This file cannot be shown here." with the button
  "Download"
- W13 (errors): not found: "This file is no longer available."; any other failure: "The file could not be
  loaded." with the button "Try again"

## Filling in (mobile)

- M1 (button on a file upload item): "Add file"
- M2 (hint under a file upload item, the sentences for the kinds it accepts, in this order): "Photos up to
  20 MB."; "Documents (PDF) up to 20 MB."; "Videos up to 200 MB."; then "1 file." or "Up to [n] files."
- M3 (sheet after "Add file", only the sources the item accepts): "Take a photo", "Record a video", "Choose from
  photos", "Choose a document", "Cancel"
- M4 (a file's status, under its name): "Preparing…" (while the app converts it); "Uploading… [n]%"; "Checking
  the file…"; "Added"
- M5 (upload failed): "Not uploaded. Check your connection." with the buttons "Try again", "Remove"
- M6 (removing a file): the button "Remove file" (also the screen-reader label of the row's remove icon); "Remove
  this file?" / "It is deleted from your answers." with the buttons "Remove", "Cancel"
- M7 (the scan rejected a file): "This file did not pass the safety check and was deleted. Choose another file."
- M8 (errors from the server): `too_large`: "This file is larger than [max] MB."; `unsupported_type`: "This kind
  of file cannot be added to this question."; `too_many_files`: "You can add up to [n] files to this
  question."; `response_full`: "The files in this questionnaire have reached 500 MB. Remove a file to add
  another."; any other failure: "The file could not be uploaded. Try again."
- M9 (the app could not convert or read the file): "This file could not be prepared for upload. Choose another
  file."
- M10 (submit while a file is being checked, `attachments_pending`): "Some files are still being checked. You can
  submit once they are ready."
- M11 (leaving while a file uploads): "A file is still uploading. Leave anyway? The upload stops." with the
  buttons "Leave", "Stay"
- M12 (read-only view after submitting): "1 file"; "[n] files"; each file as "[Title]" and "[Kind], [size]",
  with the kinds and sizes of M13. With no file, "Not answered" (questionnaire M14).
- M13 (kinds and sizes, in M4 rows and M12): "Photo", "Document", "Video"; "[n] KB", "[n.n] MB"
- M14 (screen-reader labels): the progress of an upload: "Uploading [Title], [n] percent"; a file row: "[Kind],
  [Title], [status]"

A required file upload item with no file shows "This question is required." (questionnaire M10).

## Permissions (mobile)

The app already asks for the camera and the microphone for video consultations. iOS keeps one purpose string
per permission, so the two existing strings are replaced by M15 and M16, which cover both uses.

- M15 (iOS, `NSCameraUsageDescription`, replaces the consultation's string; its last sentence changed by SAL-12, a
  draft waiting for the owner): "Salus uses the camera so the
  practitioner can see you during a video consultation, and to take the photos and videos you add to a
  questionnaire. The consultation is not recorded as audio or video."
- M16 (iOS, `NSMicrophoneUsageDescription`, replaces the consultation's string; its last sentence changed by
  SAL-12, a draft waiting for the owner): "Salus uses the microphone so
  the practitioner can hear you during a video consultation, and to record sound with the videos you add to a
  questionnaire. The consultation is not recorded as audio or video; if you agree, what is said is transcribed and
  translated."
- M17 (iOS, `NSPhotoLibraryUsageDescription`, new): "Salus opens your photo library only to add the photos and
  videos you choose to a questionnaire."
- M18 (Android, the app's own explanation before the system asks for the camera; Android writes the system prompt
  itself): "Allow camera access" / "Salus needs the camera to take the photo or video you add to this
  questionnaire." with the buttons "Continue", "Not now"
- M19 (camera refused, both platforms): "Salus does not have access to your camera. You can allow it in your
  device settings, or choose a file instead." with the buttons "Open settings" (consultation M11) and "Cancel"
- M20 (photo library refused, iOS): "Salus does not have access to your photos. You can allow it in your device
  settings." with the buttons "Open settings" (consultation M11) and "Cancel"

Never in any of these: a questionnaire's title, an item's text or a file's name, except where [Title] says so.

## Added after the acceptance walk

M21 and M22: approved by the owner, 2026-11-07.

Added on the owner's acceptance-walk findings of 2026-11-07 (WALK.md, point 2), in place of the picker's cache file
name. Files from the photo library or the document picker keep their own name.

- M21 (the title of a photo taken with the camera, shown in M4 rows and M12 and sent as the file's name, so it is
  W8's [Title] on the web): "Photo, [date], [time]", the date as the app writes dates (Saturday 7 November 2026) and
  the time in 24 hours (21:24), in the patient's local time when the photo was taken (approved by the owner, 2026-11-07)
- M22 (the title of a video recorded with the camera, as M21): "Video, [date], [time]" (approved by the owner, 2026-11-07)
