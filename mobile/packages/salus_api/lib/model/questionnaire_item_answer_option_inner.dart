//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class QuestionnaireItemAnswerOptionInner {
  /// Returns a new [QuestionnaireItemAnswerOptionInner] instance.
  QuestionnaireItemAnswerOptionInner({
    required this.valueString,
  });

  String valueString;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is QuestionnaireItemAnswerOptionInner &&
          other.valueString == valueString;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (valueString.hashCode);

  @override
  String toString() =>
      'QuestionnaireItemAnswerOptionInner[valueString=$valueString]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'valueString'] = this.valueString;
    return json;
  }

  /// Returns a new [QuestionnaireItemAnswerOptionInner] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static QuestionnaireItemAnswerOptionInner? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'valueString'),
            'Required key "QuestionnaireItemAnswerOptionInner[valueString]" is missing from JSON.');
        assert(json[r'valueString'] != null,
            'Required key "QuestionnaireItemAnswerOptionInner[valueString]" has a null value in JSON.');
        return true;
      }());

      return QuestionnaireItemAnswerOptionInner(
        valueString: mapValueOfType<String>(json, r'valueString')!,
      );
    }
    return null;
  }

  static List<QuestionnaireItemAnswerOptionInner> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <QuestionnaireItemAnswerOptionInner>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = QuestionnaireItemAnswerOptionInner.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, QuestionnaireItemAnswerOptionInner> mapFromJson(
      dynamic json) {
    final map = <String, QuestionnaireItemAnswerOptionInner>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = QuestionnaireItemAnswerOptionInner.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of QuestionnaireItemAnswerOptionInner-objects as value to a dart map
  static Map<String, List<QuestionnaireItemAnswerOptionInner>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<QuestionnaireItemAnswerOptionInner>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = QuestionnaireItemAnswerOptionInner.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'valueString',
  };
}
