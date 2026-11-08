//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class TermList {
  /// Returns a new [TermList] instance.
  TermList({
    this.terms = const [],
  });

  List<Term> terms;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is TermList && _deepEquality.equals(other.terms, terms);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (terms.hashCode);

  @override
  String toString() => 'TermList[terms=$terms]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'terms'] = this.terms;
    return json;
  }

  /// Returns a new [TermList] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static TermList? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'terms'),
            'Required key "TermList[terms]" is missing from JSON.');
        assert(json[r'terms'] != null,
            'Required key "TermList[terms]" has a null value in JSON.');
        return true;
      }());

      return TermList(
        terms: Term.listFromJson(json[r'terms']),
      );
    }
    return null;
  }

  static List<TermList> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <TermList>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = TermList.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, TermList> mapFromJson(dynamic json) {
    final map = <String, TermList>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = TermList.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of TermList-objects as value to a dart map
  static Map<String, List<TermList>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<TermList>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = TermList.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'terms',
  };
}
