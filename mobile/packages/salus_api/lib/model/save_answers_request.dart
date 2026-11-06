//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SaveAnswersRequest {
  /// Returns a new [SaveAnswersRequest] instance.
  SaveAnswersRequest({
    this.answers = const [],
  });

  /// Every saved answer; a question left out is unanswered.
  List<Answer> answers;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SaveAnswersRequest &&
          _deepEquality.equals(other.answers, answers);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (answers.hashCode);

  @override
  String toString() => 'SaveAnswersRequest[answers=$answers]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'answers'] = this.answers;
    return json;
  }

  /// Returns a new [SaveAnswersRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SaveAnswersRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'answers'),
            'Required key "SaveAnswersRequest[answers]" is missing from JSON.');
        assert(json[r'answers'] != null,
            'Required key "SaveAnswersRequest[answers]" has a null value in JSON.');
        return true;
      }());

      return SaveAnswersRequest(
        answers: Answer.listFromJson(json[r'answers']),
      );
    }
    return null;
  }

  static List<SaveAnswersRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SaveAnswersRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SaveAnswersRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SaveAnswersRequest> mapFromJson(dynamic json) {
    final map = <String, SaveAnswersRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SaveAnswersRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SaveAnswersRequest-objects as value to a dart map
  static Map<String, List<SaveAnswersRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SaveAnswersRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SaveAnswersRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'answers',
  };
}
