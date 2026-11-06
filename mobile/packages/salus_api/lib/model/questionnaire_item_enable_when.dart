//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class QuestionnaireItemEnableWhen {
  /// Returns a new [QuestionnaireItemEnableWhen] instance.
  QuestionnaireItemEnableWhen({
    required this.question,
    this.answerBoolean,
    this.answerString,
  });

  String question;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  bool? answerBoolean;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? answerString;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is QuestionnaireItemEnableWhen &&
          other.question == question &&
          other.answerBoolean == answerBoolean &&
          other.answerString == answerString;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (question.hashCode) +
      (answerBoolean == null ? 0 : answerBoolean!.hashCode) +
      (answerString == null ? 0 : answerString!.hashCode);

  @override
  String toString() =>
      'QuestionnaireItemEnableWhen[question=$question, answerBoolean=$answerBoolean, answerString=$answerString]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'question'] = this.question;
    if (this.answerBoolean != null) {
      json[r'answerBoolean'] = this.answerBoolean;
    } else {
      json[r'answerBoolean'] = null;
    }
    if (this.answerString != null) {
      json[r'answerString'] = this.answerString;
    } else {
      json[r'answerString'] = null;
    }
    return json;
  }

  /// Returns a new [QuestionnaireItemEnableWhen] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static QuestionnaireItemEnableWhen? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'question'),
            'Required key "QuestionnaireItemEnableWhen[question]" is missing from JSON.');
        assert(json[r'question'] != null,
            'Required key "QuestionnaireItemEnableWhen[question]" has a null value in JSON.');
        return true;
      }());

      return QuestionnaireItemEnableWhen(
        question: mapValueOfType<String>(json, r'question')!,
        answerBoolean: mapValueOfType<bool>(json, r'answerBoolean'),
        answerString: mapValueOfType<String>(json, r'answerString'),
      );
    }
    return null;
  }

  static List<QuestionnaireItemEnableWhen> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <QuestionnaireItemEnableWhen>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = QuestionnaireItemEnableWhen.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, QuestionnaireItemEnableWhen> mapFromJson(dynamic json) {
    final map = <String, QuestionnaireItemEnableWhen>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = QuestionnaireItemEnableWhen.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of QuestionnaireItemEnableWhen-objects as value to a dart map
  static Map<String, List<QuestionnaireItemEnableWhen>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<QuestionnaireItemEnableWhen>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = QuestionnaireItemEnableWhen.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'question',
  };
}
