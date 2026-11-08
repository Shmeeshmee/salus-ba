//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class TermLabels {
  /// Returns a new [TermLabels] instance.
  TermLabels({
    required this.language,
    required this.preferred,
    this.synonyms = const [],
  });

  TermLanguage language;

  /// The preferred label in this language; the first alphabetically when none is marked.
  String preferred;

  /// The other labels in this language, alphabetically; empty when there are none.
  List<String> synonyms;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is TermLabels &&
          other.language == language &&
          other.preferred == preferred &&
          _deepEquality.equals(other.synonyms, synonyms);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (language.hashCode) + (preferred.hashCode) + (synonyms.hashCode);

  @override
  String toString() =>
      'TermLabels[language=$language, preferred=$preferred, synonyms=$synonyms]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'language'] = this.language;
    json[r'preferred'] = this.preferred;
    json[r'synonyms'] = this.synonyms;
    return json;
  }

  /// Returns a new [TermLabels] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static TermLabels? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'language'),
            'Required key "TermLabels[language]" is missing from JSON.');
        assert(json[r'language'] != null,
            'Required key "TermLabels[language]" has a null value in JSON.');
        assert(json.containsKey(r'preferred'),
            'Required key "TermLabels[preferred]" is missing from JSON.');
        assert(json[r'preferred'] != null,
            'Required key "TermLabels[preferred]" has a null value in JSON.');
        assert(json.containsKey(r'synonyms'),
            'Required key "TermLabels[synonyms]" is missing from JSON.');
        assert(json[r'synonyms'] != null,
            'Required key "TermLabels[synonyms]" has a null value in JSON.');
        return true;
      }());

      return TermLabels(
        language: TermLanguage.fromJson(json[r'language'])!,
        preferred: mapValueOfType<String>(json, r'preferred')!,
        synonyms: json[r'synonyms'] is Iterable
            ? (json[r'synonyms'] as Iterable)
                .cast<String>()
                .toList(growable: false)
            : const [],
      );
    }
    return null;
  }

  static List<TermLabels> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <TermLabels>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = TermLabels.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, TermLabels> mapFromJson(dynamic json) {
    final map = <String, TermLabels>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = TermLabels.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of TermLabels-objects as value to a dart map
  static Map<String, List<TermLabels>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<TermLabels>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = TermLabels.listFromJson(
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
    'preferred',
    'synonyms',
  };
}
