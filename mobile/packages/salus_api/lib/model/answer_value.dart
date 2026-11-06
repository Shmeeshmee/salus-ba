//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class AnswerValue {
  /// Returns a new [AnswerValue] instance.
  AnswerValue({
    this.valueString,
    this.valueBoolean,
    this.valueInteger,
    this.valueDecimal,
    this.valueDate,
  });

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? valueString;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  bool? valueBoolean;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  int? valueInteger;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  num? valueDecimal;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? valueDate;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is AnswerValue &&
          other.valueString == valueString &&
          other.valueBoolean == valueBoolean &&
          other.valueInteger == valueInteger &&
          other.valueDecimal == valueDecimal &&
          other.valueDate == valueDate;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (valueString == null ? 0 : valueString!.hashCode) +
      (valueBoolean == null ? 0 : valueBoolean!.hashCode) +
      (valueInteger == null ? 0 : valueInteger!.hashCode) +
      (valueDecimal == null ? 0 : valueDecimal!.hashCode) +
      (valueDate == null ? 0 : valueDate!.hashCode);

  @override
  String toString() =>
      'AnswerValue[valueString=$valueString, valueBoolean=$valueBoolean, valueInteger=$valueInteger, valueDecimal=$valueDecimal, valueDate=$valueDate]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    if (this.valueString != null) {
      json[r'valueString'] = this.valueString;
    } else {
      json[r'valueString'] = null;
    }
    if (this.valueBoolean != null) {
      json[r'valueBoolean'] = this.valueBoolean;
    } else {
      json[r'valueBoolean'] = null;
    }
    if (this.valueInteger != null) {
      json[r'valueInteger'] = this.valueInteger;
    } else {
      json[r'valueInteger'] = null;
    }
    if (this.valueDecimal != null) {
      json[r'valueDecimal'] = this.valueDecimal;
    } else {
      json[r'valueDecimal'] = null;
    }
    if (this.valueDate != null) {
      json[r'valueDate'] = _dateFormatter.format(this.valueDate!);
    } else {
      json[r'valueDate'] = null;
    }
    return json;
  }

  /// Returns a new [AnswerValue] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static AnswerValue? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        return true;
      }());

      return AnswerValue(
        valueString: mapValueOfType<String>(json, r'valueString'),
        valueBoolean: mapValueOfType<bool>(json, r'valueBoolean'),
        valueInteger: mapValueOfType<int>(json, r'valueInteger'),
        valueDecimal: json[r'valueDecimal'] == null
            ? null
            : num.parse('${json[r'valueDecimal']}'),
        valueDate: mapDateTime(json, r'valueDate', r''),
      );
    }
    return null;
  }

  static List<AnswerValue> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <AnswerValue>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = AnswerValue.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, AnswerValue> mapFromJson(dynamic json) {
    final map = <String, AnswerValue>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = AnswerValue.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of AnswerValue-objects as value to a dart map
  static Map<String, List<AnswerValue>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<AnswerValue>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = AnswerValue.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{};
}
