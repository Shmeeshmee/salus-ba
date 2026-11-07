import 'package:flutter_test/flutter_test.dart';
import 'package:intl/date_symbol_data_local.dart';
import 'package:salus/zones.dart';

void main() {
  final sarajevo = Zone('Europe/Sarajevo');

  // The app has the date symbols once Flutter's localizations load; a plain test loads them itself.
  setUpAll(initializeDateFormatting);

  test('shows an instant as the date and time in the zone', () {
    final start = DateTime.utc(2026, 11, 5, 8, 30);

    expect(sarajevo.name, 'Europe/Sarajevo');
    expect(sarajevo.day(start), DateTime.utc(2026, 11, 5));
    expect(sarajevo.time(start), '09:30');
    expect(sarajevo.date(start, 'en'), 'Thursday 5 November 2026');
  });

  test('a time just after midnight UTC can still be the evening before in another zone', () {
    final newYork = Zone('America/New_York');

    expect(newYork.day(DateTime.utc(2026, 11, 5, 2)), DateTime.utc(2026, 11, 4));
    expect(newYork.time(DateTime.utc(2026, 11, 5, 2)), '21:00');
  });

  test('a day runs from local midnight to the next, 23 or 25 hours across a change', () {
    expect(sarajevo.startOf(DateTime.utc(2026, 11, 5)), DateTime.utc(2026, 11, 4, 23));
    expect(sarajevo.startOf(DateTime.utc(2027, 3, 28)), DateTime.utc(2027, 3, 27, 23));
    expect(sarajevo.startOf(DateTime.utc(2027, 3, 29)), DateTime.utc(2027, 3, 28, 22));
    expect(sarajevo.startOf(DateTime.utc(2027, 10, 31)), DateTime.utc(2027, 10, 30, 22));
    expect(sarajevo.startOf(DateTime.utc(2027, 11, 1)), DateTime.utc(2027, 10, 31, 23));
  });

  test('a long date names the weekday and the month', () {
    expect(Zone.longDate(DateTime.utc(2027, 1, 31), 'en'), 'Sunday 31 January 2027');
    expect(Zone.longDate(DateTime.utc(2026, 12, 7), 'en'), 'Monday 7 December 2026');
  });

  group('a long date is written in the language of the app (docs/i18n/SPEC.md §3 "Formatting")', () {
    // gen-l10n's locale names: Serbian by script, sr_Cyrl and sr_Latn.
    final cases = <(String, DateTime, String)>[
      ('bs', DateTime.utc(2026, 11, 4), 'srijeda, 4. novembar 2026.'),
      ('hr', DateTime.utc(2026, 11, 4), 'srijeda, 4. studenoga 2026.'),
      ('de', DateTime.utc(2026, 11, 4), 'Mittwoch, 4. November 2026'),
      ('tr', DateTime.utc(2026, 11, 4), '4 Kasım 2026 Çarşamba'),
      ('it', DateTime.utc(2026, 11, 4), 'mercoledì 4 novembre 2026'),
      ('fr', DateTime.utc(2026, 11, 4), 'mercredi 4 novembre 2026'),
      // Serbian as it is written in Bosnia and Herzegovina: ijekavian, in both scripts (SPEC §3).
      ('sr_Latn', DateTime.utc(2026, 11, 4), 'srijeda, 4. novembar 2026.'),
      ('sr_Latn', DateTime.utc(2026, 11, 2), 'ponedjeljak, 2. novembar 2026.'),
      ('sr_Latn', DateTime.utc(2026, 11, 8), 'nedjelja, 8. novembar 2026.'),
      ('sr_Cyrl', DateTime.utc(2026, 11, 4), 'сриједа, 4. новембар 2026.'),
      ('sr_Cyrl', DateTime.utc(2026, 11, 2), 'понедјељак, 2. новембар 2026.'),
      ('sr_Cyrl', DateTime.utc(2026, 11, 8), 'недјеља, 8. новембар 2026.'),
    ];
    for (final (locale, day, text) in cases) {
      test('$locale ${day.weekday}', () => expect(Zone.longDate(day, locale), text));
    }

    test('Arabic with Latin digits', () {
      final text = Zone.longDate(DateTime.utc(2026, 11, 4), 'ar');
      expect(text, contains('4'));
      expect(text, contains('2026'));
      expect(text, isNot(contains('٤')));
    });
  });

  test('a zone it does not know falls back to UTC and says so', () {
    final unknown = Zone('Not/AZone');

    expect(unknown.name, 'UTC');
    expect(unknown.time(DateTime.utc(2026, 11, 5, 8, 30)), '08:30');
  });
}
