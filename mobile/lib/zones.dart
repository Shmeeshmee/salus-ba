import 'package:intl/intl.dart' show DateFormat;
import 'package:timezone/data/latest_all.dart' as tzdata;
import 'package:timezone/timezone.dart' as tz;

// The whole zone database, loaded on first use: only `latest_all` has the link zones (Europe/Sarajevo, the
// API's default, is a link) that the API accepts.
final tz.LocationDatabase _database = () {
  tzdata.initializeTimeZones();
  return tz.timeZoneDatabase;
}();

/// The ekavian day names intl's Serbian data has, and the ijekavian ones Serbian in Bosnia and Herzegovina writes
/// (docs/i18n/SPEC.md §3 "Serbian is ijekavian"), in both scripts.
const _ijekavian = {
  'ponedeljak': 'ponedjeljak',
  'sreda': 'srijeda',
  'nedelja': 'nedjelja',
  'понедељак': 'понедјељак',
  'среда': 'сриједа',
  'недеља': 'недјеља',
};
final _ekavianDay = RegExp(_ijekavian.keys.join('|'));

/// intl's date data for one of gen-l10n's locale names: Serbian Cyrillic is plain `sr`, whose script is Cyrillic.
/// English keeps the app's own "Thursday 5 November 2026" (booking COPY), day before month, without intl's comma.
String _dateLocale(String locale) => locale == 'sr_Cyrl' ? 'sr' : locale;

/// A clinic's IANA time zone (docs/booking/SPEC.md "Time": stored in UTC, shown in the clinic's zone). A local
/// date is a [DateTime] in UTC at midnight, used only for its year, month and day.
class Zone {
  Zone(String name) : _location = _find(name);

  final tz.Location _location;

  String get name => _location.name;

  /// Whether [name] is an IANA zone such as Europe/Sarajevo: an area and a place, known to the zone database. The
  /// database also has abbreviations (CET), which a device may report but which are not the patient's zone.
  static bool isIana(String name) => name.contains('/') && _database.locations.containsKey(name);

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
  String time(DateTime instant) => clock(tz.TZDateTime.from(instant, _location));

  /// The 24-hour "HH:MM" of a time already in its zone.
  static String clock(DateTime local) => '${_two(local.hour)}:${_two(local.minute)}';

  /// The local date of [instant] in the language of [locale] (gen-l10n's `localeName`): "Thursday 5 November 2026".
  String date(DateTime instant, String locale) => longDate(day(instant), locale);

  /// [day] with its weekday and month in the language of [locale], Latin digits in every language (docs/i18n/SPEC.md
  /// §3 "Formatting").
  static String longDate(DateTime day, String locale) {
    final format = locale == 'en' ? DateFormat('EEEE d MMMM y', 'en_GB') : DateFormat.yMMMMEEEEd(_dateLocale(locale));
    final text = (format..useNativeDigits = false).format(day);
    return locale.startsWith('sr') ? text.replaceAllMapped(_ekavianDay, (m) => _ijekavian[m[0]]!) : text;
  }

  static String _two(int n) => n.toString().padLeft(2, '0');
}
