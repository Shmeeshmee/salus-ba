//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ExplainedTermList {
  /// Returns a new [ExplainedTermList] instance.
  ExplainedTermList({
    this.explanations = const [],
  });

  /// One per term with an explanation in the language or in English, by term id.
  List<ExplainedTerm> explanations;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ExplainedTermList &&
          _deepEquality.equals(other.explanations, explanations);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (explanations.hashCode);

  @override
  String toString() => 'ExplainedTermList[explanations=$explanations]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'explanations'] = this.explanations;
    return json;
  }

  /// Returns a new [ExplainedTermList] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ExplainedTermList? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'explanations'),
            'Required key "ExplainedTermList[explanations]" is missing from JSON.');
        assert(json[r'explanations'] != null,
            'Required key "ExplainedTermList[explanations]" has a null value in JSON.');
        return true;
      }());

      return ExplainedTermList(
        explanations: ExplainedTerm.listFromJson(json[r'explanations']),
      );
    }
    return null;
  }

  static List<ExplainedTermList> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ExplainedTermList>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ExplainedTermList.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ExplainedTermList> mapFromJson(dynamic json) {
    final map = <String, ExplainedTermList>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ExplainedTermList.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ExplainedTermList-objects as value to a dart map
  static Map<String, List<ExplainedTermList>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ExplainedTermList>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ExplainedTermList.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'explanations',
  };
}
