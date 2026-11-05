# Video consultations: approved text

Approved by the owner, 2026-11-05, with the owner's changes. Packages 6 and 7 put these strings into
`web/src/messages.ts` and `mobile/lib/messages.dart` exactly as written here. A change to any of them goes back
to the owner first. W is the web app (practitioners), M the mobile app (patients); [Name] is filled in from our
API.

## Join button
- W1 / M1: "Join consultation"
- W2 / M2: "You can join from 10 minutes before the start."

## Device check
- W3 / M3: "Check your camera and microphone"
- W4: "The patient will see and hear you once you join."
- M4: "The practitioner will see and hear you once you join."
- W5 (web only): "Camera", "Microphone"
- W6 / M5: "No sound detected. Check that your microphone is on and not muted."
- W7 / M6: "Join with camera off", "Join with microphone off"
- W8 / M7: "Join"
- W9 / M8: "Joining…"

## Denied permission
- W10: "Your browser is not allowing the camera. Allow it in the site settings, then reload the page. You can still join without video."
- W11: "Your browser is not allowing the microphone. Allow it in the site settings, then reload the page. Without it, the patient cannot hear you."
- M9: "Salus does not have access to your camera. You can allow it in your device settings, or join without video."
- M10: "Salus does not have access to your microphone. You can allow it in your device settings. Without it, the practitioner cannot hear you."
- M11: "Open settings"
- W12 / M12: "No camera found." / "No microphone found."

## Waiting
- M13: "Waiting for the practitioner"
- M14: "[Practitioner name] will join you here. Keep this screen open."
- W13: "Waiting for the patient"
- W14: "[Patient name] has not joined yet."

## In the call
- W15 / M15: "Mute microphone", "Unmute microphone", "Turn camera off", "Turn camera on", "Leave"
- W16 / M16: "[Name]'s camera is off"
- W17 / M17: "[Name] is muted"
- W18 / M18: "[Name] has left the consultation. They can join again until the consultation ends."
- W18a (web, instead of W18 when the patient was removed after withdrawing consent): "The patient withdrew consent for this clinic, so the consultation has ended."
- W18b (web, instead of W18 when the patient was removed after a cancellation): "This appointment was cancelled, so the consultation has ended."

## End of consultation (web)
- W19: "End consultation"
- W20: "End the consultation?"
- W21: "The patient will be disconnected and cannot join again. The appointment will be marked as fulfilled."
- W21b: "The patient has not joined. The appointment will be marked as a no-show."
- W22: "End consultation", "Cancel"
- W23: "You left the consultation. You can join again until it ends." with the button "Join again"

## Reconnecting
- W24 / M19: "Connection lost. Reconnecting…"
- W25 / M20: "The connection could not be restored." with the button "Try again"
- W26 / M21: "Reconnected"

## Ended
- W27 / M22: "The consultation has ended"
- M23: "You can close this screen."
- W28: "The appointment is marked as fulfilled." / "The appointment is marked as a no-show."
- W29 / M24: "The consultation reached its time limit and has ended."

## Practitioner did not join
- M25: "The practitioner did not join"
- M26: "The time to join this consultation has passed. The clinic can see that you joined. The clinic may contact you, or you can book a new appointment."
- W39 (web, `practitioner_did_not_join`): "The time to join this consultation has passed."

## Access ended (mobile)
- M27: "You withdrew your consent for this clinic, so you have left the consultation."
- M28: "This appointment was cancelled, so the consultation has ended."

## Errors
- W30 / M29 (`join_window_not_open`): "This consultation opens 10 minutes before the start."
- W31 / M30 (`consultation_ended`): "This consultation has ended."
- M31 (`practitioner_did_not_join`): M25 and M26.
- M32 (`consent_required`, patient): "Joining needs your consent for this clinic."
- W32 (`consent_required`, practitioner): "The patient's consent for this clinic is not active, so the consultation cannot take place."
- W32b (`consultation_not_started`): "You have not joined this consultation yet, so there is nothing to end."
- W33 / M33 (not found): "This consultation is not available."
- W34 / M34 (cannot connect): "The video connection could not be started. Check your connection and try again."

## Appointment detail (web)
- W35: "Consultation"
- W36: "Not started", "Open", "Ended"
- W37: "Patient joined at [time]" / "Patient did not join"
- W38: "Practitioner joined at [time]" / "Practitioner did not join"

## System permission text (mobile)
- iOS `NSCameraUsageDescription`: "Salus uses the camera so the practitioner can see you during a video consultation. The consultation is not recorded."
- iOS `NSMicrophoneUsageDescription`: "Salus uses the microphone so the practitioner can hear you during a video consultation. The consultation is not recorded."
- Android, the app's explanation before the system prompt (M35): "To join the video consultation, Salus needs your camera and microphone. The consultation is not recorded."
- Browsers show their own prompt; nothing is added before it beyond W3 and W4.

"The consultation is not recorded" must change if recording is ever added (ADR 0006).
