//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class TermReviewEntry {
  /// Returns a new [TermReviewEntry] instance.
  TermReviewEntry({
    required this.term,
    required this.status,
    this.translation,
    this.explanation,
  });

  Term term;

  ReviewStatus status;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  TermTranslation? translation;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  TermExplanation? explanation;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is TermReviewEntry &&
          other.term == term &&
          other.status == status &&
          other.translation == translation &&
          other.explanation == explanation;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (term.hashCode) +
      (status.hashCode) +
      (translation == null ? 0 : translation!.hashCode) +
      (explanation == null ? 0 : explanation!.hashCode);

  @override
  String toString() =>
      'TermReviewEntry[term=$term, status=$status, translation=$translation, explanation=$explanation]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'term'] = this.term;
    json[r'status'] = this.status;
    if (this.translation != null) {
      json[r'translation'] = this.translation;
    } else {
      json[r'translation'] = null;
    }
    if (this.explanation != null) {
      json[r'explanation'] = this.explanation;
    } else {
      json[r'explanation'] = null;
    }
    return json;
  }

  /// Returns a new [TermReviewEntry] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static TermReviewEntry? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'term'),
            'Required key "TermReviewEntry[term]" is missing from JSON.');
        assert(json[r'term'] != null,
            'Required key "TermReviewEntry[term]" has a null value in JSON.');
        assert(json.containsKey(r'status'),
            'Required key "TermReviewEntry[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "TermReviewEntry[status]" has a null value in JSON.');
        return true;
      }());

      return TermReviewEntry(
        term: Term.fromJson(json[r'term'])!,
        status: ReviewStatus.fromJson(json[r'status'])!,
        translation: TermTranslation.fromJson(json[r'translation']),
        explanation: TermExplanation.fromJson(json[r'explanation']),
      );
    }
    return null;
  }

  static List<TermReviewEntry> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <TermReviewEntry>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = TermReviewEntry.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, TermReviewEntry> mapFromJson(dynamic json) {
    final map = <String, TermReviewEntry>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = TermReviewEntry.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of TermReviewEntry-objects as value to a dart map
  static Map<String, List<TermReviewEntry>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<TermReviewEntry>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = TermReviewEntry.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'term',
    'status',
  };
}
