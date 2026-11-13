//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ClinicAppointmentsSeriesInner {
  /// Returns a new [ClinicAppointmentsSeriesInner] instance.
  ClinicAppointmentsSeriesInner({
    required this.period,
    required this.booked,
    required this.fulfilled,
    required this.noshow,
    required this.cancelled,
    required this.refilled,
  });

  /// The month, YYYY-MM.
  String period;

  /// Every appointment starting in the month.
  int booked;

  int fulfilled;

  int noshow;

  int cancelled;

  /// Booked through the waitlist (a freed slot refilled), whatever the status now (SAL-36).
  int refilled;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ClinicAppointmentsSeriesInner &&
          other.period == period &&
          other.booked == booked &&
          other.fulfilled == fulfilled &&
          other.noshow == noshow &&
          other.cancelled == cancelled &&
          other.refilled == refilled;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (period.hashCode) +
      (booked.hashCode) +
      (fulfilled.hashCode) +
      (noshow.hashCode) +
      (cancelled.hashCode) +
      (refilled.hashCode);

  @override
  String toString() =>
      'ClinicAppointmentsSeriesInner[period=$period, booked=$booked, fulfilled=$fulfilled, noshow=$noshow, cancelled=$cancelled, refilled=$refilled]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'period'] = this.period;
    json[r'booked'] = this.booked;
    json[r'fulfilled'] = this.fulfilled;
    json[r'noshow'] = this.noshow;
    json[r'cancelled'] = this.cancelled;
    json[r'refilled'] = this.refilled;
    return json;
  }

  /// Returns a new [ClinicAppointmentsSeriesInner] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ClinicAppointmentsSeriesInner? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'period'),
            'Required key "ClinicAppointmentsSeriesInner[period]" is missing from JSON.');
        assert(json[r'period'] != null,
            'Required key "ClinicAppointmentsSeriesInner[period]" has a null value in JSON.');
        assert(json.containsKey(r'booked'),
            'Required key "ClinicAppointmentsSeriesInner[booked]" is missing from JSON.');
        assert(json[r'booked'] != null,
            'Required key "ClinicAppointmentsSeriesInner[booked]" has a null value in JSON.');
        assert(json.containsKey(r'fulfilled'),
            'Required key "ClinicAppointmentsSeriesInner[fulfilled]" is missing from JSON.');
        assert(json[r'fulfilled'] != null,
            'Required key "ClinicAppointmentsSeriesInner[fulfilled]" has a null value in JSON.');
        assert(json.containsKey(r'noshow'),
            'Required key "ClinicAppointmentsSeriesInner[noshow]" is missing from JSON.');
        assert(json[r'noshow'] != null,
            'Required key "ClinicAppointmentsSeriesInner[noshow]" has a null value in JSON.');
        assert(json.containsKey(r'cancelled'),
            'Required key "ClinicAppointmentsSeriesInner[cancelled]" is missing from JSON.');
        assert(json[r'cancelled'] != null,
            'Required key "ClinicAppointmentsSeriesInner[cancelled]" has a null value in JSON.');
        assert(json.containsKey(r'refilled'),
            'Required key "ClinicAppointmentsSeriesInner[refilled]" is missing from JSON.');
        assert(json[r'refilled'] != null,
            'Required key "ClinicAppointmentsSeriesInner[refilled]" has a null value in JSON.');
        return true;
      }());

      return ClinicAppointmentsSeriesInner(
        period: mapValueOfType<String>(json, r'period')!,
        booked: mapValueOfType<int>(json, r'booked')!,
        fulfilled: mapValueOfType<int>(json, r'fulfilled')!,
        noshow: mapValueOfType<int>(json, r'noshow')!,
        cancelled: mapValueOfType<int>(json, r'cancelled')!,
        refilled: mapValueOfType<int>(json, r'refilled')!,
      );
    }
    return null;
  }

  static List<ClinicAppointmentsSeriesInner> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ClinicAppointmentsSeriesInner>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ClinicAppointmentsSeriesInner.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ClinicAppointmentsSeriesInner> mapFromJson(dynamic json) {
    final map = <String, ClinicAppointmentsSeriesInner>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ClinicAppointmentsSeriesInner.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ClinicAppointmentsSeriesInner-objects as value to a dart map
  static Map<String, List<ClinicAppointmentsSeriesInner>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ClinicAppointmentsSeriesInner>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ClinicAppointmentsSeriesInner.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'period',
    'booked',
    'fulfilled',
    'noshow',
    'cancelled',
    'refilled',
  };
}
