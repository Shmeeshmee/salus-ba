import 'package:flutter_test/flutter_test.dart';
import 'package:salus/zones.dart';

void main() {
  final sarajevo = Zone('Europe/Sarajevo');

  test('shows an instant as the date and time in the zone', () {
    final start = DateTime.utc(2026, 11, 5, 8, 30);

    expect(sarajevo.name, 'Europe/Sarajevo');
    expect(sarajevo.day(start), DateTime.utc(2026, 11, 5));
    expect(sarajevo.time(start), '09:30');
    expect(sarajevo.date(start), 'Thursday 5 November 2026');
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
    expect(Zone.longDate(DateTime.utc(2027, 1, 31)), 'Sunday 31 January 2027');
    expect(Zone.longDate(DateTime.utc(2026, 12, 7)), 'Monday 7 December 2026');
  });

  test('a zone it does not know falls back to UTC and says so', () {
    final unknown = Zone('Not/AZone');

    expect(unknown.name, 'UTC');
    expect(unknown.time(DateTime.utc(2026, 11, 5, 8, 30)), '08:30');
  });
}
