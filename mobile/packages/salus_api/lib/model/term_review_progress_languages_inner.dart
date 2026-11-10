//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class TermReviewProgressLanguagesInner {
  /// Returns a new [TermReviewProgressLanguagesInner] instance.
  TermReviewProgressLanguagesInner({
    required this.language,
    required this.terms,
    required this.approved,
    required this.needsChanges,
  });

  TermLanguage language;

  /// Every term.
  int terms;

  int approved;

  int needsChanges;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is TermReviewProgressLanguagesInner &&
          other.language == language &&
          other.terms == terms &&
          other.approved == approved &&
          other.needsChanges == needsChanges;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (language.hashCode) +
      (terms.hashCode) +
      (approved.hashCode) +
      (needsChanges.hashCode);

  @override
  String toString() =>
      'TermReviewProgressLanguagesInner[language=$language, terms=$terms, approved=$approved, needsChanges=$needsChanges]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'language'] = this.language;
    json[r'terms'] = this.terms;
    json[r'approved'] = this.approved;
    json[r'needsChanges'] = this.needsChanges;
    return json;
  }

  /// Returns a new [TermReviewProgressLanguagesInner] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static TermReviewProgressLanguagesInner? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'language'),
            'Required key "TermReviewProgressLanguagesInner[language]" is missing from JSON.');
        assert(json[r'language'] != null,
            'Required key "TermReviewProgressLanguagesInner[language]" has a null value in JSON.');
        assert(json.containsKey(r'terms'),
            'Required key "TermReviewProgressLanguagesInner[terms]" is missing from JSON.');
        assert(json[r'terms'] != null,
            'Required key "TermReviewProgressLanguagesInner[terms]" has a null value in JSON.');
        assert(json.containsKey(r'approved'),
            'Required key "TermReviewProgressLanguagesInner[approved]" is missing from JSON.');
        assert(json[r'approved'] != null,
            'Required key "TermReviewProgressLanguagesInner[approved]" has a null value in JSON.');
        assert(json.containsKey(r'needsChanges'),
            'Required key "TermReviewProgressLanguagesInner[needsChanges]" is missing from JSON.');
        assert(json[r'needsChanges'] != null,
            'Required key "TermReviewProgressLanguagesInner[needsChanges]" has a null value in JSON.');
        return true;
      }());

      return TermReviewProgressLanguagesInner(
        language: TermLanguage.fromJson(json[r'language'])!,
        terms: mapValueOfType<int>(json, r'terms')!,
        approved: mapValueOfType<int>(json, r'approved')!,
        needsChanges: mapValueOfType<int>(json, r'needsChanges')!,
      );
    }
    return null;
  }

  static List<TermReviewProgressLanguagesInner> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <TermReviewProgressLanguagesInner>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = TermReviewProgressLanguagesInner.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, TermReviewProgressLanguagesInner> mapFromJson(
      dynamic json) {
    final map = <String, TermReviewProgressLanguagesInner>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = TermReviewProgressLanguagesInner.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of TermReviewProgressLanguagesInner-objects as value to a dart map
  static Map<String, List<TermReviewProgressLanguagesInner>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<TermReviewProgressLanguagesInner>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = TermReviewProgressLanguagesInner.listFromJson(
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
    'terms',
    'approved',
    'needsChanges',
  };
}
