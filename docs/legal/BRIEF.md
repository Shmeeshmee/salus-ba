# Legal questions for counsel

Salus is a telemedicine platform for private clinics in Bosnia and Herzegovina: patients find a clinic, book,
fill in questionnaires and attend video consultations in a mobile app, and practitioners and clinic admins work
in a web app; the clinical record is kept as FHIR resources and the platform is designed around the GDPR. Our
privacy notes treat each clinic as the controller of its patients' data and Salus as a processor acting on the
clinic's behalf (Q37 asks you to confirm this). The questions below were collected on 2026-11-08 from the
feature SPECs and the privacy notes, and each one links to the section it comes from. Where a question states a
period or a rule, that is the default the software applies today, not a legal conclusion; please answer by
question number.

## 1. Retention periods

**Q1.** For a private clinic in Bosnia and Herzegovina, how long must the medical record of a remote
consultation be kept, and from when does that period run?
Several records are set to "kept with the clinical record" with no period yet: the appointment, the consent,
the consultation's `Encounter`, submitted questionnaire answers and their attached files.
Sources: [booking §2](../booking/SPEC.md#2-settled-decisions), [booking §4](../booking/SPEC.md#4-personal-data),
[booking §7](../booking/SPEC.md#7-before-production), [consultation §5](../consultation/SPEC.md#5-personal-data),
[consultation §8](../consultation/SPEC.md#8-before-production), [questionnaire §5](../questionnaire/SPEC.md#5-personal-data),
[questionnaire §8](../questionnaire/SPEC.md#8-before-production), [attachments §5](../attachments/SPEC.md#5-personal-data),
[attachments §8](../attachments/SPEC.md#8-before-production).

**Q2.** Is it right to keep the record of a patient's consent (given and withdrawn, with dates) for as long as
the clinical record, as proof of the decision, or should it follow a different period?
The consent row is kept as proof under Art. 7(1) GDPR and has no deletion rule.
Sources: [booking §4](../booking/SPEC.md#4-personal-data), [booking §7](../booking/SPEC.md#7-before-production).

**Q3.** Is ten years an appropriate retention period for the audit log of who accessed which record, and is
there a minimum or maximum the law sets?
Every read or write of personal data writes an append-only audit event, removed by whole time ranges after ten
years.
Sources: [accounts §4](../accounts/SPEC.md#4-personal-data), [DPIA, accounts package 1](../privacy/dpia.md#package-1-privacy-foundation).

**Q4.** May the patient profile (name, birth date, optional phone) be kept for as long as the patient's account
exists, and what may be deleted when the patient closes the account or asks for erasure while appointments and
encounters still reference the profile?
Account closure and erasure are planned for a later subject-rights feature, limited by medical-record
retention.
Sources: [booking §4](../booking/SPEC.md#4-personal-data), [DPIA, booking package 2](../privacy/dpia.md#package-2-patient-profile).

**Q5.** Are these periods for login data acceptable: accounts kept while they exist, sessions deleted 30 days
after they expire or are revoked, and staff invitations deleted 30 days after they are accepted or expire?
Sessions hold no IP address or device data; invitations hold an e-mail address and, for practitioners, a name.
Sources: [accounts §4](../accounts/SPEC.md#4-personal-data).

**Q6.** When a practitioner leaves a clinic, may their name, role and working schedule be restricted rather
than deleted for as long as clinical records reference them, and for how long?
Practitioner, role and schedule records are set to "restrict" after leaving.
Sources: [accounts §4](../accounts/SPEC.md#4-personal-data), [booking §4](../booking/SPEC.md#4-personal-data).

**Q7.** Is there any obligation to keep these short-lived records longer than we do: free appointment slots
(deleted 30 days after their end), technical video-session rows (deleted 30 days after the call), questionnaires
never submitted (deleted once the appointment starts) and files the virus scanner rejected (deleted after 24
hours)?
None of these is part of the clinical record in our design; each is deleted by a scheduled job.
Sources: [booking §4](../booking/SPEC.md#4-personal-data), [consultation §2](../consultation/SPEC.md#2-settled-decisions),
[consultation §5](../consultation/SPEC.md#5-personal-data), [questionnaire §5](../questionnaire/SPEC.md#5-personal-data),
[attachments §5](../attachments/SPEC.md#5-personal-data).

**Q8.** When a future appointment is cancelled, or the patient withdraws consent before it, may we delete the
patient's questionnaire answers and attached files, including answers already submitted to the clinic, or must
submitted answers be kept as part of the record?
Today they are deleted, submitted or not, because the appointment they prepared will not take place (decision
D8).
Sources: [questionnaire §3](../questionnaire/SPEC.md#3-answered-decisions), [questionnaire §8](../questionnaire/SPEC.md#8-before-production),
[attachments §3](../attachments/SPEC.md#3-open-points), [attachments §8](../attachments/SPEC.md#8-before-production).

## 2. Consent versus care: GDPR Art. 9(2)(h)

**Q9.** Is it correct to treat a patient's profile and appointments as health data (Art. 9), since they reveal
that a person seeks care and with whom?
The design treats them so, and treats even a patient account on a telemedicine service as sensitive.
Sources: [booking §3](../booking/SPEC.md#3-lawful-basis), [accounts §3](../accounts/SPEC.md#3-lawful-basis).

**Q10.** Is our split of lawful bases right: the care record rests on Art. 9(2)(h) with Art. 6(1)(b) and the
clinic's record-keeping duties, while sharing the profile with a clinic before care, and each new booking, rest
on explicit consent under Art. 9(2)(a)?
This split decides what a withdrawal of consent stops and what the clinic keeps.
Sources: [booking §3](../booking/SPEC.md#3-lawful-basis), [booking §7](../booking/SPEC.md#7-before-production),
[DPIA, booking package 3](../privacy/dpia.md#package-3-consent-and-access).

**Q11.** Should booking an appointment require consent at all, given that care itself rests on Art. 9(2)(h)?
The owner raised this as an open question; today a patient cannot book without giving consent.
Sources: [booking §7](../booking/SPEC.md#7-before-production).

**Q12.** After a patient withdraws consent, may the clinic keep reading the minimal record of appointments that
took place or were due (name, times, status), while losing the profile and every future appointment?
Withdrawal takes effect at once and cancels future appointments; past ones stay readable in this minimal form.
Sources: [booking §3](../booking/SPEC.md#3-lawful-basis), [DPIA, booking package 9](../privacy/dpia.md#package-9-mobile-app).

**Q13.** Is it right that joining a video consultation requires the patient's active consent, on the ground
that the practitioner sees and hears the patient, while the consultation itself rests on Art. 9(2)(h)?
A withdrawal during a call removes the patient from it; the fact and times of the consultation are kept as the
care record.
Sources: [consultation §4](../consultation/SPEC.md#4-lawful-basis), [consultation §8](../consultation/SPEC.md#8-before-production).

**Q14.** May questionnaire answers be saved and submitted on the basis of care (Art. 9(2)(h)) with the consent
for sharing before care, as designed, and may the appointment's practitioner still read submitted answers after
the patient withdraws consent, as part of the care record (decision D9)?
Saving and submitting require active consent; the questionnaire's title alone can reveal a condition.
Sources: [questionnaire §3](../questionnaire/SPEC.md#3-answered-decisions), [questionnaire §4](../questionnaire/SPEC.md#4-lawful-basis),
[questionnaire §8](../questionnaire/SPEC.md#8-before-production).

**Q15.** Do files a patient attaches to a questionnaire (photos, documents, short videos) rest on the same basis
as the answers, given that they often show more than the patient meant to share?
Uploading requires active consent; the app strips location and other metadata before upload, and files are
used for the patient's care only.
Sources: [attachments §4](../attachments/SPEC.md#4-lawful-basis), [attachments §8](../attachments/SPEC.md#8-before-production).

**Q16.** For the planned live transcription of consultations, would care (Art. 9(2)(h)) be a sufficient basis,
or would it need the patient's explicit consent, and what changes if the speech-to-text engine is hosted by a
third party?
Transcription is not built yet and needs its own SPEC; a hosted engine would be a processor holding patient
audio.
Interim choice (owner, 2026-11-09, pending this answer): the transcription agent joins and transcribes only while
the patient's existing care consent with the clinic is active, the same consent that gives the clinic access to the
encounter, and leaves the call as soon as it is withdrawn (SAL-4, [plan, Gates](../plans/consult-intelligence.md#gates-before-real-engines-are-wired-in)).
The only engine enabled is a fake that sends audio nowhere; Azure AI Speech is built but not enabled.
Sources: [consultation §7](../consultation/SPEC.md#7-up-next), [processors.md, pending](../privacy/processors.md#pending).

## 3. Consent, privacy and permission texts in each language

**Q17.** Does the consent screen, the dialog for withdrawing consent and the privacy notice tell patients what
the law requires for explicit, informed consent, and can you review their wording?
The consent screen says what is shared, who sees it, for what, who is responsible (the clinic, with Salus
processing on its behalf), how to withdraw, and links to the privacy notice, which is not written yet.
Sources: [booking §7](../booking/SPEC.md#7-before-production), [DPIA, booking package 9](../privacy/dpia.md#package-9-mobile-app).

**Q18.** Who must write and publish the privacy notice: each clinic as controller, Salus, or both, and must it
exist in every language the patient app offers?
The app links to one privacy notice address today (a placeholder); the plan is a published notice per patient
language.
Sources: [booking §7](../booking/SPEC.md#7-before-production), [i18n §8](../i18n/SPEC.md#8-before-production).

**Q19.** Can you review, or name who should review, the strings tagged `legal` in each of the ten languages:
Bosnian, Croatian, Serbian (Latin and Cyrillic) and English for staff and patients, and German, Arabic,
Turkish, Italian and French for patients?
There are 36 per staff language (6 web, 30 mobile) and 30 per patient-only language: consent, withdrawal and
cancellation effects, privacy-notice link text and permission explanations, all drafted by machine and not yet
reviewed by a native speaker.
Sources: [i18n §3, catalogs](../i18n/SPEC.md#catalogs), [i18n §8](../i18n/SPEC.md#8-before-production),
[TRANSLATIONS.md](../i18n/TRANSLATIONS.md#review-rules).

**Q20.** Which legal terms should the translations use for "consent" and for the party "responsible" for the
data?
The drafts use "saglasnost" (bs, sr), "privola" (hr) and "onay" (tr, where "rıza" may be preferred), and
"titolare del trattamento" (it) and "verantwortlich" (de) for "responsible".
Sources: [i18n PROGRESS, package 1.7](../i18n/PROGRESS.md#17-mobile-every-screen-in-ten-languages-1),
[i18n §8](../i18n/SPEC.md#8-before-production).

**Q21.** Does the sentence in the booking confirmation e-mail, "You can cancel it up to 24 hours before it
starts.", state the cancellation rule correctly in each of the ten languages?
E-mail templates carry no `legal` tag, so this one sentence with legal weight is listed for review separately.
Sources: [i18n §8](../i18n/SPEC.md#8-before-production), [TRANSLATIONS.md](../i18n/TRANSLATIONS.md).

**Q22.** Must the explanations shown before the iOS and Android camera, microphone, photo library and document
picker prompts say anything in particular, beyond why the app needs that access?
They are part of the `legal` strings in Q19, and still need wording for the Android explanation screen and for
Bluetooth headsets.
Sources: [consultation §7](../consultation/SPEC.md#7-up-next), [consultation §8](../consultation/SPEC.md#8-before-production),
[attachments §8](../attachments/SPEC.md#8-before-production).

## 4. E-mail processors

**Q23.** Is Amazon Web Services' data processing agreement for SES, used in an EU region, sufficient for sending
our account and appointment e-mails, and what should we check (sub-processors, transfers) before signing it?
SES is the chosen sender (ADR 0007) and is not wired in; nothing can be sent outside local development until
the DPA is signed and listed.
Sources: [processors.md, pending](../privacy/processors.md#pending).

**Q24.** Appointment e-mails reveal that a person has an appointment with a named clinic, at a date and time,
and may say a questionnaire is waiting: is that content acceptable by e-mail, and is the contract with the
patient (Art. 6(1)(b)) enough, given the processors list treats it as health data?
E-mails never name the practitioner, specialty, reason or any clinical content, and go only to the patient.
Sources: [booking §3](../booking/SPEC.md#3-lawful-basis), [processors.md, pending](../privacy/processors.md#pending),
[questionnaire §5](../questionnaire/SPEC.md#5-personal-data).

**Q25.** Can you confirm that Mailpit, the local mail catcher used only in development with synthetic data,
needs no DPA and no entry as a processor?
It runs inside our own development stack and receives no real personal data.
Sources: [processors.md, not processors](../privacy/processors.md#not-processors).

## 5. Showing ICD-10 and ATC codes from Wikidata

**Q26.** Does showing ICD-10 and ATC codes, taken from Wikidata as identifiers, next to our own labels need
anything from WHO (ICD-10, CC BY-ND 3.0 IGO) or from the WHO Collaborating Centre (the ATC/DDD index, which
forbids commercial copying)?
We take no WHO text and nothing from the ATC/DDD index, only the codes as Wikidata (CC0) records them; nothing
reaches production before this check.
Sources: [terminology §7](../terminology/SPEC.md#7-before-production), [SOURCES.md, for the lawyer](../terminology/SOURCES.md#for-the-lawyer-specmd-7),
[SOURCES.md, not used](../terminology/SOURCES.md#not-used).

**Q27.** Is our selection of about 5,300 Wikidata items, chosen by those classifications' identifiers, free of
any database right of the classifications' owners?
The selection and ranking are ours; the values are Wikidata's under CC0.
Sources: [SOURCES.md, for the lawyer](../terminology/SOURCES.md#for-the-lawyer-specmd-7).

**Q28.** Can we rely on Wikidata's CC0 release for commercial use of the labels, synonyms and codes we copied,
with no attribution required?
Wikidata states that all structured data is released under CC0; we take no text from its other namespaces.
Sources: [terminology §3, data](../terminology/SPEC.md#data), [SOURCES.md, used](../terminology/SOURCES.md#used).

**Q29.** Does using Terminologia Anatomica (TA98) identifiers to select anatomy terms, and TA98's Latin names as
the curator's reference, need any permission, and does the same question arise for the ICD-10-PCS and ICD-9-CM
procedure codes we take from Wikidata?
`SOURCES.md` records the licences of Wikidata, WHO ICD and the ATC index, but not of TA98, ICD-10-PCS or
ICD-9-CM.
Sources: [terminology §3, data](../terminology/SPEC.md#data), [terminology §7](../terminology/SPEC.md#7-before-production),
[SOURCES.md, used](../terminology/SOURCES.md#used).

**Q30.** Do we need a SNOMED CT licence to code practitioners' specialties with SNOMED CT codes, given that
Bosnia and Herzegovina is not a SNOMED International member?
Not marked for legal review in the SPECs: the terminology plan avoided SNOMED CT for this reason, but the
booking feature already stores and publishes specialty codes from SNOMED CT.
Sources: [booking §2](../booking/SPEC.md#2-settled-decisions), [SOURCES.md, not used](../terminology/SOURCES.md#not-used).

## 6. ClamAV placement

**Q31.** Is it correct that ClamAV, run on our own servers beside the worker so that files are scanned inside
our system and never leave it, is not a processor, and that scanning rests on security of processing (Art. 32)
and the clinic's legitimate interest?
Each file is decrypted only inside our stack and streamed to the scanner; a staff member can open it only after
it is found clean.
Sources: [attachments §4](../attachments/SPEC.md#4-lawful-basis), [attachments §8](../attachments/SPEC.md#8-before-production),
[processors.md, not processors](../privacy/processors.md#not-processors).

**Q32.** Is there any issue with the scanner downloading virus signatures daily from ClamAV's CDN, or from a
mirror we run, given that the download sends no personal data?
Production may use either the official CDN or a mirror; the choice is still open.
Sources: [attachments §8](../attachments/SPEC.md#8-before-production), [processors.md, not processors](../privacy/processors.md#not-processors).

## 7. Hosting and residency

**Q33.** Which data protection law applies to Salus and to the clinics, as well as or instead of the GDPR,
given that the clinics are in Bosnia and Herzegovina and some patients are visitors from abroad?
The privacy notes are written to the GDPR and name no Bosnian law.
Sources: [processors.md](../privacy/processors.md), [i18n §1](../i18n/SPEC.md#1-goal).

**Q34.** May the patient data of clinics in Bosnia and Herzegovina be hosted in the EU/EEA, or must it stay in
Bosnia and Herzegovina, and does hosting it in the EU/EEA count as a transfer that needs its own basis or
approval?
The processors list says data stays in the EU/EEA; the hosting provider is not chosen.
Sources: [processors.md](../privacy/processors.md), [processors.md, pending](../privacy/processors.md#pending).

**Q35.** What must the hosting provider's DPA cover, given that it will hold everything we store (health data
included) in managed Postgres, an object store and backups, and the encryption keys in its key management
service?
Before production the object store must be private with versioning off and encryption at rest on, and the keys
for TOTP secrets and file encryption come from the provider's KMS.
Sources: [processors.md, pending](../privacy/processors.md#pending), [attachments §8](../attachments/SPEC.md#8-before-production),
[accounts §2](../accounts/SPEC.md#2-settled-decisions).

**Q36.** Is it right that a LiveKit video server and TURN relay we run ourselves on the hosting provider are
covered by that provider's DPA, and that only a hosted LiveKit Cloud or a separate TURN service would be a
processor of its own?
Live audio and video pass through our LiveKit server and are not stored; LiveKit never receives a name.
Sources: [consultation §2](../consultation/SPEC.md#2-settled-decisions), [consultation §4](../consultation/SPEC.md#4-lawful-basis),
[processors.md, pending](../privacy/processors.md#pending).

## 8. Other

**Q37.** Is each clinic the controller of its patients' data and Salus its processor, and what must the contract
between Salus and each clinic contain for that (Art. 28)?
The privacy notes and the consent screen already assume these roles, but no SPEC records a legal decision on
them.
Sources: [breach.md](../privacy/breach.md#3-notify-the-supervisory-authority-within-72-hours-of-awareness),
[DPIA, booking package 9](../privacy/dpia.md#package-9-mobile-app).

**Q38.** Does Salus need a data protection officer, and which supervisory authority must be notified of a
personal data breach?
Both are placeholders in the breach procedure.
Sources: [breach.md](../privacy/breach.md).
