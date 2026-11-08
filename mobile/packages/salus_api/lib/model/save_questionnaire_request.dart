//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SaveQuestionnaireRequest {
  /// Returns a new [SaveQuestionnaireRequest] instance.
  SaveQuestionnaireRequest({
    required this.title,
    this.description,
    this.item = const [],
    this.language,
    this.translations = const [],
  });

  /// Not blank.
  String title;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? description;

  /// At most 200 items in all, sections and instructions included.
  List<QuestionnaireItem> item;

  /// The original language; absent keeps the draft's (a new questionnaire's is the clinic's default language).
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  Language? language;

  /// The draft's translations, all of them: one per language, none in the original language. A translation may be incomplete while the version is a draft; publishing refuses it until complete. Absent keeps the draft's.
  List<QuestionnaireTranslation> translations;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SaveQuestionnaireRequest &&
          other.title == title &&
          other.description == description &&
          _deepEquality.equals(other.item, item) &&
          other.language == language &&
          _deepEquality.equals(other.translations, translations);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (title.hashCode) +
      (description == null ? 0 : description!.hashCode) +
      (item.hashCode) +
      (language == null ? 0 : language!.hashCode) +
      (translations.hashCode);

  @override
  String toString() =>
      'SaveQuestionnaireRequest[title=$title, description=$description, item=$item, language=$language, translations=$translations]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'title'] = this.title;
    if (this.description != null) {
      json[r'description'] = this.description;
    } else {
      json[r'description'] = null;
    }
    json[r'item'] = this.item;
    if (this.language != null) {
      json[r'language'] = this.language;
    } else {
      json[r'language'] = null;
    }
    json[r'translations'] = this.translations;
    return json;
  }

  /// Returns a new [SaveQuestionnaireRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SaveQuestionnaireRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'title'),
            'Required key "SaveQuestionnaireRequest[title]" is missing from JSON.');
        assert(json[r'title'] != null,
            'Required key "SaveQuestionnaireRequest[title]" has a null value in JSON.');
        assert(json.containsKey(r'item'),
            'Required key "SaveQuestionnaireRequest[item]" is missing from JSON.');
        assert(json[r'item'] != null,
            'Required key "SaveQuestionnaireRequest[item]" has a null value in JSON.');
        return true;
      }());

      return SaveQuestionnaireRequest(
        title: mapValueOfType<String>(json, r'title')!,
        description: mapValueOfType<String>(json, r'description'),
        item: QuestionnaireItem.listFromJson(json[r'item']),
        language: Language.fromJson(json[r'language']),
        translations:
            QuestionnaireTranslation.listFromJson(json[r'translations']),
      );
    }
    return null;
  }

  static List<SaveQuestionnaireRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SaveQuestionnaireRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SaveQuestionnaireRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SaveQuestionnaireRequest> mapFromJson(dynamic json) {
    final map = <String, SaveQuestionnaireRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SaveQuestionnaireRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SaveQuestionnaireRequest-objects as value to a dart map
  static Map<String, List<SaveQuestionnaireRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SaveQuestionnaireRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SaveQuestionnaireRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'title',
    'item',
  };
}
