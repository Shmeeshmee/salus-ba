//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class PatientImportRow {
  /// Returns a new [PatientImportRow] instance.
  PatientImportRow({
    required this.line,
    this.patient,
    required this.duplicate,
    this.errors = const [],
  });

  /// The row's line in the file
  int line;

  /// The profile the row would store; absent for a row with errors.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  ImportedPatient? patient;

  /// A patient of the same name and birth date is the clinic's already, or on an earlier row.
  bool duplicate;

  /// Codes of what is wrong with the row; empty when it is valid.
  List<PatientImportRowErrorsEnum> errors;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is PatientImportRow &&
          other.line == line &&
          other.patient == patient &&
          other.duplicate == duplicate &&
          _deepEquality.equals(other.errors, errors);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (line.hashCode) +
      (patient == null ? 0 : patient!.hashCode) +
      (duplicate.hashCode) +
      (errors.hashCode);

  @override
  String toString() =>
      'PatientImportRow[line=$line, patient=$patient, duplicate=$duplicate, errors=$errors]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'line'] = this.line;
    if (this.patient != null) {
      json[r'patient'] = this.patient;
    } else {
      json[r'patient'] = null;
    }
    json[r'duplicate'] = this.duplicate;
    json[r'errors'] = this.errors;
    return json;
  }

  /// Returns a new [PatientImportRow] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static PatientImportRow? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'line'),
            'Required key "PatientImportRow[line]" is missing from JSON.');
        assert(json[r'line'] != null,
            'Required key "PatientImportRow[line]" has a null value in JSON.');
        assert(json.containsKey(r'duplicate'),
            'Required key "PatientImportRow[duplicate]" is missing from JSON.');
        assert(json[r'duplicate'] != null,
            'Required key "PatientImportRow[duplicate]" has a null value in JSON.');
        assert(json.containsKey(r'errors'),
            'Required key "PatientImportRow[errors]" is missing from JSON.');
        assert(json[r'errors'] != null,
            'Required key "PatientImportRow[errors]" has a null value in JSON.');
        return true;
      }());

      return PatientImportRow(
        line: mapValueOfType<int>(json, r'line')!,
        patient: ImportedPatient.fromJson(json[r'patient']),
        duplicate: mapValueOfType<bool>(json, r'duplicate')!,
        errors: PatientImportRowErrorsEnum.listFromJson(json[r'errors']),
      );
    }
    return null;
  }

  static List<PatientImportRow> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <PatientImportRow>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = PatientImportRow.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, PatientImportRow> mapFromJson(dynamic json) {
    final map = <String, PatientImportRow>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = PatientImportRow.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of PatientImportRow-objects as value to a dart map
  static Map<String, List<PatientImportRow>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<PatientImportRow>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = PatientImportRow.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'line',
    'duplicate',
    'errors',
  };
}

enum PatientImportRowErrorsEnum {
  givenNameInvalid._(r'given_name_invalid'),
  familyNameInvalid._(r'family_name_invalid'),
  birthDateInvalid._(r'birth_date_invalid'),
  phoneInvalid._(r'phone_invalid'),
  ;

  /// Instantiate a new enum with the provided value.
  const PatientImportRowErrorsEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [PatientImportRowErrorsEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static PatientImportRowErrorsEnum? fromJson(dynamic value) =>
      PatientImportRowErrorsEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [PatientImportRowErrorsEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<PatientImportRowErrorsEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <PatientImportRowErrorsEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = PatientImportRowErrorsEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [PatientImportRowErrorsEnum] to String,
/// and [decode] dynamic data back to [PatientImportRowErrorsEnum].
class PatientImportRowErrorsEnumTypeTransformer {
  factory PatientImportRowErrorsEnumTypeTransformer() =>
      _instance ??= const PatientImportRowErrorsEnumTypeTransformer._();

  const PatientImportRowErrorsEnumTypeTransformer._();

  String encode(PatientImportRowErrorsEnum data) => data._value;

  /// Returns the instance of [PatientImportRowErrorsEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  PatientImportRowErrorsEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is PatientImportRowErrorsEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'given_name_invalid':
          return PatientImportRowErrorsEnum.givenNameInvalid;
        case r'family_name_invalid':
          return PatientImportRowErrorsEnum.familyNameInvalid;
        case r'birth_date_invalid':
          return PatientImportRowErrorsEnum.birthDateInvalid;
        case r'phone_invalid':
          return PatientImportRowErrorsEnum.phoneInvalid;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static PatientImportRowErrorsEnumTypeTransformer? _instance;
}
