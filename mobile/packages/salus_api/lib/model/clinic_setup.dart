//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ClinicSetup {
  /// Returns a new [ClinicSetup] instance.
  ClinicSetup({
    required this.details,
    required this.billing,
    required this.hours,
    required this.practitioner,
    required this.services,
  });

  SetupStepState details;

  SetupStepState billing;

  SetupStepState hours;

  SetupStepState practitioner;

  SetupStepState services;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ClinicSetup &&
          other.details == details &&
          other.billing == billing &&
          other.hours == hours &&
          other.practitioner == practitioner &&
          other.services == services;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (details.hashCode) +
      (billing.hashCode) +
      (hours.hashCode) +
      (practitioner.hashCode) +
      (services.hashCode);

  @override
  String toString() =>
      'ClinicSetup[details=$details, billing=$billing, hours=$hours, practitioner=$practitioner, services=$services]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'details'] = this.details;
    json[r'billing'] = this.billing;
    json[r'hours'] = this.hours;
    json[r'practitioner'] = this.practitioner;
    json[r'services'] = this.services;
    return json;
  }

  /// Returns a new [ClinicSetup] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ClinicSetup? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'details'),
            'Required key "ClinicSetup[details]" is missing from JSON.');
        assert(json[r'details'] != null,
            'Required key "ClinicSetup[details]" has a null value in JSON.');
        assert(json.containsKey(r'billing'),
            'Required key "ClinicSetup[billing]" is missing from JSON.');
        assert(json[r'billing'] != null,
            'Required key "ClinicSetup[billing]" has a null value in JSON.');
        assert(json.containsKey(r'hours'),
            'Required key "ClinicSetup[hours]" is missing from JSON.');
        assert(json[r'hours'] != null,
            'Required key "ClinicSetup[hours]" has a null value in JSON.');
        assert(json.containsKey(r'practitioner'),
            'Required key "ClinicSetup[practitioner]" is missing from JSON.');
        assert(json[r'practitioner'] != null,
            'Required key "ClinicSetup[practitioner]" has a null value in JSON.');
        assert(json.containsKey(r'services'),
            'Required key "ClinicSetup[services]" is missing from JSON.');
        assert(json[r'services'] != null,
            'Required key "ClinicSetup[services]" has a null value in JSON.');
        return true;
      }());

      return ClinicSetup(
        details: SetupStepState.fromJson(json[r'details'])!,
        billing: SetupStepState.fromJson(json[r'billing'])!,
        hours: SetupStepState.fromJson(json[r'hours'])!,
        practitioner: SetupStepState.fromJson(json[r'practitioner'])!,
        services: SetupStepState.fromJson(json[r'services'])!,
      );
    }
    return null;
  }

  static List<ClinicSetup> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ClinicSetup>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ClinicSetup.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ClinicSetup> mapFromJson(dynamic json) {
    final map = <String, ClinicSetup>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ClinicSetup.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ClinicSetup-objects as value to a dart map
  static Map<String, List<ClinicSetup>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ClinicSetup>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ClinicSetup.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'details',
    'billing',
    'hours',
    'practitioner',
    'services',
  };
}
