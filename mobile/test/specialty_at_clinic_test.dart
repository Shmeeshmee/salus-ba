import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:salus/l10n/app_localizations.dart';
import 'package:salus/messages.dart';

void main() {
  // The practitioner's line on the free times screen is one message, so each language writes its own separator
  // (docs/i18n/SPEC.md §3 "No string is built by joining pieces"): Arabic takes its own comma.
  test('the specialty and the clinic are one message in the language', () {
    String line(String tag, String? specialty) =>
        specialtyAtClinic(lookupAppLocalizations(Locale(tag)), specialty, 'Synthetic clinic');
    expect(line('en', 'Cardiology'), 'Cardiology, Synthetic clinic');
    expect(line('ar', 'أمراض القلب'), 'أمراض القلب، Synthetic clinic');
    expect(line('ar', null), 'Synthetic clinic');
  });
}
