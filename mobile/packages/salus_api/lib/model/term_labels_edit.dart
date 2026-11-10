//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class TermLabelsEdit {
  /// Returns a new [TermLabelsEdit] instance.
  TermLabelsEdit({
    required this.preferred,
    this.synonyms = const [],
  });

  String preferred;

  List<String> synonyms;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is TermLabelsEdit &&
          other.preferred == preferred &&
          _deepEquality.equals(other.synonyms, synonyms);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (preferred.hashCode) + (synonyms.hashCode);

  @override
  String toString() =>
      'TermLabelsEdit[preferred=$preferred, synonyms=$synonyms]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'preferred'] = this.preferred;
    json[r'synonyms'] = this.synonyms;
    return json;
  }

  /// Returns a new [TermLabelsEdit] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static TermLabelsEdit? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'preferred'),
            'Required key "TermLabelsEdit[preferred]" is missing from JSON.');
        assert(json[r'preferred'] != null,
            'Required key "TermLabelsEdit[preferred]" has a null value in JSON.');
        assert(json.containsKey(r'synonyms'),
            'Required key "TermLabelsEdit[synonyms]" is missing from JSON.');
        assert(json[r'synonyms'] != null,
            'Required key "TermLabelsEdit[synonyms]" has a null value in JSON.');
        return true;
      }());

      return TermLabelsEdit(
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

  static List<TermLabelsEdit> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <TermLabelsEdit>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = TermLabelsEdit.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, TermLabelsEdit> mapFromJson(dynamic json) {
    final map = <String, TermLabelsEdit>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = TermLabelsEdit.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of TermLabelsEdit-objects as value to a dart map
  static Map<String, List<TermLabelsEdit>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<TermLabelsEdit>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = TermLabelsEdit.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'preferred',
    'synonyms',
  };
}
