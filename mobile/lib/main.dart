import 'package:flutter/material.dart';
import 'package:salus_api/api.dart';

import 'health_screen.dart';

/// The API's address, set at build time: `flutter run --dart-define=API_BASE_URL=http://10.0.2.2:8080`
/// (the host as seen from the Android emulator). See README.md.
const apiBaseUrl = String.fromEnvironment('API_BASE_URL', defaultValue: 'http://localhost:8080');

void main() {
  runApp(SalusApp(api: HealthApi(ApiClient(basePath: apiBaseUrl))));
}

class SalusApp extends StatelessWidget {
  const SalusApp({super.key, required this.api});

  final HealthApi api;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(title: 'Salus', home: HealthScreen(api: api));
  }
}
