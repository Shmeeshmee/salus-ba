# Copy waiting for the owner's approval

Collected 2026-11-11 from the catalogs on `main` (web `web/src/i18n`, mobile `mobile/lib/l10n`, iOS
`mobile/ios/Runner/*.lproj/InfoPlist.strings`, e-mail `internal/notification/templates`) and the COPY files that list
them as pending: docs/consultation/COPY.md (M35, M36, M37 to M48, W41 to W43), docs/attachments/COPY.md (M15, M16),
docs/terminology/COPY.md (T16 and the mobile SAL-13 strings), docs/consult-intelligence/COPY.md (W9 and W24
translations, the call page) and docs/plans/v1-finish.PROGRESS.md (SAL-14 and SAL-17). Every text below is copied
from the catalog as the app shows it today. Nothing was changed.

**How to decide.** Fill in the `decision` column of `docs/copy/PENDING_REVIEW.csv` (this file shows the same rows
for reading): `approve`, or `change: <new text>` (English; say which language if the change is only to one
translation, e.g. `change bs: <new text>`). An empty decision leaves the row pending. The `key` column is where the
string lives, so the next run applies the decisions without guessing. Placeholders in `{braces}` or `{{.Name}}`
are filled in by the app and stay as they are.

Every non-English text here was drafted by Claude and has not been read by a native speaker
(docs/i18n/TRANSLATIONS.md: `drafted`). An approval of the English does not approve a translation.

## Totals

| Feature | Rows | Web (5 languages) | Mobile, iOS, e-mail (10 languages) | `legal` |
| --- | ---: | ---: | ---: | ---: |
| Transcription consent (SAL-12) | 20 | 2 | 18 | 15 |
| Patient documents (SAL-17) | 15 | 2 | 13 | 0 |
| Term review panel (SAL-14) | 48 | 48 | 0 | 0 |
| Term explanations (SAL-13) | 4 | 1 | 3 | 2 |
| Call page | 4 | 3 | 1 | 1 |
| **Total** | **91** | **56** | **35** | **18** |

Texts per language (a row counts once per language it has a text in):

| Language | Texts |
| --- | ---: |
| English (`en`) | 91 |
| Bosnian (`bs`) | 91 |
| Croatian (`hr`) | 91 |
| Serbian (Latin) (`sr-Latn`) | 91 |
| Serbian (Cyrillic) (`sr-Cyrl`) | 91 |
| German (`de`) | 35 |
| Arabic (`ar`) | 35 |
| Turkish (`tr`) | 35 |
| Italian (`it`) | 35 |
| French (`fr`) | 35 |
| **All languages** | **630** |

Not counted as rows: the 2,160 drafted term explanations (216 terms × 10 languages,
`assets/terminology/explanations.draft.jsonl`). They are the doctor partner's to review in the review panel
(`/term-review`), where they already sit as unreviewed.

## Also waiting on someone else

- **The lawyer's answer to Q16** (docs/legal/BRIEF.md; Q17 covers the same consent texts): 21 rows,
  M35, M15, M16, M37, M38, M39, M40, M41, M42, M43a, M43b, M44, M45, M46a, M46b, M47, M48a, M48b, W42, W43, W9. W9 is `legal` and goes to the lawyer with the other tagged strings, not under Q16.
- **The doctor partner**: 20 rows, M35, M15, M16, M37, M38, M39, M40, M41, M42, M43a, M43b, M44, M45, M46a, M46b, M47, M48a, M48b, W42, W43, plus the 2,160 explanations.
- **`legal` strings** (their wording has legal weight; the lawyer reads every one in every language): 18 rows,
  M35, M15, M16, M38, M39, M40, M41, M42, M43a, M43b, M45, M46b, M47, W42, W43, T16, TH1, W9.

## Who checks what

- **Bosnian, Croatian, Serbian (Latin and Cyrillic)**: every row (91), web and mobile alike. Native speakers
  of `sr-Latn` and `sr-Cyrl` can be the same person; Serbian is ijekavian in both scripts.
- **German, Arabic, Turkish, Italian, French**: the 35 mobile, iOS and e-mail rows only (the web app is
  staff-only, in five languages): M35, M15, M16, M37, M38, M39, M40, M41, M42, M43a, M43b, M44, M45, M46a, M46b, M47, M48a, M48b, PD3, PD4, PD5, PD6, PD7, PD8, PD9, PD10, PD11, PD12, PD13, PD14, PD15, TH1, TH2, TH3, M36.
