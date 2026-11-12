//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class DiagnosisGroupName {
  /// Returns a new [DiagnosisGroupName] instance.
  DiagnosisGroupName({
    required this.code,
    this.parent,
    required this.language,
    required this.text,
    required this.review,
    required this.versionId,
  });

  String code;

  /// The group of a subgroup; absent for a group and in a write's answer.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? parent;

  Language language;

  String text;

  Review review;

  int versionId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DiagnosisGroupName &&
          other.code == code &&
          other.parent == parent &&
          other.language == language &&
          other.text == text &&
          other.review == review &&
          other.versionId == versionId;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (code.hashCode) +
      (parent == null ? 0 : parent!.hashCode) +
      (language.hashCode) +
      (text.hashCode) +
      (review.hashCode) +
      (versionId.hashCode);

  @override
  String toString() =>
      'DiagnosisGroupName[code=$code, parent=$parent, language=$language, text=$text, review=$review, versionId=$versionId]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'code'] = this.code;
    if (this.parent != null) {
      json[r'parent'] = this.parent;
    } else {
      json[r'parent'] = null;
    }
    json[r'language'] = this.language;
    json[r'text'] = this.text;
    json[r'review'] = this.review;
    json[r'versionId'] = this.versionId;
    return json;
  }

  /// Returns a new [DiagnosisGroupName] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static DiagnosisGroupName? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'code'),
            'Required key "DiagnosisGroupName[code]" is missing from JSON.');
        assert(json[r'code'] != null,
            'Required key "DiagnosisGroupName[code]" has a null value in JSON.');
        assert(json.containsKey(r'language'),
            'Required key "DiagnosisGroupName[language]" is missing from JSON.');
        assert(json[r'language'] != null,
            'Required key "DiagnosisGroupName[language]" has a null value in JSON.');
        assert(json.containsKey(r'text'),
            'Required key "DiagnosisGroupName[text]" is missing from JSON.');
        assert(json[r'text'] != null,
            'Required key "DiagnosisGroupName[text]" has a null value in JSON.');
        assert(json.containsKey(r'review'),
            'Required key "DiagnosisGroupName[review]" is missing from JSON.');
        assert(json[r'review'] != null,
            'Required key "DiagnosisGroupName[review]" has a null value in JSON.');
        assert(json.containsKey(r'versionId'),
            'Required key "DiagnosisGroupName[versionId]" is missing from JSON.');
        assert(json[r'versionId'] != null,
            'Required key "DiagnosisGroupName[versionId]" has a null value in JSON.');
        return true;
      }());

      return DiagnosisGroupName(
        code: mapValueOfType<String>(json, r'code')!,
        parent: mapValueOfType<String>(json, r'parent'),
        language: Language.fromJson(json[r'language'])!,
        text: mapValueOfType<String>(json, r'text')!,
        review: Review.fromJson(json[r'review'])!,
        versionId: mapValueOfType<int>(json, r'versionId')!,
      );
    }
    return null;
  }

  static List<DiagnosisGroupName> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DiagnosisGroupName>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DiagnosisGroupName.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, DiagnosisGroupName> mapFromJson(dynamic json) {
    final map = <String, DiagnosisGroupName>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = DiagnosisGroupName.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of DiagnosisGroupName-objects as value to a dart map
  static Map<String, List<DiagnosisGroupName>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<DiagnosisGroupName>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = DiagnosisGroupName.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'code',
    'language',
    'text',
    'review',
    'versionId',
  };
}
