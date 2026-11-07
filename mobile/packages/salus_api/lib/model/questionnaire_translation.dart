//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class QuestionnaireTranslation {
  /// Returns a new [QuestionnaireTranslation] instance.
  QuestionnaireTranslation({
    required this.language,
    required this.title,
    this.description,
    this.items = const {},
  });

  Language language;

  /// Empty while not yet translated.
  String title;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? description;

  /// The items' texts by linkId; an item missing here is not yet translated.
  Map<String, QuestionnaireItemTranslation> items;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is QuestionnaireTranslation &&
          other.language == language &&
          other.title == title &&
          other.description == description &&
          _deepEquality.equals(other.items, items);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (language.hashCode) +
      (title.hashCode) +
      (description == null ? 0 : description!.hashCode) +
      (items.hashCode);

  @override
  String toString() =>
      'QuestionnaireTranslation[language=$language, title=$title, description=$description, items=$items]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'language'] = this.language;
    json[r'title'] = this.title;
    if (this.description != null) {
      json[r'description'] = this.description;
    } else {
      json[r'description'] = null;
    }
    json[r'items'] = this.items;
    return json;
  }

  /// Returns a new [QuestionnaireTranslation] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static QuestionnaireTranslation? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'language'),
            'Required key "QuestionnaireTranslation[language]" is missing from JSON.');
        assert(json[r'language'] != null,
            'Required key "QuestionnaireTranslation[language]" has a null value in JSON.');
        assert(json.containsKey(r'title'),
            'Required key "QuestionnaireTranslation[title]" is missing from JSON.');
        assert(json[r'title'] != null,
            'Required key "QuestionnaireTranslation[title]" has a null value in JSON.');
        assert(json.containsKey(r'items'),
            'Required key "QuestionnaireTranslation[items]" is missing from JSON.');
        assert(json[r'items'] != null,
            'Required key "QuestionnaireTranslation[items]" has a null value in JSON.');
        return true;
      }());

      return QuestionnaireTranslation(
        language: Language.fromJson(json[r'language'])!,
        title: mapValueOfType<String>(json, r'title')!,
        description: mapValueOfType<String>(json, r'description'),
        items: QuestionnaireItemTranslation.mapFromJson(json[r'items']),
      );
    }
    return null;
  }

  static List<QuestionnaireTranslation> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <QuestionnaireTranslation>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = QuestionnaireTranslation.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, QuestionnaireTranslation> mapFromJson(dynamic json) {
    final map = <String, QuestionnaireTranslation>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = QuestionnaireTranslation.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of QuestionnaireTranslation-objects as value to a dart map
  static Map<String, List<QuestionnaireTranslation>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<QuestionnaireTranslation>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = QuestionnaireTranslation.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'language',
    'title',
    'items',
  };
}
