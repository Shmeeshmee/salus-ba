//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class MedicationStatementInput {
  /// Returns a new [MedicationStatementInput] instance.
  MedicationStatementInput({
    this.termId,
    required this.text,
    required this.status,
    required this.dosage,
  });

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? termId;

  String text;

  MedicationStatus status;

  Dosage dosage;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is MedicationStatementInput &&
          other.termId == termId &&
          other.text == text &&
          other.status == status &&
          other.dosage == dosage;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (termId == null ? 0 : termId!.hashCode) +
      (text.hashCode) +
      (status.hashCode) +
      (dosage.hashCode);

  @override
  String toString() =>
      'MedicationStatementInput[termId=$termId, text=$text, status=$status, dosage=$dosage]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    if (this.termId != null) {
      json[r'termId'] = this.termId;
    } else {
      json[r'termId'] = null;
    }
    json[r'text'] = this.text;
    json[r'status'] = this.status;
    json[r'dosage'] = this.dosage;
    return json;
  }

  /// Returns a new [MedicationStatementInput] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static MedicationStatementInput? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'text'),
            'Required key "MedicationStatementInput[text]" is missing from JSON.');
        assert(json[r'text'] != null,
            'Required key "MedicationStatementInput[text]" has a null value in JSON.');
        assert(json.containsKey(r'status'),
            'Required key "MedicationStatementInput[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "MedicationStatementInput[status]" has a null value in JSON.');
        assert(json.containsKey(r'dosage'),
            'Required key "MedicationStatementInput[dosage]" is missing from JSON.');
        assert(json[r'dosage'] != null,
            'Required key "MedicationStatementInput[dosage]" has a null value in JSON.');
        return true;
      }());

      return MedicationStatementInput(
        termId: mapValueOfType<String>(json, r'termId'),
        text: mapValueOfType<String>(json, r'text')!,
        status: MedicationStatus.fromJson(json[r'status'])!,
        dosage: Dosage.fromJson(json[r'dosage'])!,
      );
    }
    return null;
  }

  static List<MedicationStatementInput> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <MedicationStatementInput>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = MedicationStatementInput.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, MedicationStatementInput> mapFromJson(dynamic json) {
    final map = <String, MedicationStatementInput>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = MedicationStatementInput.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of MedicationStatementInput-objects as value to a dart map
  static Map<String, List<MedicationStatementInput>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<MedicationStatementInput>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = MedicationStatementInput.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'text',
    'status',
    'dosage',
  };
}
