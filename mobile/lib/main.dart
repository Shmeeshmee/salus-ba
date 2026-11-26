import 'package:flutter/foundation.dart' show kReleaseMode;
import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:salus_api/api.dart';
import 'package:url_launcher/url_launcher.dart';

import 'app.dart';
import 'booking.dart';
import 'call.dart';
import 'chat_client.dart';
import 'close_account_screen.dart' show supportEmail;
import 'files.dart';
import 'l10n/language.dart';
import 'release_settings.dart';
import 'session.dart';
import 'token_store.dart';

/// The API's address, set at build time: `flutter run --dart-define=API_BASE_URL=http://10.0.2.2:8080`
/// (the host as seen from the Android emulator). See README.md.
const apiBaseUrl = String.fromEnvironment('API_BASE_URL', defaultValue: 'http://localhost:8080');

/// The privacy notice the consent screen links to, set at build time:
/// `--dart-define=PRIVACY_NOTICE_URL=https://…`. The default is a placeholder (.invalid resolves nowhere) until
/// the notice is published (docs/booking/SPEC.md, before production).
const privacyNoticeUrl = String.fromEnvironment('PRIVACY_NOTICE_URL', defaultValue: 'https://salus.invalid/privacy');

void main() {
  // A release build given no real API, privacy notice or support address (scripts/mobile-release.sh) does not start:
  // a patient never sees a placeholder. Debug and test builds keep the defaults.
  if (kReleaseMode) {
    final problems = releaseSettingProblems(
      apiBaseUrl: apiBaseUrl,
      privacyNoticeUrl: privacyNoticeUrl,
      supportEmail: supportEmail,
    );
    if (problems.isNotEmpty) {
      throw StateError('release build without its --dart-define settings: ${problems.join(', ')}');
    }
  }
  const storage = FlutterSecureStorage();
  final bearer = HttpBearerAuth();
  final client = ApiClient(basePath: apiBaseUrl, authentication: bearer);
  final session = Session(api: AuthApi(client), bearer: bearer, store: const SecureTokenStore(storage));
  runApp(
    SalusApp(
      session: session,
      booking: Booking(client),
      chats: ChatClient(client),
      now: DateTime.now,
      media: LiveKitMedia(),
      files: DeviceFiles(),
      languages: const SecureLanguageStore(storage),
      privacyNoticeUrl: Uri.parse(privacyNoticeUrl),
      openUrl: (url) => launchUrl(url, mode: LaunchMode.externalApplication),
    ),
  );
}
