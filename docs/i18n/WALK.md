# Languages: acceptance walk

The walk of SPEC.md §6, done by package 3.1 on 2026-11-08 during the unattended run, against the dev stack built
from `feat/i18n` after wave 2. Synthetic accounts and placeholder questionnaire wording only. Screenshots and
e-mail texts are in `docs/i18n/walk/`.

## How it was walked

- **Web (steps 1 to 4, 7, 8):** no browser pane was available, so the web app was walked by a Playwright script in
  Chromium (the `playwright` service, the Vite dev server of `web/` proxying the running API, as the end-to-end
  web journeys run). A Go driver made the synthetic clinics, accounts and bookings through the API and read the
  e-mails from Mailpit's API. Desktop is 1280 px wide; the `-390` files are the same screen at 390 px. The two
  driver files were run from the worktree and not committed (they are a one-off; the committed end-to-end
  journeys in `test/e2e/language_test.go` and `test/e2e/web/language.spec.ts` check the same behaviour on every
  `make e2e`).
- **Mobile (steps 5, 6, 8, 9):** the owner's emulator was down, so **no mobile step was walked on a real device or
  an emulator**. The screens were rendered by Flutter widget tests with the real fonts (Roboto from the SDK, the
  bundled Noto Sans Arabic), as 1.7's golden screenshots are, on a 390 × 844 phone, with the app's fake API (the
  fixtures of `mobile/test/`). That shows the catalogs, the layout and the direction, not the device's own
  behaviour (the device language list, the keyboard, the system permission dialogs). The red "DEBUG" ribbon is
  Flutter's debug banner of widget tests. **The owner walks steps 5, 6, 8 and 9 on a phone or emulator before
  the feature is called done.**
- **E-mails (steps 4, 5, 9):** read from Mailpit's API after the booking or invite through the running stack;
  the clinic's random name, the addresses and the invite token are replaced with `<…>` in the saved texts.

## Steps

