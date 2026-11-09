//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class DocumentTemplateTranslation {
  /// Returns a new [DocumentTemplateTranslation] instance.
  DocumentTemplateTranslation({
    required this.language,
    required this.title,
    this.fields = const {},
  });

  Language language;

  String title;

  /// Each field's texts by key; every field, and an instruction exactly where the original has one.
  Map<String, DocumentFieldTexts> fields;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DocumentTemplateTranslation &&
          other.language == language &&
          other.title == title &&
          _deepEquality.equals(other.fields, fields);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (language.hashCode) + (title.hashCode) + (fields.hashCode);

  @override
  String toString() =>
      'DocumentTemplateTranslation[language=$language, title=$title, fields=$fields]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'language'] = this.language;
    json[r'title'] = this.title;
    json[r'fields'] = this.fields;
    return json;
  }

  /// Returns a new [DocumentTemplateTranslation] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static DocumentTemplateTranslation? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'language'),
            'Required key "DocumentTemplateTranslation[language]" is missing from JSON.');
        assert(json[r'language'] != null,
            'Required key "DocumentTemplateTranslation[language]" has a null value in JSON.');
        assert(json.containsKey(r'title'),
            'Required key "DocumentTemplateTranslation[title]" is missing from JSON.');
        assert(json[r'title'] != null,
            'Required key "DocumentTemplateTranslation[title]" has a null value in JSON.');
        assert(json.containsKey(r'fields'),
            'Required key "DocumentTemplateTranslation[fields]" is missing from JSON.');
        assert(json[r'fields'] != null,
            'Required key "DocumentTemplateTranslation[fields]" has a null value in JSON.');
        return true;
      }());

      return DocumentTemplateTranslation(
        language: Language.fromJson(json[r'language'])!,
        title: mapValueOfType<String>(json, r'title')!,
        fields: DocumentFieldTexts.mapFromJson(json[r'fields']),
      );
    }
    return null;
  }

  static List<DocumentTemplateTranslation> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DocumentTemplateTranslation>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DocumentTemplateTranslation.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, DocumentTemplateTranslation> mapFromJson(dynamic json) {
    final map = <String, DocumentTemplateTranslation>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = DocumentTemplateTranslation.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of DocumentTemplateTranslation-objects as value to a dart map
  static Map<String, List<DocumentTemplateTranslation>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<DocumentTemplateTranslation>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = DocumentTemplateTranslation.listFromJson(
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
    'fields',
  };
}
