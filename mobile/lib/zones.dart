import 'package:timezone/data/latest_all.dart' as tzdata;
import 'package:timezone/timezone.dart' as tz;

// The whole zone database, loaded on first use: only `latest_all` has the link zones (Europe/Sarajevo, the
// API's default, is a link) that the API accepts.
final tz.LocationDatabase _database = () {
  tzdata.initializeTimeZones();
  return tz.timeZoneDatabase;
}();

const _weekdays = ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday'];
const _months = [
  'January',
  'February',
  'March',
  'April',
  'May',
  'June',
  'July',
  'August',
  'September',
  'October',
  'November',
  'December',
];

/// A clinic's IANA time zone (docs/booking/SPEC.md "Time": stored in UTC, shown in the clinic's zone). A local
/// date is a [DateTime] in UTC at midnight, used only for its year, month and day.
class Zone {
  Zone(String name) : _location = _find(name);

  final tz.Location _location;

  String get name => _location.name;

  static tz.Location _find(String name) => _database.locations[name] ?? _database.get('UTC');

  /// The local date [instant] falls on.
  DateTime day(DateTime instant) {
    final local = tz.TZDateTime.from(instant, _location);
    return DateTime.utc(local.year, local.month, local.day);
  }

  /// The instant the local date [day] starts.
  DateTime startOf(DateTime day) => DateTime.fromMillisecondsSinceEpoch(
    tz.TZDateTime(_location, day.year, day.month, day.day).millisecondsSinceEpoch,
    isUtc: true,
  );

  /// The local time of [instant], 24-hour "HH:MM".
  String time(DateTime instant) {
    final local = tz.TZDateTime.from(instant, _location);
    return '${_two(local.hour)}:${_two(local.minute)}';
  }

  /// The local date of [instant], "Thursday 5 November 2026".
  String date(DateTime instant) => longDate(day(instant));

  static String longDate(DateTime day) =>
      '${_weekdays[day.weekday - 1]} ${day.day} ${_months[day.month - 1]} ${day.year}';

  static String _two(int n) => n.toString().padLeft(2, '0');
}