- **English**: the owner, every row. W9 and W24 are already approved in English; only their translations wait.
- Arabic: check that the right-to-left layout reads well with the Latin placeholders (dates, the clinic's name).

## Open decisions without a string

- **Call page, a line whose translation failed** (docs/consult-intelligence/COPY.md, "Reused on the call page"):
  it shows the original text, as if its translation were still coming. Marking it (for example "Translation
  unavailable") needs a new string, left for you to decide. Add a row with your decision if you want one.

## Transcription consent (SAL-12)

| ID | Where it appears | English | Also waits on | Note |
| --- | --- | --- | --- | --- |
| M35 | Android, the app's explanation before the system asks for the camera and microphone to join a consultation (`mobile:whyCameraAndMicrophone`) | To join the video consultation, Salus needs your camera and microphone. The consultation is not recorded as audio or video. If you agree, what is said is transcribed and translated. | `legal`, lawyer (Q16); doctor partner |  |
| M15 | iOS system prompt for the camera (consultation and questionnaire photos) (`ios:NSCameraUsageDescription`) | Salus uses the camera so the practitioner can see you during a video consultation, and to take the photos and videos you add to a questionnaire. The consultation is not recorded as audio or video. | `legal`, lawyer (Q16); doctor partner | Only the last sentence changed (SAL-12) |
| M16 | iOS system prompt for the microphone (consultation and questionnaire videos) (`ios:NSMicrophoneUsageDescription`) | Salus uses the microphone so the practitioner can hear you during a video consultation, and to record sound with the videos you add to a questionnaire. The consultation is not recorded as audio or video; if you agree, what is said is transcribed and translated. | `legal`, lawyer (Q16); doctor partner | Only the last sentence changed (SAL-12) |
| M37 | Notice before the device check, while the patient's transcription consent with the clinic is not active: heading (`mobile:transcriptionTitle`) | Transcription and translation | lawyer (Q16); doctor partner |  |
| M38 | Notice before the device check, while the patient's transcription consent with the clinic is not active: text (`mobile:transcriptionIntro`) | In your video consultations with {clinic}, what you and the practitioner say can be transcribed as you speak and translated, so the practitioner can follow you in your language. | `legal`, lawyer (Q16); doctor partner |  |
| M39 | Notice before the device check, while the patient's transcription consent with the clinic is not active: text (`mobile:transcriptionAssistance`) | From the transcript, the practitioner also gets AI assistance: suggestions during the consultation and drafts of documents. These are suggestions only; the practitioner checks them and makes every decision. | `legal`, lawyer (Q16); doctor partner |  |
| M40 | Notice before the device check, while the patient's transcription consent with the clinic is not active: text (`mobile:transcriptionKept`) | The transcript is kept with your medical record at {clinic}. The consultation is not recorded as audio or video. | `legal`, lawyer (Q16); doctor partner |  |
| M41 | Notice before the device check, while the patient's transcription consent with the clinic is not active: text (`mobile:transcriptionDecline`) | If you do not agree, the consultation goes ahead as usual, without transcription, translation or AI assistance. | `legal`, lawyer (Q16); doctor partner |  |
| M42 | Notice before the device check, while the patient's transcription consent with the clinic is not active: text (`mobile:transcriptionWithdraw`) | You can withdraw your agreement at any time, also during the consultation: tap “Transcription is on”, or open My consents. Transcription stops at once; what was transcribed until then stays in your record. | `legal`, lawyer (Q16); doctor partner |  |
| M43a | Notice before the device check, while the patient's transcription consent with the clinic is not active: button that gives the consent (`mobile:transcriptionAgree`) | Agree and continue | `legal`, lawyer (Q16); doctor partner |  |
| M43b | Notice before the device check, while the patient's transcription consent with the clinic is not active: button that declines (`mobile:transcriptionWithout`) | Continue without transcription | `legal`, lawyer (Q16); doctor partner |  |
| M44 | Withdraw dialog (from the badge in the call or My consents): title (`mobile:stopTranscriptionTitle`) | Stop transcription? | lawyer (Q16); doctor partner |  |
| M45 | Withdraw dialog: text (`mobile:stopTranscriptionBody`) | Transcription, translation and AI assistance stop at once in your consultations with {clinic}. A consultation in progress goes on without them, and your appointments stay as they are. What was transcribed until now stays in your record. | `legal`, lawyer (Q16); doctor partner |  |
| M46a | Withdraw dialog: button that cancels (`mobile:keepTranscription`) | Keep transcription | lawyer (Q16); doctor partner |  |
| M46b | Withdraw dialog: button that withdraws (`mobile:stopTranscription`) | Stop transcription | `legal`, lawyer (Q16); doctor partner |  |
| M47 | Message after the consent was withdrawn (`mobile:transcriptionStopped`) | Transcription is off. Your consultations with this clinic go ahead without it. | `legal`, lawyer (Q16); doctor partner |  |
| M48a | My consents: the kind of a care consent (`mobile:consentForCare`) | Sharing for care | lawyer (Q16); doctor partner |  |
| M48b | My consents: the kind of a transcription consent (`mobile:consentForTranscription`) | Transcription and AI assistance | lawyer (Q16); doctor partner |  |
| W42 | Practitioner's call page, while the agent is not in the call because the patient has not agreed or withdrew (`web:consultation.transcriptionOff`) | Transcription is off: the patient has not agreed to transcription and AI assistance. The consultation goes ahead without them. | `legal`, lawyer (Q16); doctor partner |  |
| W43 | Practitioner's call page, the draft panel: why no draft can be generated (`web:consultation.live.transcriptionConsentRequired`) | The patient has not agreed to transcription and AI assistance, so no draft can be generated. | `legal`, lawyer (Q16); doctor partner |  |

<details><summary>Translations</summary>

| ID | `bs` | `hr` | `sr-Latn` | `sr-Cyrl` | `de` | `ar` | `tr` | `it` | `fr` |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| M35 | Da biste se pridružili video konsultaciji, Salus treba pristup vašoj kameri i mikrofonu. Konsultacija se ne snima ni kao zvuk ni kao video. Ako se složite, ono što se kaže transkribuje se i prevodi. | Da biste se pridružili videokonzultaciji, Salus treba pristup vašoj kameri i mikrofonu. Konzultacija se ne snima ni kao zvuk ni kao video. Ako pristanete, ono što se izgovori transkribira se i prevodi. | Da biste se uključili u video-konsultaciju, Salus treba pristup vašoj kameri i mikrofonu. Konsultacija se ne snima ni kao zvuk ni kao video. Ako se saglasite, ono što se kaže transkribuje se i prevodi. | Да бисте се укључили у видео-консултацију, Salus треба приступ вашој камери и микрофону. Консултација се не снима ни као звук ни као видео. Ако се сагласите, оно што се каже транскрибује се и преводи. | Für die Teilnahme an der Videosprechstunde benötigt Salus Zugriff auf Ihre Kamera und Ihr Mikrofon. Die Sprechstunde wird weder als Ton noch als Video aufgezeichnet. Wenn Sie einwilligen, wird das Gesprochene transkribiert und übersetzt. | للانضمام إلى استشارة الفيديو، يحتاج Salus إلى الكاميرا والميكروفون. لا تُسجَّل الاستشارة صوتًا ولا فيديو. وإذا وافقت، يُنسخ ما يُقال نصيًا ويُترجم. | Görüntülü görüşmeye katılmak için Salus'un kameranıza ve mikrofonunuza erişmesi gerekir. Görüşme ses veya görüntü olarak kaydedilmez. Onay verirseniz söylenenler yazıya dökülür ve çevrilir. | Per partecipare alla videovisita, Salus ha bisogno di accedere alla fotocamera e al microfono. La visita non viene registrata né in audio né in video. Se acconsente, ciò che viene detto viene trascritto e tradotto. | Pour rejoindre la consultation vidéo, Salus a besoin de votre caméra et de votre micro. La consultation n’est enregistrée ni en audio ni en vidéo. Si vous l’acceptez, ce qui est dit est transcrit et traduit. |
| M15 | Salus koristi kameru kako bi vas ljekar mogao vidjeti tokom video konsultacije i kako biste snimili fotografije i videozapise koje dodajete u upitnik. Konsultacija se ne snima ni kao zvuk ni kao video. | Salus koristi kameru kako bi vas liječnik mogao vidjeti tijekom videokonzultacije te kako biste snimili fotografije i videozapise koje dodajete u upitnik. Konzultacija se ne snima ni kao zvuk ni kao video. | Salus koristi kameru kako bi vas ljekar mogao vidjeti tokom video-konsultacije i za snimanje fotografija i video-snimaka koje dodajete u upitnik. Konsultacija se ne snima ni kao zvuk ni kao video. | Salus користи камеру како би вас љекар могао видјети током видео-консултације и за снимање фотографија и видео-снимака које додајете у упитник. Консултација се не снима ни као звук ни као видео. | Salus verwendet die Kamera, damit Ihr Behandler Sie während einer Videosprechstunde sehen kann, und um die Fotos und Videos aufzunehmen, die Sie einem Fragebogen hinzufügen. Die Sprechstunde wird weder als Ton noch als Video aufgezeichnet. | يستخدم Salus الكاميرا ليتمكّن الطبيب من رؤيتك أثناء استشارة الفيديو، ولالتقاط الصور ومقاطع الفيديو التي تضيفها إلى الاستبيان. لا تُسجَّل الاستشارة صوتًا ولا فيديو. | Salus, görüntülü görüşme sırasında sağlık uzmanının sizi görebilmesi ve bir soru formuna eklediğiniz fotoğraf ve videoları çekmek için kamerayı kullanır. Görüşme ses veya görüntü olarak kaydedilmez. | Salus usa la fotocamera per consentire al medico di vederLa durante una videovisita e per scattare le foto e registrare i video da aggiungere a un questionario. La visita non viene registrata né in audio né in video. | Salus utilise la caméra pour que le praticien puisse vous voir pendant une consultation vidéo, et pour prendre les photos et vidéos que vous ajoutez à un questionnaire. La consultation n’est enregistrée ni en audio ni en vidéo. |
| M16 | Salus koristi mikrofon kako bi vas ljekar mogao čuti tokom video konsultacije i kako bi se snimio zvuk uz videozapise koje dodajete u upitnik. Konsultacija se ne snima ni kao zvuk ni kao video; ako se složite, ono što se kaže transkribuje se i prevodi. | Salus koristi mikrofon kako bi vas liječnik mogao čuti tijekom videokonzultacije te kako bi se snimio zvuk uz videozapise koje dodajete u upitnik. Konzultacija se ne snima ni kao zvuk ni kao video; ako pristanete, ono što se izgovori transkribira se i prevodi. | Salus koristi mikrofon kako bi vas ljekar mogao čuti tokom video-konsultacije i za snimanje zvuka uz video-snimke koje dodajete u upitnik. Konsultacija se ne snima ni kao zvuk ni kao video; ako se saglasite, ono što se kaže transkribuje se i prevodi. | Salus користи микрофон како би вас љекар могао чути током видео-консултације и за снимање звука уз видео-снимке које додајете у упитник. Консултација се не снима ни као звук ни као видео; ако се сагласите, оно што се каже транскрибује се и преводи. | Salus verwendet das Mikrofon, damit Ihr Behandler Sie während einer Videosprechstunde hören kann, und um den Ton der Videos aufzunehmen, die Sie einem Fragebogen hinzufügen. Die Sprechstunde wird weder als Ton noch als Video aufgezeichnet; wenn Sie einwilligen, wird das Gesprochene transkribiert und übersetzt. | يستخدم Salus الميكروفون ليتمكّن الطبيب من سماعك أثناء استشارة الفيديو، ولتسجيل الصوت مع مقاطع الفيديو التي تضيفها إلى الاستبيان. لا تُسجَّل الاستشارة صوتًا ولا فيديو، وإذا وافقت، يُنسخ ما يُقال نصيًا ويُترجم. | Salus, görüntülü görüşme sırasında sağlık uzmanının sizi duyabilmesi ve bir soru formuna eklediğiniz videoların sesini kaydetmek için mikrofonu kullanır. Görüşme ses veya görüntü olarak kaydedilmez; onay verirseniz söylenenler yazıya dökülür ve çevrilir. | Salus usa il microfono per consentire al medico di sentirLa durante una videovisita e per registrare l'audio dei video da aggiungere a un questionario. La visita non viene registrata né in audio né in video; se acconsente, ciò che viene detto viene trascritto e tradotto. | Salus utilise le micro pour que le praticien puisse vous entendre pendant une consultation vidéo, et pour enregistrer le son des vidéos que vous ajoutez à un questionnaire. La consultation n’est enregistrée ni en audio ni en vidéo ; si vous l’acceptez, ce qui est dit est transcrit et traduit. |
| M37 | Transkripcija i prijevod | Transkripcija i prijevod | Transkripcija i prevod | Транскрипција и превод | Transkription und Übersetzung | النسخ النصي والترجمة | Transkripsiyon ve çeviri | Trascrizione e traduzione | Transcription et traduction |
| M38 | U vašim video konsultacijama s klinikom {clinic} ono što vi i ljekar kažete može se transkribovati dok govorite i prevesti, kako bi vas ljekar mogao pratiti na vašem jeziku. | U vašim videokonzultacijama s klinikom {clinic} ono što vi i liječnik kažete može se transkribirati dok govorite i prevesti, kako bi vas liječnik mogao pratiti na vašem jeziku. | U vašim video-konsultacijama s klinikom {clinic} ono što vi i ljekar kažete može da se transkribuje dok govorite i da se prevede, kako bi vas ljekar mogao pratiti na vašem jeziku. | У вашим видео-консултацијама с клиником {clinic} оно што ви и љекар кажете може да се транскрибује док говорите и да се преведе, како би вас љекар могао пратити на вашем језику. | In Ihren Videosprechstunden bei {clinic} kann das, was Sie und Ihr Behandler sagen, beim Sprechen transkribiert und übersetzt werden, damit Ihr Behandler Ihnen in Ihrer Sprache folgen kann. | في استشارات الفيديو مع {clinic}، يمكن نسخ ما تقوله أنت والطبيب نصيًا أثناء الحديث وترجمته، ليتمكن الطبيب من متابعتك بلغتك. | {clinic} ile görüntülü görüşmelerinizde sizin ve sağlık uzmanının söyledikleri konuşurken yazıya dökülebilir ve çevrilebilir; böylece sağlık uzmanı sizi kendi dilinizde takip edebilir. | Nelle Sue videovisite con {clinic}, ciò che Lei e il medico dite può essere trascritto mentre parlate e tradotto, così il medico può seguirLa nella Sua lingua. | Lors de vos consultations vidéo avec {clinic}, ce que vous et le praticien dites peut être transcrit au fil de la conversation et traduit, afin que le praticien puisse vous suivre dans votre langue. |
| M39 | Na osnovu transkripta ljekar dobija i pomoć vještačke inteligencije (AI): prijedloge tokom konsultacije i nacrte dokumenata. To su samo prijedlozi; ljekar ih provjerava i donosi svaku odluku. | Na temelju transkripta liječnik dobiva i pomoć umjetne inteligencije (AI): prijedloge tijekom konzultacije i nacrte dokumenata. To su samo prijedlozi; liječnik ih provjerava i donosi svaku odluku. | Na osnovu transkripta ljekar dobija i pomoć vještačke inteligencije (AI): prijedloge tokom konsultacije i nacrte dokumenata. To su samo prijedlozi; ljekar ih provjerava i donosi svaku odluku. | На основу транскрипта љекар добија и помоћ вјештачке интелигенције (AI): приједлоге током консултације и нацрте докумената. То су само приједлози; љекар их провјерава и доноси сваку одлуку. | Aus dem Transkript erhält Ihr Behandler außerdem KI-Unterstützung: Vorschläge während der Sprechstunde und Entwürfe von Dokumenten. Es sind nur Vorschläge; Ihr Behandler prüft sie und trifft jede Entscheidung selbst. | ويحصل الطبيب أيضًا على مساعدة من الذكاء الاصطناعي انطلاقًا من النص المنسوخ: اقتراحات أثناء الاستشارة ومسودات للمستندات. وهي مجرد اقتراحات، يراجعها الطبيب ويتخذ كل قرار بنفسه. | Sağlık uzmanı ayrıca yazıya dökülen metinden yapay zekâ desteği alır: görüşme sırasında öneriler ve belge taslakları. Bunlar yalnızca önerilerdir; sağlık uzmanı bunları kontrol eder ve her kararı kendisi verir. | Dalla trascrizione il medico riceve anche un supporto dell'intelligenza artificiale: suggerimenti durante la visita e bozze di documenti. Sono solo suggerimenti; il medico li verifica e prende ogni decisione. | À partir de la transcription, le praticien reçoit aussi une aide de l’intelligence artificielle : des suggestions pendant la consultation et des brouillons de documents. Ce ne sont que des suggestions ; le praticien les vérifie et prend chaque décision. |
| M40 | Transkript se čuva uz vašu medicinsku dokumentaciju u klinici {clinic}. Konsultacija se ne snima ni kao zvuk ni kao video. | Transkript se čuva uz vašu medicinsku dokumentaciju u klinici {clinic}. Konzultacija se ne snima ni kao zvuk ni kao video. | Transkript se čuva uz vašu medicinsku dokumentaciju u klinici {clinic}. Konsultacija se ne snima ni kao zvuk ni kao video. | Транскрипт се чува уз вашу медицинску документацију у клиници {clinic}. Консултација се не снима ни као звук ни као видео. | Das Transkript wird mit Ihren Behandlungsunterlagen bei {clinic} aufbewahrt. Die Sprechstunde wird weder als Ton noch als Video aufgezeichnet. | يُحفظ النص المنسوخ مع سجلك الطبي لدى {clinic}. لا تُسجَّل الاستشارة صوتًا ولا فيديو. | Yazıya dökülen metin, sağlık kaydınızla birlikte {clinic} tarafından saklanır. Görüşme ses veya görüntü olarak kaydedilmez. | La trascrizione viene conservata con la Sua documentazione sanitaria presso {clinic}. La visita non viene registrata né in audio né in video. | La transcription est conservée avec votre dossier médical chez {clinic}. La consultation n’est enregistrée ni en audio ni en vidéo. |
| M41 | Ako se ne složite, konsultacija se odvija kao i obično, bez transkripcije, prijevoda i pomoći vještačke inteligencije. | Ako ne pristanete, konzultacija se odvija kao i obično, bez transkripcije, prijevoda i pomoći umjetne inteligencije. | Ako se ne saglasite, konsultacija teče kao i obično, bez transkripcije, prevoda i pomoći vještačke inteligencije. | Ако се не сагласите, консултација тече као и обично, без транскрипције, превода и помоћи вјештачке интелигенције. | Wenn Sie nicht einwilligen, findet die Sprechstunde wie gewohnt statt, ohne Transkription, Übersetzung und KI-Unterstützung. | إذا لم توافق، تجري الاستشارة كالمعتاد، دون نسخ نصي أو ترجمة أو مساعدة من الذكاء الاصطناعي. | Onay vermezseniz görüşme her zamanki gibi; transkripsiyon, çeviri ve yapay zekâ desteği olmadan yapılır. | Se non acconsente, la visita si svolge come di consueto, senza trascrizione, traduzione né supporto dell'intelligenza artificiale. | Si vous refusez, la consultation se déroule normalement, sans transcription, traduction ni aide de l’intelligence artificielle. |
| M42 | Saglasnost možete povući u bilo kojem trenutku, i tokom konsultacije: dodirnite „Transkripcija je uključena” ili otvorite Moje saglasnosti. Transkripcija prestaje odmah; ono što je do tada transkribovano ostaje u vašoj dokumentaciji. | Privolu možete povući u bilo kojem trenutku, i tijekom konzultacije: dodirnite „Transkripcija je uključena” ili otvorite Moje privole. Transkripcija prestaje odmah; ono što je dotad transkribirano ostaje u vašoj dokumentaciji. | Saglasnost možete povući u bilo kom trenutku, i tokom konsultacije: dodirnite „Transkripcija je uključena” ili otvorite Moje saglasnosti. Transkripcija prestaje odmah; ono što je do tada transkribovano ostaje u vašoj dokumentaciji. | Сагласност можете повући у било ком тренутку, и током консултације: додирните „Транскрипција је укључена” или отворите Моје сагласности. Транскрипција престаје одмах; оно што је до тада транскрибовано остаје у вашој документацији. | Sie können Ihre Einwilligung jederzeit widerrufen, auch während der Sprechstunde: Tippen Sie auf „Transkription ist aktiv“ oder öffnen Sie Meine Einwilligungen. Die Transkription endet sofort; was bis dahin transkribiert wurde, bleibt in Ihren Unterlagen. | يمكنك سحب موافقتك في أي وقت، حتى أثناء الاستشارة: اضغط على «النسخ النصي مفعّل» أو افتح موافقاتي. يتوقف النسخ النصي فورًا، ويبقى ما نُسخ حتى ذلك الحين في سجلك. | Onayınızı istediğiniz zaman, görüşme sırasında da geri alabilirsiniz: “Transkripsiyon açık” yazısına dokunun veya Onaylarım bölümünü açın. Transkripsiyon hemen durur; o ana kadar yazıya dökülenler kaydınızda kalır. | Può revocare il consenso in qualsiasi momento, anche durante la visita: tocchi «Trascrizione attiva» oppure apra I miei consensi. La trascrizione si interrompe subito; ciò che è stato trascritto fino a quel momento resta nella Sua documentazione. | Vous pouvez retirer votre accord à tout moment, y compris pendant la consultation : touchez « Transcription activée » ou ouvrez Mes consentements. La transcription s’arrête aussitôt ; ce qui a été transcrit jusque-là reste dans votre dossier. |
| M43a | Prihvati i nastavi | Prihvati i nastavi | Prihvatite i nastavite | Прихватите и наставите | Einwilligen und fortfahren | الموافقة والمتابعة | Onayla ve devam et | Acconsenti e continua | Accepter et continuer |
| M43b | Nastavi bez transkripcije | Nastavi bez transkripcije | Nastavite bez transkripcije | Наставите без транскрипције | Ohne Transkription fortfahren | المتابعة دون نسخ نصي | Transkripsiyon olmadan devam et | Continua senza trascrizione | Continuer sans transcription |
| M44 | Zaustaviti transkripciju? | Zaustaviti transkripciju? | Zaustaviti transkripciju? | Зауставити транскрипцију? | Transkription beenden? | هل تريد إيقاف النسخ النصي؟ | Transkripsiyon durdurulsun mu? | Interrompere la trascrizione? | Arrêter la transcription ? |
| M45 | Transkripcija, prijevod i pomoć vještačke inteligencije odmah prestaju u vašim konsultacijama s klinikom {clinic}. Konsultacija koja je u toku nastavlja se bez njih, a vaši termini ostaju kakvi jesu. Ono što je do sada transkribovano ostaje u vašoj dokumentaciji. | Transkripcija, prijevod i pomoć umjetne inteligencije odmah prestaju u vašim konzultacijama s klinikom {clinic}. Konzultacija koja je u tijeku nastavlja se bez njih, a vaši termini ostaju nepromijenjeni. Ono što je dosad transkribirano ostaje u vašoj dokumentaciji. | Transkripcija, prevod i pomoć vještačke inteligencije odmah prestaju u vašim konsultacijama s klinikom {clinic}. Konsultacija koja je u toku nastavlja se bez njih, a vaši termini ostaju kakvi jesu. Ono što je do sada transkribovano ostaje u vašoj dokumentaciji. | Транскрипција, превод и помоћ вјештачке интелигенције одмах престају у вашим консултацијама с клиником {clinic}. Консултација која је у току наставља се без њих, а ваши термини остају какви јесу. Оно што је до сада транскрибовано остаје у вашој документацији. | Transkription, Übersetzung und KI-Unterstützung enden sofort in Ihren Sprechstunden bei {clinic}. Eine laufende Sprechstunde geht ohne sie weiter, und Ihre Termine bleiben bestehen. Was bis jetzt transkribiert wurde, bleibt in Ihren Unterlagen. | يتوقف النسخ النصي والترجمة ومساعدة الذكاء الاصطناعي فورًا في استشاراتك مع {clinic}. تستمر الاستشارة الجارية بدونها، وتبقى مواعيدك كما هي. ويبقى ما نُسخ حتى الآن في سجلك. | {clinic} ile görüşmelerinizde transkripsiyon, çeviri ve yapay zekâ desteği hemen durur. Devam eden bir görüşme bunlar olmadan sürer ve randevularınız olduğu gibi kalır. Şimdiye kadar yazıya dökülenler kaydınızda kalır. | Trascrizione, traduzione e supporto dell'intelligenza artificiale si interrompono subito nelle Sue visite con {clinic}. Una visita in corso prosegue senza di essi e i Suoi appuntamenti restano invariati. Ciò che è stato trascritto finora resta nella Sua documentazione. | La transcription, la traduction et l’aide de l’intelligence artificielle s’arrêtent aussitôt dans vos consultations avec {clinic}. Une consultation en cours se poursuit sans elles, et vos rendez-vous restent inchangés. Ce qui a été transcrit jusqu’ici reste dans votre dossier. |
| M46a | Zadrži transkripciju | Zadrži transkripciju | Zadržite transkripciju | Задржите транскрипцију | Transkription behalten | الإبقاء على النسخ النصي | Transkripsiyonu sürdür | Mantieni la trascrizione | Garder la transcription |
| M46b | Zaustavi transkripciju | Zaustavi transkripciju | Zaustavite transkripciju | Зауставите транскрипцију | Transkription beenden | إيقاف النسخ النصي | Transkripsiyonu durdur | Interrompi la trascrizione | Arrêter la transcription |
| M47 | Transkripcija je isključena. Vaše konsultacije s ovom klinikom odvijaju se bez nje. | Transkripcija je isključena. Vaše konzultacije s ovom klinikom odvijaju se bez nje. | Transkripcija je isključena. Vaše konsultacije s ovom klinikom teku bez nje. | Транскрипција је искључена. Ваше консултације с овом клиником теку без ње. | Die Transkription ist aus. Ihre Sprechstunden bei dieser Klinik finden ohne sie statt. | النسخ النصي متوقف. تجري استشاراتك مع هذه العيادة بدونه. | Transkripsiyon kapalı. Bu klinikle görüşmeleriniz transkripsiyon olmadan yapılır. | La trascrizione è disattivata. Le Sue visite con questa clinica si svolgono senza. | La transcription est désactivée. Vos consultations avec cette clinique se déroulent sans elle. |
| M48a | Dijeljenje radi liječenja | Dijeljenje radi liječenja | Dijeljenje radi liječenja | Дијељење ради лијечења | Weitergabe für die Behandlung | المشاركة لأغراض الرعاية | Bakım için paylaşım | Condivisione per le cure | Partage pour les soins |
| M48b | Transkripcija i pomoć vještačke inteligencije | Transkripcija i pomoć umjetne inteligencije | Transkripcija i pomoć vještačke inteligencije | Транскрипција и помоћ вјештачке интелигенције | Transkription und KI-Unterstützung | النسخ النصي ومساعدة الذكاء الاصطناعي | Transkripsiyon ve yapay zekâ desteği | Trascrizione e supporto dell'intelligenza artificiale | Transcription et aide de l’IA |
| W42 | Transkripcija je isključena: pacijent nije dao saglasnost za transkripciju i AI pomoć. Konsultacija se odvija bez njih. | Transkripcija je isključena: pacijent nije dao privolu za transkripciju i AI pomoć. Konzultacija se odvija bez njih. | Transkripcija je isključena: pacijent nije dao saglasnost za transkripciju i AI pomoć. Konsultacija teče bez njih. | Транскрипција је искључена: пацијент није дао сагласност за транскрипцију и AI помоћ. Консултација тече без њих. |  |  |  |  |  |
| W43 | Pacijent nije dao saglasnost za transkripciju i AI pomoć, pa nacrt nije moguće generisati. | Pacijent nije dao privolu za transkripciju i AI pomoć pa nacrt nije moguće generirati. | Pacijent nije dao saglasnost za transkripciju i AI pomoć, pa nacrt nije moguće generisati. | Пацијент није дао сагласност за транскрипцију и AI помоћ, па нацрт није могуће генерисати. |  |  |  |  |  |

</details>

## Patient documents (SAL-17)

| ID | Where it appears | English | Also waits on | Note |
| --- | --- | --- | --- | --- |
| PD1 | Clinic settings (admin): the checkbox that gives patients a transcript copy (off by default) (`web:admin.transcriptCopy`) | Give patients a copy of their consultation transcripts |  |  |
| PD2 | Clinic settings (admin): hint under that checkbox (`web:admin.transcriptCopyHint`) | Patients can then read the transcripts of their own consultations in the patient app. |  |  |
| PD3 | Mobile: the button under an appointment that has started, and the title of its documents screen (`mobile:documents`) | Documents |  |  |
| PD4 | Mobile: the documents screen when the appointment has no signed document (`mobile:noDocuments`) | There are no signed documents for this appointment yet. A document appears here once your practitioner signs it. |  |  |
| PD5 | Mobile: under a signed document's title: the day it was signed, as a long date in the clinic's zone (`mobile:documentSignedOn`) | Signed on {date} |  |  |
| PD6 | Mobile: under a signed document written in another language than the app's: that language by its own name (`mobile:documentLanguage`) | Language: {language} |  |  |
| PD7 | Mobile: button: the signed document's PDF to the device's share sheet (open, save, send) (`mobile:sharePdf`) | Save or share the PDF |  |  |
| PD8 | Mobile: the button to the transcript copy, and its screen's title (`mobile:transcript`) | Transcript |  |  |
| PD9 | Mobile: at the top of the transcript copy (`mobile:transcriptNote`) | The consultation as it was transcribed and translated automatically. It may contain mistakes. |  |  |
| PD10 | Mobile: the transcript copy when nothing was stored (`mobile:transcriptEmpty`) | Nothing was transcribed in this consultation. |  |  |
| PD11 | Mobile: the transcript: the patient's own lines (`mobile:speakerYou`) | You |  |  |
| PD12 | Mobile: the transcript: the practitioner's lines (`mobile:speakerPractitioner`) | Practitioner |  |  |
| PD13 | Mobile: button: the transcript copy to the device's share sheet as text (`mobile:shareTranscript`) | Share the transcript |  |  |
| PD14 | Mobile: under a transcript line: its translation (`mobile:transcriptTranslation`) | Translation: {text} |  |  |
| PD15 | E-mail to the patient when a document from their appointment is signed (first line is the subject; {{.Clinic}}, {{.When}} and {{.Link}} are filled in) (`email:signed.txt`) | A document from your appointment is ready<br><br>A document from your appointment at {{.Clinic}} on {{.When}} is ready.<br><br>To read it, open the Salus app:<br>{{.Link}} |  |  |

<details><summary>Translations</summary>

| ID | `bs` | `hr` | `sr-Latn` | `sr-Cyrl` | `de` | `ar` | `tr` | `it` | `fr` |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| PD1 | Omogući pacijentima kopiju transkripta njihovih konsultacija | Omogući pacijentima kopiju transkripta njihovih konzultacija | Omogući pacijentima kopiju transkripta njihovih konsultacija | Омогући пацијентима копију транскрипта њихових консултација |  |  |  |  |  |
| PD2 | Pacijenti tada mogu čitati transkripte svojih konsultacija u aplikaciji za pacijente. | Pacijenti tada mogu čitati transkripte svojih konzultacija u aplikaciji za pacijente. | Pacijenti tada mogu da čitaju transkripte svojih konsultacija u aplikaciji za pacijente. | Пацијенти тада могу да читају транскрипте својих консултација у апликацији за пацијенте. |  |  |  |  |  |
| PD3 | Dokumenti | Dokumenti | Dokumenti | Документи | Dokumente | المستندات | Belgeler | Documenti | Documents |
| PD4 | Za ovaj termin još nema potpisanih dokumenata. Dokument se pojavljuje ovdje kada ga vaš ljekar potpiše. | Za ovaj termin još nema potpisanih dokumenata. Dokument se ovdje pojavljuje kada ga vaš liječnik potpiše. | Za ovaj termin još nema potpisanih dokumenata. Dokument se pojavljuje ovdje kada ga vaš ljekar potpiše. | За овај термин још нема потписаних докумената. Документ се појављује овдје када га ваш љекар потпише. | Für diesen Termin gibt es noch keine unterschriebenen Dokumente. Ein Dokument erscheint hier, sobald Ihr Behandler es unterschrieben hat. | لا توجد بعد مستندات موقعة لهذا الموعد. يظهر المستند هنا بعد أن يوقعه طبيبك. | Bu randevu için henüz imzalanmış belge yok. Sağlık uzmanınız bir belgeyi imzaladığında belge burada görünür. | Per questo appuntamento non ci sono ancora documenti firmati. Un documento compare qui quando il medico lo firma. | Il n’y a pas encore de document signé pour ce rendez-vous. Un document apparaît ici dès que votre praticien l’a signé. |
| PD5 | Potpisano {date} | Potpisano {date} | Potpisano {date} | Потписано {date} | Unterschrieben am {date} | تم التوقيع في {date} | İmza tarihi: {date} | Firmato il {date} | Signé le {date} |
| PD6 | Jezik: {language} | Jezik: {language} | Jezik: {language} | Језик: {language} | Sprache: {language} | اللغة: {language} | Dil: {language} | Lingua: {language} | Langue : {language} |
| PD7 | Sačuvaj ili podijeli PDF | Spremi ili podijeli PDF | Sačuvaj ili podijeli PDF | Сачувај или подијели PDF | PDF speichern oder teilen | حفظ ملف PDF أو مشاركته | PDF'yi kaydet veya paylaş | Salva o condividi il PDF | Enregistrer ou partager le PDF |
| PD8 | Transkript | Transkript | Transkript | Транскрипт | Transkript | النص المكتوب | Görüşme dökümü | Trascrizione | Transcription |
| PD9 | Konsultacija onako kako je automatski transkribovana i prevedena. Može sadržavati greške. | Konzultacija onako kako je automatski transkribirana i prevedena. Može sadržavati pogreške. | Konsultacija onako kako je automatski transkribovana i prevedena. Može da sadrži greške. | Консултација онако како је аутоматски транскрибована и преведена. Може да садржи грешке. | Die Sprechstunde, wie sie automatisch transkribiert und übersetzt wurde. Sie kann Fehler enthalten. | الاستشارة كما تم تحويلها إلى نص وترجمتها تلقائيًا. قد تحتوي على أخطاء. | Görüşmenin otomatik olarak yazıya dökülmüş ve çevrilmiş hâli. Hatalar içerebilir. | La visita come è stata trascritta e tradotta automaticamente. Può contenere errori. | La consultation telle qu’elle a été transcrite et traduite automatiquement. Elle peut contenir des erreurs. |
| PD10 | U ovoj konsultaciji ništa nije transkribovano. | U ovoj konzultaciji ništa nije transkribirano. | U ovoj konsultaciji ništa nije transkribovano. | У овој консултацији ништа није транскрибовано. | In dieser Sprechstunde wurde nichts transkribiert. | لم يُحوَّل أي شيء إلى نص في هذه الاستشارة. | Bu görüşmede yazıya dökülen bir şey yok. | In questa visita non è stato trascritto nulla. | Rien n’a été transcrit pendant cette consultation. |
| PD11 | Vi | Vi | Vi | Ви | Sie | أنت | Siz | Lei | Vous |
| PD12 | Ljekar | Liječnik | Ljekar | Љекар | Behandler | الطبيب | Sağlık uzmanı | Medico | Praticien |
| PD13 | Podijeli transkript | Podijeli transkript | Podijeli transkript | Подијели транскрипт | Transkript teilen | مشاركة النص المكتوب | Dökümü paylaş | Condividi la trascrizione | Partager la transcription |
| PD14 | Prijevod: {text} | Prijevod: {text} | Prijevod: {text} | Пријевод: {text} | Übersetzung: {text} | الترجمة: {text} | Çeviri: {text} | Traduzione: {text} | Traduction : {text} |
| PD15 | Dokument s vašeg termina je spreman<br><br>Dokument s vašeg termina u ustanovi {{.Clinic}} je spreman.<br>Datum i vrijeme: {{.When}}<br><br>Da biste ga pročitali, otvorite aplikaciju Salus:<br>{{.Link}} | Dokument s vašeg termina spreman je<br><br>Dokument s vašeg termina u ustanovi {{.Clinic}} spreman je.<br>Datum i vrijeme: {{.When}}<br><br>Za čitanje otvorite aplikaciju Salus:<br>{{.Link}} | Dokument sa vašeg termina je spreman<br><br>Dokument sa vašeg termina u ustanovi {{.Clinic}} je spreman.<br>Datum i vrijeme: {{.When}}<br><br>Da biste ga pročitali, otvorite aplikaciju Salus:<br>{{.Link}} | Документ са вашег термина је спреман<br><br>Документ са вашег термина у установи {{.Clinic}} је спреман.<br>Датум и вријеме: {{.When}}<br><br>Да бисте га прочитали, отворите апликацију Salus:<br>{{.Link}} | Ein Dokument zu Ihrem Termin liegt vor<br><br>Ein Dokument zu Ihrem Termin bei {{.Clinic}} liegt vor.<br>Datum und Uhrzeit: {{.When}}<br><br>Um es zu lesen, öffnen Sie die Salus-App:<br>{{.Link}} | مستند من موعدك جاهز<br><br>مستند من موعدك في {{.Clinic}} جاهز.<br>التاريخ والوقت: {{.When}}<br><br>لقراءته، افتح تطبيق Salus:<br>{{.Link}} | Randevunuza ait bir belge hazır<br><br>Randevunuza ait bir belge hazır.<br>Klinik: {{.Clinic}}<br>Tarih ve saat: {{.When}}<br><br>Belgeyi okumak için Salus uygulamasını açın:<br>{{.Link}} | È disponibile un documento del suo appuntamento<br><br>È disponibile un documento del suo appuntamento presso {{.Clinic}}.<br>Data e ora: {{.When}}<br><br>Per leggerlo, apra l'app Salus:<br>{{.Link}} | Un document de votre rendez-vous est disponible<br><br>Un document de votre rendez-vous chez {{.Clinic}} est disponible.<br>Date et heure : {{.When}}<br><br>Pour le consulter, ouvrez l'application Salus :<br>{{.Link}} |

</details>

## Term review panel (SAL-14)

| ID | Where it appears | English | Also waits on | Note |
| --- | --- | --- | --- | --- |
| TR1 | Navigation link, for reviewers (`web:shell.termReview`) | Term review |  |  |
| TR2 | /term-review: page title (`web:review.title`) | Term review |  |  |
| TR3 | /term-review: text under the title (`web:review.intro`) | Check the labels and explanations of the medical terms. A change reaches every clinic. |  |  |
| TR4 | /term-review: notice for staff who are not reviewers (read-only page) (`web:review.readOnly`) | Only reviewers can edit and approve terms. |  |  |
| TR5 | /term-review: filter label (`web:review.languageFilter`) | Language |  |  |
| TR6 | /term-review: filter label (`web:review.statusFilter`) | Status |  |  |
| TR7 | /term-review: status name (filter and entries) (`web:review.status.unreviewed`) | Unreviewed |  |  |
| TR8 | /term-review: status name (filter and entries) (`web:review.status.needs_changes`) | Needs changes |  |  |
| TR9 | /term-review: status name (filter and entries) (`web:review.status.approved`) | Approved |  |  |
| TR10 | /term-review: status filter: every status (`web:review.status.all`) | All |  |  |
| TR11 | /term-review: progress line for the selected language (`web:review.progress`) | {approved} of {terms} approved in {language} |  |  |
| TR12 | /term-review: progress: how many entries need changes (`web:review.progressNeedsChanges`) | {count, plural, one {# needs changes} other {# need changes}} |  |  |
| TR13 | /term-review: accessible name of the progress bar (`web:review.progressBar`) | Review progress in {language} |  |  |
| TR14 | /term-review: disclosure that shows the progress of the other languages (`web:review.progressAll`) | All languages |  |  |
| TR15 | /term-review: heading of the queue (`web:review.terms`) | Terms |  |  |
| TR16 | /term-review: heading of the keyboard shortcuts (`web:review.shortcuts`) | Keyboard shortcuts |  |  |
| TR17 | /term-review: keyboard shortcut help (`web:review.shortcut.move`) | ↓ and ↑, or J and K: next or previous term |  |  |
| TR18 | /term-review: keyboard shortcut help (`web:review.shortcut.labels`) | A, E, N: approve, edit or request changes to the labels |  |  |
| TR19 | /term-review: keyboard shortcut help (`web:review.shortcut.explanation`) | Shift with A, E, N: the same for the explanation |  |  |
| TR20 | /term-review: keyboard shortcut help (`web:review.shortcut.save`) | Enter saves, Escape cancels |  |  |
| TR21 | /term-review: error: the queue could not be loaded (`web:review.loadFailed`) | The review queue is not available. Try again. |  |  |
| TR22 | /term-review: empty queue for the filters (`web:review.empty`) | Nothing to review here. |  |  |
| TR23 | /term-review: button under the queue (`web:review.more`) | Load more |  |  |
| TR24 | /term-review: error: someone else changed the entry (412) (`web:review.conflict`) | Someone changed this entry in the meantime. The queue has been reloaded. |  |  |
| TR25 | /term-review: error: requesting changes without a note (`web:review.noteRequired`) | Write what needs to change. |  |  |
| TR26 | /term-review: error: a label or explanation the server refuses (`web:review.invalid`) | Check the text: a label has at most 300 characters and no \|, an explanation at most 500 characters. |  |  |
| TR27 | /term-review: entry: the term has no label in the language (`web:review.noLabel`) | No label in {language} yet |  |  |
| TR28 | /term-review: entry: heading of the labels (`web:review.labels`) | Labels |  |  |
| TR29 | /term-review: entry: heading of the explanation (`web:review.explanation`) | Explanation for patients (informational) |  |  |
| TR30 | /term-review: explanation editor: field label (`web:review.explanationText`) | Explanation text |  |  |
| TR31 | /term-review: explanation editor: hint (`web:review.explanationHint`) | Plain language, at most 500 characters. Informational only, never a diagnosis. |  |  |
| TR32 | /term-review: entry: no explanation yet (`web:review.noExplanation`) | No explanation yet |  |  |
| TR33 | /term-review: entry: the reviewer's note (`web:review.note`) | Note: {note} |  |  |
| TR34 | /term-review: button (`web:review.edit`) | Edit |  |  |
| TR35 | /term-review: button (labels in a language that has none) (`web:review.add`) | Add |  |  |
| TR36 | /term-review: button (an explanation that does not exist) (`web:review.write`) | Write |  |  |
| TR37 | /term-review: button (`web:review.approve`) | Approve |  |  |
| TR38 | /term-review: button (`web:review.requestChanges`) | Request changes |  |  |
| TR39 | /term-review: label editor: field label (`web:review.preferred`) | Preferred label |  |  |
| TR40 | /term-review: label editor: field label (`web:review.synonyms`) | Synonyms, separated by \| |  |  |
| TR41 | /term-review: request changes: field label (`web:review.noteLabel`) | What needs to change |  |  |
| TR42 | /term-review: button (`web:review.save`) | Save |  |  |
| TR43 | /term-review: button (`web:review.cancel`) | Cancel |  |  |
| TR44 | /term-review: language name in the filter and progress (`web:review.language.de`) | German |  |  |
| TR45 | /term-review: language name in the filter and progress (`web:review.language.ar`) | Arabic |  |  |
| TR46 | /term-review: language name in the filter and progress (`web:review.language.tr`) | Turkish |  |  |
| TR47 | /term-review: language name in the filter and progress (`web:review.language.it`) | Italian |  |  |
| TR48 | /term-review: language name in the filter and progress (`web:review.language.fr`) | French |  |  |

<details><summary>Translations</summary>

| ID | `bs` | `hr` | `sr-Latn` | `sr-Cyrl` |
| --- | --- | --- | --- | --- |
| TR1 | Pregled pojmova | Pregled pojmova | Pregled pojmova | Преглед појмова |
| TR2 | Pregled pojmova | Pregled pojmova | Pregled pojmova | Преглед појмова |
| TR3 | Provjerite nazive i objašnjenja medicinskih pojmova. Izmjena stiže do svih klinika. | Provjerite nazive i objašnjenja medicinskih pojmova. Izmjena stiže do svih klinika. | Provjerite nazive i objašnjenja medicinskih pojmova. Izmjena stiže do svih klinika. | Провјерите називе и објашњења медицинских појмова. Измјена стиже до свих клиника. |
| TR4 | Samo recenzenti mogu uređivati i odobravati pojmove. | Samo recenzenti mogu uređivati i odobravati pojmove. | Samo recenzenti mogu da uređuju i odobravaju pojmove. | Само рецензенти могу да уређују и одобравају појмове. |
| TR5 | Jezik | Jezik | Jezik | Језик |
| TR6 | Status | Status | Status | Статус |
| TR7 | Nepregledano | Nepregledano | Nepregledano | Непрегледано |
| TR8 | Potrebne izmjene | Potrebne izmjene | Potrebne izmjene | Потребне измјене |
| TR9 | Odobreno | Odobreno | Odobreno | Одобрено |
| TR10 | Sve | Sve | Sve | Све |
| TR11 | Odobreno {approved} od {terms} na jeziku: {language} | Odobreno {approved} od {terms} na jeziku: {language} | Odobreno {approved} od {terms} na jeziku: {language} | Одобрено {approved} од {terms} на језику: {language} |
| TR12 | {count, plural, one {# traži izmjene} few {# traže izmjene} other {# traži izmjene}} | {count, plural, one {# traži izmjene} few {# traže izmjene} other {# traži izmjene}} | {count, plural, one {# traži izmjene} few {# traže izmjene} other {# traži izmjene}} | {count, plural, one {# тражи измјене} few {# траже измјене} other {# тражи измјене}} |
| TR13 | Napredak pregleda: {language} | Napredak pregleda: {language} | Napredak pregleda: {language} | Напредак прегледа: {language} |
| TR14 | Svi jezici | Svi jezici | Svi jezici | Сви језици |
| TR15 | Pojmovi | Pojmovi | Pojmovi | Појмови |
| TR16 | Prečice na tastaturi | Tipkovnički prečaci | Prečice na tastaturi | Пречице на тастатури |
| TR17 | ↓ i ↑, ili J i K: sljedeći ili prethodni pojam | ↓ i ↑, ili J i K: sljedeći ili prethodni pojam | ↓ i ↑, ili J i K: sljedeći ili prethodni pojam | ↓ и ↑, или J и K: сљедећи или претходни појам |
| TR18 | A, E, N: odobri, uredi ili zatraži izmjene naziva | A, E, N: odobri, uredi ili zatraži izmjene naziva | A, E, N: odobri, uredi ili zatraži izmjene naziva | A, E, N: одобри, уреди или затражи измјене назива |
| TR19 | Shift uz A, E, N: isto za objašnjenje | Shift uz A, E, N: isto za objašnjenje | Shift uz A, E, N: isto za objašnjenje | Shift уз A, E, N: исто за објашњење |
| TR20 | Enter sprema, Escape otkazuje | Enter sprema, Escape odustaje | Enter čuva, Escape otkazuje | Enter чува, Escape отказује |
| TR21 | Red za pregled nije dostupan. Pokušajte ponovo. | Red za pregled nije dostupan. Pokušajte ponovno. | Red za pregled nije dostupan. Pokušajte ponovo. | Ред за преглед није доступан. Покушајте поново. |
| TR22 | Ovdje nema ničega za pregled. | Ovdje nema ničega za pregled. | Ovdje nema ničega za pregled. | Овдје нема ничега за преглед. |
| TR23 | Učitaj još | Učitaj još | Učitaj još | Учитај још |
| TR24 | Neko je u međuvremenu izmijenio ovaj unos. Red je ponovo učitan. | Netko je u međuvremenu izmijenio ovaj unos. Red je ponovno učitan. | Neko je u međuvremenu izmijenio ovaj unos. Red je ponovo učitan. | Неко је у међувремену измијенио овај унос. Ред је поново учитан. |
| TR25 | Napišite šta treba izmijeniti. | Napišite što treba izmijeniti. | Napišite šta treba izmijeniti. | Напишите шта треба измијенити. |
| TR26 | Provjerite tekst: naziv ima najviše 300 znakova i bez \|, objašnjenje najviše 500 znakova. | Provjerite tekst: naziv ima najviše 300 znakova i bez \|, objašnjenje najviše 500 znakova. | Provjerite tekst: naziv ima najviše 300 znakova i bez \|, objašnjenje najviše 500 znakova. | Провјерите текст: назив има највише 300 знакова и без \|, објашњење највише 500 знакова. |
| TR27 | Još nema naziva na jeziku: {language} | Još nema naziva na jeziku: {language} | Još nema naziva na jeziku: {language} | Још нема назива на језику: {language} |
| TR28 | Nazivi | Nazivi | Nazivi | Називи |
| TR29 | Objašnjenje za pacijente (informativno) | Objašnjenje za pacijente (informativno) | Objašnjenje za pacijente (informativno) | Објашњење за пацијенте (информативно) |
| TR30 | Tekst objašnjenja | Tekst objašnjenja | Tekst objašnjenja | Текст објашњења |
| TR31 | Jednostavnim jezikom, najviše 500 znakova. Samo informativno, nikada dijagnoza. | Jednostavnim jezikom, najviše 500 znakova. Samo informativno, nikada dijagnoza. | Jednostavnim jezikom, najviše 500 znakova. Samo informativno, nikada dijagnoza. | Једноставним језиком, највише 500 знакова. Само информативно, никада дијагноза. |
| TR32 | Još nema objašnjenja | Još nema objašnjenja | Još nema objašnjenja | Још нема објашњења |
| TR33 | Napomena: {note} | Napomena: {note} | Napomena: {note} | Напомена: {note} |
| TR34 | Uredi | Uredi | Uredi | Уреди |
| TR35 | Dodaj | Dodaj | Dodaj | Додај |
| TR36 | Napiši | Napiši | Napiši | Напиши |
| TR37 | Odobri | Odobri | Odobri | Одобри |
| TR38 | Zatraži izmjene | Zatraži izmjene | Zatraži izmjene | Затражи измјене |
| TR39 | Preferirani naziv | Preferirani naziv | Željeni naziv | Жељени назив |
| TR40 | Sinonimi, odvojeni znakom \| | Sinonimi, odvojeni znakom \| | Sinonimi, odvojeni znakom \| | Синоними, одвојени знаком \| |
| TR41 | Šta treba izmijeniti | Što treba izmijeniti | Šta treba izmijeniti | Шта треба измијенити |
| TR42 | Spremi | Spremi | Sačuvaj | Сачувај |
| TR43 | Otkaži | Odustani | Otkaži | Откажи |
| TR44 | Njemački | Njemački | Njemački | Њемачки |
| TR45 | Arapski | Arapski | Arapski | Арапски |
| TR46 | Turski | Turski | Turski | Турски |
| TR47 | Italijanski | Talijanski | Italijanski | Италијански |
| TR48 | Francuski | Francuski | Francuski | Француски |

</details>

## Term explanations (SAL-13)

| ID | Where it appears | English | Also waits on | Note |
| --- | --- | --- | --- | --- |
| T16 | Tooltip of a highlighted term (transcript, captions, documents): under its explanation (`web:terminology.explainNote`) | Informational, not a diagnosis | `legal` |  |
| TH1 | Mobile: the explanation dialog after tapping a highlighted term (documents, transcript copy), under the explanation (`mobile:termExplanationNote`) | Informational, not a diagnosis | `legal` |  |
| TH2 | Mobile: beside an explanation of a medical term a doctor has not reviewed yet (`mobile:termNotReviewed`) | Not reviewed yet |  |  |
| TH3 | Mobile: button: closes a medical term's explanation (`mobile:termClose`) | Close |  |  |

<details><summary>Translations</summary>

| ID | `bs` | `hr` | `sr-Latn` | `sr-Cyrl` | `de` | `ar` | `tr` | `it` | `fr` |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| T16 | Informativno, nije dijagnoza | Informativno, nije dijagnoza | Informativno, nije dijagnoza | Информативно, није дијагноза |  |  |  |  |  |
| TH1 | Informativno, nije dijagnoza | Informativno, nije dijagnoza | Informativno, nije dijagnoza | Информативно, није дијагноза | Zur Information, keine Diagnose | للعلم فقط، وليس تشخيصًا | Bilgi amaçlıdır, tanı değildir | A titolo informativo, non è una diagnosi | À titre informatif, pas un diagnostic |
| TH2 | Još nije provjereno | Još nije provjereno | Još nije provjereno | Још није провјерено | Noch nicht geprüft | لم تتم مراجعته بعد | Henüz incelenmedi | Non ancora verificato | Pas encore vérifié |
| TH3 | Zatvori | Zatvori | Zatvori | Затвори | Schließen | إغلاق | Kapat | Chiudi | Fermer |

</details>

## Call page

| ID | Where it appears | English | Also waits on | Note |
| --- | --- | --- | --- | --- |
| W41 | Practitioner's call page, top right with a mint dot, while the transcription agent is in the call (`web:consultation.transcriptionOn`) | Transcription is on |  | English not approved yet (SAL-3) |
| M36 | Patient's call screen, while the transcription agent is in the call; tapping it opens the withdraw dialog (M44) (`mobile:transcriptionOn`) | Transcription is on |  | English not approved yet (SAL-3) |
| W9 | Practitioner's call page, the suggestions panel: what a suggestion is (`web:consultation.live.suggestionsNote`) | AI suggestions are prompts drawn from the transcript, not a diagnosis. Every decision is yours. | `legal`, lawyer | English approved 2026-11-09; only the translations wait for a native speaker's check |
| W24 | Practitioner's call page, instead of the sign confirmation while a required field is empty (`web:consultation.live.requiredEmpty`) | {count, plural, one {# required field is empty. Fill it in before you sign.} other {# required fields are empty. Fill them in before you sign.}} |  | English approved 2026-11-09; only the translations wait for a native speaker's check |

<details><summary>Translations</summary>

| ID | `bs` | `hr` | `sr-Latn` | `sr-Cyrl` | `de` | `ar` | `tr` | `it` | `fr` |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| W41 | Transkripcija je uključena | Transkripcija je uključena | Transkripcija je uključena | Транскрипција је укључена |  |  |  |  |  |
| M36 | Transkripcija je uključena | Transkripcija je uključena | Transkripcija je uključena | Транскрипција је укључена | Transkription ist aktiv | النسخ النصي مفعّل | Transkripsiyon açık | Trascrizione attiva | Transcription activée |
| W9 | AI prijedlozi su podsticaji izvedeni iz transkripta, a ne dijagnoza. Svaka odluka je vaša. | AI prijedlozi su poticaji izvedeni iz transkripta, a ne dijagnoza. Svaka je odluka vaša. | AI prijedlozi su podsticaji izvedeni iz transkripta, a ne dijagnoza. Svaka odluka je vaša. | AI приједлози су подстицаји изведени из транскрипта, а не дијагноза. Свака одлука је ваша. |  |  |  |  |  |
| W24 | {count, plural, one {# obavezno polje je prazno. Popunite ga prije potpisivanja.} few {# obavezna polja su prazna. Popunite ih prije potpisivanja.} other {# obaveznih polja je prazno. Popunite ih prije potpisivanja.}} | {count, plural, one {# obvezno polje je prazno. Ispunite ga prije potpisivanja.} few {# obvezna polja su prazna. Ispunite ih prije potpisivanja.} other {# obveznih polja je prazno. Ispunite ih prije potpisivanja.}} | {count, plural, one {# obavezno polje je prazno. Popunite ga prije potpisivanja.} few {# obavezna polja su prazna. Popunite ih prije potpisivanja.} other {# obaveznih polja je prazno. Popunite ih prije potpisivanja.}} | {count, plural, one {# обавезно поље је празно. Попуните га прије потписивања.} few {# обавезна поља су празна. Попуните их прије потписивања.} other {# обавезних поља је празно. Попуните их прије потписивања.}} |  |  |  |  |  |

</details>

