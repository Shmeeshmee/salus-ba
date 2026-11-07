import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:salus_api/api.dart';
import 'package:url_launcher/url_launcher.dart';

import 'app.dart';
import 'booking.dart';
import 'call.dart';
import 'files.dart';
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
  final bearer = HttpBearerAuth();
  final client = ApiClient(basePath: apiBaseUrl, authentication: bearer);
  final session = Session(api: AuthApi(client), bearer: bearer, store: const SecureTokenStore(FlutterSecureStorage()));
  runApp(
    SalusApp(
      session: session,
      booking: Booking(client),
      now: DateTime.now,
      media: LiveKitMedia(),
      files: DeviceFiles(),
      privacyNoticeUrl: Uri.parse(privacyNoticeUrl),
      openUrl: (url) => launchUrl(url, mode: LaunchMode.externalApplication),
    ),
  );
}
