# Push notifications: what the owner creates before they are switched on

The chat tells a patient of a new message with a push to the patient app (SAL-29, docs/plans/chat.md). The notifier
sends it through `pkg/push`: Firebase Cloud Messaging (FCM) for Android, the Apple Push Notification service (APNs)
for iOS. Until the credentials below exist, both platforms run on the fake (`pkg/push/pushtest`), which sends
nothing; the notifier's start line says which each is (`push_android`, `push_ios`: `fake`, `fcm` or `apns`).

A push says only "New message from your clinic", in the patient's language. It carries no message text, no name, no
clinic, no id, nothing clinical. The device token is still personal data, and Google and Apple are outside the
EU/EEA: both stay off until their entries in `docs/privacy/processors.md` are complete (DPA and transfer basis).

The patient app does not register a token yet: the Firebase SDK, the APNs entitlement and the app's call to
`PUT /me/devices` land with these credentials, not before.

## Firebase Cloud Messaging (Android)

1. In the [Firebase console](https://console.firebase.google.com/), create a project for Salus (or use the Google
   Cloud project that will own the app). Turn Google Analytics off: nothing else of Firebase is used.
2. Add an Android app with the patient app's application id. Keep `google-services.json` for the app's build (it is
   configuration, not a secret, and goes in with the app's FCM change).
3. In Google Cloud's IAM for the project, create a service account for the server only, e.g. `salus-push`, with the
   one role **Firebase Cloud Messaging API Admin** (`roles/firebasecloudmessaging.admin`). No other role. Make sure
   the **Firebase Cloud Messaging API** (HTTP v1) is enabled; the legacy API is not used.
4. Create a **JSON key** for that service account and download it. It holds the private key: never mail it, never
   commit it, never put it in the env file or a GitHub secret.
5. Store the JSON file as its own Bitwarden item (file attachment).
6. On the server, as root, put it on the encrypted volume beside the other keys (ADR 0011):
   `/mnt/salus-data/keys/fcm_credentials`, owner `65532:65532`, mode `0400` (`salus-volume check` lists the key
   files and their modes).

The notifier reads it through `FCM_CREDENTIALS_FILE`. The project id comes from the file; nothing else is needed.

## Apple Push Notification service (iOS)

1. The Apple Developer Program membership of the organisation that publishes the patient app: its **Team ID** is on
   the membership page (10 characters).
2. In Certificates, Identifiers & Profiles, the app's identifier (its **bundle id**, e.g. `ba.salus.patient`) with
   the **Push Notifications** capability turned on.
3. Under Keys, create a key with **Apple Push Notifications service (APNs)** enabled. Download the **.p8** file
   (Apple lets you download it once) and note its **Key ID** (10 characters). One key serves every app of the team,
   production and sandbox.
4. Store the .p8 file as its own Bitwarden item, with the Key ID, the Team ID and the bundle id in its notes.
5. On the server, as root: `/mnt/salus-data/keys/apns_key`, owner `65532:65532`, mode `0400`.

The notifier reads the key through `APNS_KEY_FILE`, and `APNS_KEY_ID`, `APNS_TEAM_ID` and `APNS_BUNDLE_ID` from the
env file (they are identifiers, not secrets). `APNS_SANDBOX=true` only for a development build of the app; production
builds (TestFlight, the App Store) use the production service.

## Switching them on

The keys go only where ADR 0011 puts keys: the root-only key files on the LUKS-encrypted volume, mounted into the one
container that reads them as compose secrets. They are never in `/opt/salus/.env`, `SALUS_ENV_FILE`, a GitHub secret
or a commit.

1. Complete the Google and Apple rows of `docs/privacy/processors.md` (DPA, transfer basis) and the DPIA line.
2. Add the two key files as above, and the compose secrets for the **notifier only** to
   `deployments/production/docker-compose.prod.yml` (`fcm_credentials`, `apns_key`, and `FCM_CREDENTIALS_FILE`,
   `APNS_KEY_FILE` pointing at `/run/secrets/…`), with the three APNs ids in the env file. This is a change to the
   compose file and the deploy's key check that lands together with the app's token registration.
3. Deploy. The notifier refuses to start on a half configuration (some of a platform's variables without the others,
   or a file it cannot read), naming the variable, never a value or path.

Each platform is independent: FCM can go live before APNs, the other staying on the fake.

## What the server does with a token

- Stored per device in `push_device` (one row per token; a token signed into another account moves to it), with the
  platform only. Deleted when the app unregisters on sign-out (`DELETE /me/devices`), when FCM answers
  `UNREGISTERED` or APNs answers 410 or `BadDeviceToken`, and with the account.
- A push is a River job per device on the notifier's `push` queue, retried 5 times; a failure is logged with its
  status, never the token.
