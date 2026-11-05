import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

// The debug and profile manifests declare INTERNET for Flutter's tooling; a release build reads only the main
// manifest, so without it the app reaches neither the API nor LiveKit.
void main() {
  test('the release manifest lets the app use the network', () {
    final manifest = File('android/app/src/main/AndroidManifest.xml').readAsStringSync();
    expect(manifest, contains('<uses-permission android:name="android.permission.INTERNET"/>'));
  });
}
