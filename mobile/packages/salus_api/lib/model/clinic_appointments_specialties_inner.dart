//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ClinicAppointmentsSpecialtiesInner {
  /// Returns a new [ClinicAppointmentsSpecialtiesInner] instance.
  ClinicAppointmentsSpecialtiesInner({
    required this.specialty,
    required this.booked,
    required this.fulfilled,
    required this.noshow,
    required this.cancelled,
  });

  /// The role's SNOMED CT specialty code; empty for roles without one.
  String specialty;

  int booked;

  int fulfilled;

  int noshow;

  int cancelled;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ClinicAppointmentsSpecialtiesInner &&
          other.specialty == specialty &&
          other.booked == booked &&
          other.fulfilled == fulfilled &&
          other.noshow == noshow &&
          other.cancelled == cancelled;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (specialty.hashCode) +
      (booked.hashCode) +
      (fulfilled.hashCode) +
      (noshow.hashCode) +
      (cancelled.hashCode);

  @override
  String toString() =>
      'ClinicAppointmentsSpecialtiesInner[specialty=$specialty, booked=$booked, fulfilled=$fulfilled, noshow=$noshow, cancelled=$cancelled]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'specialty'] = this.specialty;
    json[r'booked'] = this.booked;
    json[r'fulfilled'] = this.fulfilled;
    json[r'noshow'] = this.noshow;
    json[r'cancelled'] = this.cancelled;
    return json;
  }

  /// Returns a new [ClinicAppointmentsSpecialtiesInner] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ClinicAppointmentsSpecialtiesInner? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'specialty'),
            'Required key "ClinicAppointmentsSpecialtiesInner[specialty]" is missing from JSON.');
        assert(json[r'specialty'] != null,
            'Required key "ClinicAppointmentsSpecialtiesInner[specialty]" has a null value in JSON.');
        assert(json.containsKey(r'booked'),
            'Required key "ClinicAppointmentsSpecialtiesInner[booked]" is missing from JSON.');
        assert(json[r'booked'] != null,
            'Required key "ClinicAppointmentsSpecialtiesInner[booked]" has a null value in JSON.');
        assert(json.containsKey(r'fulfilled'),
            'Required key "ClinicAppointmentsSpecialtiesInner[fulfilled]" is missing from JSON.');
        assert(json[r'fulfilled'] != null,
            'Required key "ClinicAppointmentsSpecialtiesInner[fulfilled]" has a null value in JSON.');
        assert(json.containsKey(r'noshow'),
            'Required key "ClinicAppointmentsSpecialtiesInner[noshow]" is missing from JSON.');
        assert(json[r'noshow'] != null,
            'Required key "ClinicAppointmentsSpecialtiesInner[noshow]" has a null value in JSON.');
        assert(json.containsKey(r'cancelled'),
            'Required key "ClinicAppointmentsSpecialtiesInner[cancelled]" is missing from JSON.');
        assert(json[r'cancelled'] != null,
            'Required key "ClinicAppointmentsSpecialtiesInner[cancelled]" has a null value in JSON.');
        return true;
      }());

      return ClinicAppointmentsSpecialtiesInner(
        specialty: mapValueOfType<String>(json, r'specialty')!,
        booked: mapValueOfType<int>(json, r'booked')!,
        fulfilled: mapValueOfType<int>(json, r'fulfilled')!,
        noshow: mapValueOfType<int>(json, r'noshow')!,
        cancelled: mapValueOfType<int>(json, r'cancelled')!,
      );
    }
    return null;
  }

  static List<ClinicAppointmentsSpecialtiesInner> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ClinicAppointmentsSpecialtiesInner>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ClinicAppointmentsSpecialtiesInner.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ClinicAppointmentsSpecialtiesInner> mapFromJson(
      dynamic json) {
    final map = <String, ClinicAppointmentsSpecialtiesInner>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ClinicAppointmentsSpecialtiesInner.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ClinicAppointmentsSpecialtiesInner-objects as value to a dart map
  static Map<String, List<ClinicAppointmentsSpecialtiesInner>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ClinicAppointmentsSpecialtiesInner>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ClinicAppointmentsSpecialtiesInner.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'specialty',
    'booked',
    'fulfilled',
    'noshow',
    'cancelled',
  };
}
