import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

// The owner's Android build settings (docs/attachments/IMPLEMENTATION_PLAN.md, package 1.7).
void main() {
  test('the app compiles against Android SDK 37', () {
    final gradle = File('android/app/build.gradle.kts').readAsStringSync();
    expect(gradle, contains('compileSdk = 37'));
  });
}
