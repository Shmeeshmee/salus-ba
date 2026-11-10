//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class TranscriptTerms {
  /// Returns a new [TranscriptTerms] instance.
  TranscriptTerms({
    this.original = const [],
    this.translation = const [],
  });

  /// In `original`, in the speaker's language.
  List<TermSpan> original;

  /// In `translation`, in the listener's language.
  List<TermSpan> translation;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is TranscriptTerms &&
          _deepEquality.equals(other.original, original) &&
          _deepEquality.equals(other.translation, translation);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (original.hashCode) + (translation.hashCode);

  @override
  String toString() =>
      'TranscriptTerms[original=$original, translation=$translation]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'original'] = this.original;
    json[r'translation'] = this.translation;
    return json;
  }

  /// Returns a new [TranscriptTerms] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static TranscriptTerms? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        return true;
      }());

      return TranscriptTerms(
        original: TermSpan.listFromJson(json[r'original']),
        translation: TermSpan.listFromJson(json[r'translation']),
      );
    }
    return null;
  }

  static List<TranscriptTerms> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <TranscriptTerms>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = TranscriptTerms.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, TranscriptTerms> mapFromJson(dynamic json) {
    final map = <String, TranscriptTerms>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = TranscriptTerms.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of TranscriptTerms-objects as value to a dart map
  static Map<String, List<TranscriptTerms>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<TranscriptTerms>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = TranscriptTerms.listFromJson(
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
