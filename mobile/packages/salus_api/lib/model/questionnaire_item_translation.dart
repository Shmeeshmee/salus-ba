//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class QuestionnaireItemTranslation {
  /// Returns a new [QuestionnaireItemTranslation] instance.
  QuestionnaireItemTranslation({
    this.text,
    this.help,
    this.options = const [],
  });

  /// The item's text, within its limits as in the original.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? text;

  /// The help text; for an item that has one.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? help;

  /// A choice item's option texts, by the options' position; as many as the item has.
  List<String> options;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is QuestionnaireItemTranslation &&
          other.text == text &&
          other.help == help &&
          _deepEquality.equals(other.options, options);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (text == null ? 0 : text!.hashCode) +
      (help == null ? 0 : help!.hashCode) +
      (options.hashCode);

  @override
  String toString() =>
      'QuestionnaireItemTranslation[text=$text, help=$help, options=$options]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    if (this.text != null) {
      json[r'text'] = this.text;
    } else {
      json[r'text'] = null;
    }
    if (this.help != null) {
      json[r'help'] = this.help;
    } else {
      json[r'help'] = null;
    }
    json[r'options'] = this.options;
    return json;
  }

  /// Returns a new [QuestionnaireItemTranslation] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static QuestionnaireItemTranslation? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        return true;
      }());

      return QuestionnaireItemTranslation(
        text: mapValueOfType<String>(json, r'text'),
        help: mapValueOfType<String>(json, r'help'),
        options: json[r'options'] is Iterable
            ? (json[r'options'] as Iterable)
                .cast<String>()
                .toList(growable: false)
            : const [],
      );
    }
    return null;
  }

  static List<QuestionnaireItemTranslation> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <QuestionnaireItemTranslation>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = QuestionnaireItemTranslation.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, QuestionnaireItemTranslation> mapFromJson(dynamic json) {
    final map = <String, QuestionnaireItemTranslation>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = QuestionnaireItemTranslation.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of QuestionnaireItemTranslation-objects as value to a dart map
  static Map<String, List<QuestionnaireItemTranslation>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<QuestionnaireItemTranslation>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = QuestionnaireItemTranslation.listFromJson(
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
