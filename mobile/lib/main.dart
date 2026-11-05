import 'package:flutter/material.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:salus_api/api.dart';

import 'app.dart';
import 'booking.dart';
import 'session.dart';
import 'token_store.dart';

/// The API's address, set at build time: `flutter run --dart-define=API_BASE_URL=http://10.0.2.2:8080`
/// (the host as seen from the Android emulator). See README.md.
const apiBaseUrl = String.fromEnvironment('API_BASE_URL', defaultValue: 'http://localhost:8080');

void main() {
  final bearer = HttpBearerAuth();
  final client = ApiClient(basePath: apiBaseUrl, authentication: bearer);
  final session = Session(api: AuthApi(client), bearer: bearer, store: const SecureTokenStore(FlutterSecureStorage()));
  runApp(SalusApp(session: session, booking: Booking(client), now: DateTime.now));
}
