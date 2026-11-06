//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Answer {
  /// Returns a new [Answer] instance.
  Answer({
    required this.linkId,
    this.values = const [],
  });

  String linkId;

  List<AnswerValue> values;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Answer &&
          other.linkId == linkId &&
          _deepEquality.equals(other.values, values);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (linkId.hashCode) + (values.hashCode);

  @override
  String toString() => 'Answer[linkId=$linkId, values=$values]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'linkId'] = this.linkId;
    json[r'values'] = this.values;
    return json;
  }

  /// Returns a new [Answer] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Answer? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'linkId'),
            'Required key "Answer[linkId]" is missing from JSON.');
        assert(json[r'linkId'] != null,
            'Required key "Answer[linkId]" has a null value in JSON.');
        assert(json.containsKey(r'values'),
            'Required key "Answer[values]" is missing from JSON.');
        assert(json[r'values'] != null,
            'Required key "Answer[values]" has a null value in JSON.');
        return true;
      }());

      return Answer(
        linkId: mapValueOfType<String>(json, r'linkId')!,
        values: AnswerValue.listFromJson(json[r'values']),
      );
    }
    return null;
  }

  static List<Answer> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Answer>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Answer.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Answer> mapFromJson(dynamic json) {
    final map = <String, Answer>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Answer.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Answer-objects as value to a dart map
  static Map<String, List<Answer>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Answer>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Answer.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'linkId',
    'values',
  };
}