| Step | What | Result | Evidence |
| --- | --- | --- | --- |
| 1 | A browser in `sr-Cyrl` opens the web app signed out | Pass: "Пријава", "Е-пошта", "Лозинка", "Пријави се", the picker "Језик" on "Користи језик прегледача"; `<html lang="sr-Cyrl">` | `walk/step1-sign-in-sr-Cyrl.png`, `walk/step1-sign-in-sr-Cyrl-390.png` |
| 2 | The practitioner picks Hrvatski, signs out, signs in from another browser in English | Pass: "Jezik je spremljen." at once; the new browser's English sign-in screen leads to the app in Croatian ("Kalendar", "Upitnici", "Odjavi se") | `walk/step2-calendar-en-before.png`, `walk/step2-picked-hr.png`, `walk/step2-signed-out.png`, `walk/step2-other-browser-hr.png`, `walk/step2-other-browser-hr-390.png` |
| 3 | "Use the browser's language" clears it | Pass: "Language saved." and the app in English; the next sign-in, from a fresh browser in English, stays English | `walk/step3-browser-language.png`, `walk/step3-next-sign-in-en.png` |
| 4 | The admin sets the clinic's default language to English and invites a practitioner | Pass: "Saved."; the invite e-mail is the English one ("Your invitation to Salus") | `walk/step4-clinic-default-en.png`, `walk/step4-clinic-default-en-390.png`, `walk/step4-invite-sent.png`, `walk/step4-invite-email-en.txt` |
| 5 | A phone set to German registers; the app and the booking confirmation are in German | Pass (rendered, not on a device): sign-in, registration, home, My appointments, clinics, practitioners, times, booking and consent in German; the confirmation e-mail of a patient registered with `detectedLanguage` `de` is German ("Ihr Termin ist gebucht", "Datum und Uhrzeit: Sonntag, 8. November 2026, 03:00 Uhr") | `walk/step5-sign-in-de.png`, `walk/step5-register-de.png`, `walk/step5-home-de.png`, `walk/step5-appointments-de.png`, `walk/step5-clinics-de.png`, `walk/step5-practitioners-de.png`, `walk/step5-slots-de.png`, `walk/step5-book-de.png`, `walk/step5-consent-de.png`, `walk/step5-booking-email-de.txt` |
| 6 | The same phone set to Arabic: right to left, back arrows point right, times and phone numbers left to right, nothing cut off | Pass (rendered, not on a device): every screen right to left with the back arrow pointing right; times read left to right ("14:00 إلى 14:30", the 14:00 slot); the e-mail field is typed left to right (1.7's test asserts it); nothing cut off at 390 px. No phone number is on these screens (the profile form was not rendered), so phone numbers wait for the device walk. The device check and call screen, and the questionnaire choice, in Arabic too | `walk/step6-sign-in-ar.png`, `walk/step6-register-ar.png`, `walk/step6-home-ar.png`, `walk/step6-appointments-ar.png`, `walk/step6-clinics-ar.png`, `walk/step6-practitioners-ar.png`, `walk/step6-slots-ar.png`, `walk/step6-book-ar.png`, `walk/step6-consent-ar.png`, `walk/step6-device-check-ar.png`, `walk/step6-call-ar.png`, `walk/step6-questionnaire-choice-ar.png`, `walk/step6-questionnaire-in-english-ar.png` |
| 7 | The doctor builds a questionnaire in Bosnian, starts an English translation, tries to publish it half done, finishes and publishes | Pass: "English: 2 of 5 texts translated"; Publish refused with "The English translation is not complete. Finish it or remove it to publish." and "Missing in English" on the untranslated item; once finished "English: complete" and published without the B5 prompt | `walk/step7-original-bs.png`, `walk/step7-translation-half.png`, `walk/step7-translation-half-390.png`, `walk/step7-publish-refused.png`, `walk/step7-missing-in-english.png`, `walk/step7-translation-complete.png`, `walk/step7-published.png` |
| 8 | The German patient opens it: Bosnian and English, English preselected, answers in English; the practitioner sees "Filled in in English" and the English questions | Pass: the app (rendered) says "Dieser Fragebogen ist nicht auf Deutsch verfügbar." with Bosanski and English and English preselected, then the questions in English; the patient answered through the API in English; the web app shows "Filled in in English", the English questions and answers, and the Bosnian questions with "Show the original questions" | `walk/step8-questionnaire-choice-de.png`, `walk/step8-questionnaire-in-english-de.png`, `walk/step8-filled-in-english.png`, `walk/step8-filled-in-english-390.png`, `walk/step8-original-questions.png` |
| 9 | An Italian patient with a Bosnian-only questionnaire is told it is only in Bosnian and fills it in | Pass (rendered): "Questo questionario non è disponibile in Italiano." and "È disponibile in: Bosanski.", Bosanski preselected, then the questionnaire in Bosnian; the confirmation e-mail is Italian; the end-to-end journey answers in Bosnian | `walk/step9-questionnaire-choice-it.png`, `walk/step9-questionnaire-in-bosnian-it.png`, `walk/step9-booking-email-it.txt` |
| 10 | Every catalog passes `scripts/i18n-check.sh`; `TRANSLATIONS.md` lists every language as `drafted` | Pass: the check passes; every language but the English source is `drafted`, with its count of `legal` strings | `docs/i18n/TRANSLATIONS.md` |

The mobile questionnaire screenshots use the test fixtures' placeholder questionnaire, whose translated texts are
written `<tag>: <original>`; the web ones use the questionnaire built in step 7.

## Noticed on the way, for the native-speaker review

- hr calendar: "Nema termina ovaj tjedan." reads better as "Nema termina ovog tjedna." (genitive).
- it, de, fr: the language's own name stays capitalised inside a sentence ("non è disponibile in Italiano", "Weiter
  auf English"), as COPY.md L1 decides (the name is never changed); a reviewer may prefer it otherwise.
- ar, the practitioner's line on the times screen: "أمراض القلب, Synthetic clinic Čaršija" joins the specialty and the
  clinic with a Latin comma; an Arabic reader may want "،". The line is joined in code (`mobile/lib/slots_screen.dart`,
  `join(', ')`), against SPEC §3 "No string is built by joining pieces": outside 3.1's files, left for a fix.
- The booking screen and the questionnaire choice are centred vertically on the phone, in every language (layout,
  not a translation matter).

## Waits on the owner

- Steps 5, 6, 8 and 9 on a real phone or emulator (the emulator was down during the run), Arabic included.
- The native-speaker review of every language and the lawyer's review of the `legal` strings (SPEC §8).
