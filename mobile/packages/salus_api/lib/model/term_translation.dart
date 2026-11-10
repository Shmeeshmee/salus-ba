//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class TermTranslation {
  /// Returns a new [TermTranslation] instance.
  TermTranslation({
    required this.termId,
    required this.language,
    required this.review,
    required this.versionId,
    required this.lastUpdated,
  });

  String termId;

  TermLanguage language;

  Review review;

  /// For If-Match on an edit.
  ///
  /// Minimum value: 1
  int versionId;

  DateTime lastUpdated;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is TermTranslation &&
          other.termId == termId &&
          other.language == language &&
          other.review == review &&
          other.versionId == versionId &&
          other.lastUpdated == lastUpdated;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (termId.hashCode) +
      (language.hashCode) +
      (review.hashCode) +
      (versionId.hashCode) +
      (lastUpdated.hashCode);

  @override
  String toString() =>
      'TermTranslation[termId=$termId, language=$language, review=$review, versionId=$versionId, lastUpdated=$lastUpdated]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'termId'] = this.termId;
    json[r'language'] = this.language;
    json[r'review'] = this.review;
    json[r'versionId'] = this.versionId;
    json[r'lastUpdated'] = this.lastUpdated.toUtc().toIso8601String();
    return json;
  }

  /// Returns a new [TermTranslation] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static TermTranslation? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'termId'),
            'Required key "TermTranslation[termId]" is missing from JSON.');
        assert(json[r'termId'] != null,
            'Required key "TermTranslation[termId]" has a null value in JSON.');
        assert(json.containsKey(r'language'),
            'Required key "TermTranslation[language]" is missing from JSON.');
        assert(json[r'language'] != null,
            'Required key "TermTranslation[language]" has a null value in JSON.');
        assert(json.containsKey(r'review'),
            'Required key "TermTranslation[review]" is missing from JSON.');
        assert(json[r'review'] != null,
            'Required key "TermTranslation[review]" has a null value in JSON.');
        assert(json.containsKey(r'versionId'),
            'Required key "TermTranslation[versionId]" is missing from JSON.');
        assert(json[r'versionId'] != null,
            'Required key "TermTranslation[versionId]" has a null value in JSON.');
        assert(json.containsKey(r'lastUpdated'),
            'Required key "TermTranslation[lastUpdated]" is missing from JSON.');
        assert(json[r'lastUpdated'] != null,
            'Required key "TermTranslation[lastUpdated]" has a null value in JSON.');
        return true;
      }());

      return TermTranslation(
        termId: mapValueOfType<String>(json, r'termId')!,
        language: TermLanguage.fromJson(json[r'language'])!,
        review: Review.fromJson(json[r'review'])!,
        versionId: mapValueOfType<int>(json, r'versionId')!,
        lastUpdated: mapDateTime(json, r'lastUpdated', r'')!,
      );
    }
    return null;
  }

  static List<TermTranslation> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <TermTranslation>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = TermTranslation.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, TermTranslation> mapFromJson(dynamic json) {
    final map = <String, TermTranslation>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = TermTranslation.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of TermTranslation-objects as value to a dart map
  static Map<String, List<TermTranslation>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<TermTranslation>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = TermTranslation.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'termId',
    'language',
    'review',
    'versionId',
    'lastUpdated',
  };
}
