//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ClinicAppointmentsPractitionersInner {
  /// Returns a new [ClinicAppointmentsPractitionersInner] instance.
  ClinicAppointmentsPractitionersInner({
    required this.practitionerRole,
    required this.fulfilled,
    required this.noshow,
    required this.booked,
  });

  String practitionerRole;

  int fulfilled;

  int noshow;

  /// Every appointment of the role (SAL-36).
  int booked;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ClinicAppointmentsPractitionersInner &&
          other.practitionerRole == practitionerRole &&
          other.fulfilled == fulfilled &&
          other.noshow == noshow &&
          other.booked == booked;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (practitionerRole.hashCode) +
      (fulfilled.hashCode) +
      (noshow.hashCode) +
      (booked.hashCode);

  @override
  String toString() =>
      'ClinicAppointmentsPractitionersInner[practitionerRole=$practitionerRole, fulfilled=$fulfilled, noshow=$noshow, booked=$booked]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'practitionerRole'] = this.practitionerRole;
    json[r'fulfilled'] = this.fulfilled;
    json[r'noshow'] = this.noshow;
    json[r'booked'] = this.booked;
    return json;
  }

  /// Returns a new [ClinicAppointmentsPractitionersInner] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ClinicAppointmentsPractitionersInner? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'practitionerRole'),
            'Required key "ClinicAppointmentsPractitionersInner[practitionerRole]" is missing from JSON.');
        assert(json[r'practitionerRole'] != null,
            'Required key "ClinicAppointmentsPractitionersInner[practitionerRole]" has a null value in JSON.');
        assert(json.containsKey(r'fulfilled'),
            'Required key "ClinicAppointmentsPractitionersInner[fulfilled]" is missing from JSON.');
        assert(json[r'fulfilled'] != null,
            'Required key "ClinicAppointmentsPractitionersInner[fulfilled]" has a null value in JSON.');
        assert(json.containsKey(r'noshow'),
            'Required key "ClinicAppointmentsPractitionersInner[noshow]" is missing from JSON.');
        assert(json[r'noshow'] != null,
            'Required key "ClinicAppointmentsPractitionersInner[noshow]" has a null value in JSON.');
        assert(json.containsKey(r'booked'),
            'Required key "ClinicAppointmentsPractitionersInner[booked]" is missing from JSON.');
        assert(json[r'booked'] != null,
            'Required key "ClinicAppointmentsPractitionersInner[booked]" has a null value in JSON.');
        return true;
      }());

      return ClinicAppointmentsPractitionersInner(
        practitionerRole: mapValueOfType<String>(json, r'practitionerRole')!,
        fulfilled: mapValueOfType<int>(json, r'fulfilled')!,
        noshow: mapValueOfType<int>(json, r'noshow')!,
        booked: mapValueOfType<int>(json, r'booked')!,
      );
    }
    return null;
  }

  static List<ClinicAppointmentsPractitionersInner> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ClinicAppointmentsPractitionersInner>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ClinicAppointmentsPractitionersInner.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ClinicAppointmentsPractitionersInner> mapFromJson(
      dynamic json) {
    final map = <String, ClinicAppointmentsPractitionersInner>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value =
            ClinicAppointmentsPractitionersInner.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ClinicAppointmentsPractitionersInner-objects as value to a dart map
  static Map<String, List<ClinicAppointmentsPractitionersInner>>
      mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ClinicAppointmentsPractitionersInner>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ClinicAppointmentsPractitionersInner.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'practitionerRole',
    'fulfilled',
    'noshow',
    'booked',
  };
}
