//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ClinicWaitlistEntry {
  /// Returns a new [ClinicWaitlistEntry] instance.
  ClinicWaitlistEntry({
    required this.entry,
    required this.patientId,
    this.patientName,
    required this.priority,
  });

  WaitlistEntry entry;

  String patientId;

  /// Absent where the care consent does not give it.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? patientName;

  /// Minimum value: 0
  /// Maximum value: 100
  int priority;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ClinicWaitlistEntry &&
          other.entry == entry &&
          other.patientId == patientId &&
          other.patientName == patientName &&
          other.priority == priority;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (entry.hashCode) +
      (patientId.hashCode) +
      (patientName == null ? 0 : patientName!.hashCode) +
      (priority.hashCode);

  @override
  String toString() =>
      'ClinicWaitlistEntry[entry=$entry, patientId=$patientId, patientName=$patientName, priority=$priority]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'entry'] = this.entry;
    json[r'patient_id'] = this.patientId;
    if (this.patientName != null) {
      json[r'patient_name'] = this.patientName;
    } else {
      json[r'patient_name'] = null;
    }
    json[r'priority'] = this.priority;
    return json;
  }

  /// Returns a new [ClinicWaitlistEntry] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ClinicWaitlistEntry? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'entry'),
            'Required key "ClinicWaitlistEntry[entry]" is missing from JSON.');
        assert(json[r'entry'] != null,
            'Required key "ClinicWaitlistEntry[entry]" has a null value in JSON.');
        assert(json.containsKey(r'patient_id'),
            'Required key "ClinicWaitlistEntry[patient_id]" is missing from JSON.');
        assert(json[r'patient_id'] != null,
            'Required key "ClinicWaitlistEntry[patient_id]" has a null value in JSON.');
        assert(json.containsKey(r'priority'),
            'Required key "ClinicWaitlistEntry[priority]" is missing from JSON.');
        assert(json[r'priority'] != null,
            'Required key "ClinicWaitlistEntry[priority]" has a null value in JSON.');
        return true;
      }());

      return ClinicWaitlistEntry(
        entry: WaitlistEntry.fromJson(json[r'entry'])!,
        patientId: mapValueOfType<String>(json, r'patient_id')!,
        patientName: mapValueOfType<String>(json, r'patient_name'),
        priority: mapValueOfType<int>(json, r'priority')!,
      );
    }
    return null;
  }

  static List<ClinicWaitlistEntry> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ClinicWaitlistEntry>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ClinicWaitlistEntry.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ClinicWaitlistEntry> mapFromJson(dynamic json) {
    final map = <String, ClinicWaitlistEntry>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ClinicWaitlistEntry.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ClinicWaitlistEntry-objects as value to a dart map
  static Map<String, List<ClinicWaitlistEntry>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ClinicWaitlistEntry>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ClinicWaitlistEntry.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'entry',
    'patient_id',
    'priority',
  };
}
