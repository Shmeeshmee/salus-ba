//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class StartQuestionnaireRequest {
  /// Returns a new [StartQuestionnaireRequest] instance.
  StartQuestionnaireRequest({
    required this.language,
  });

  Language language;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is StartQuestionnaireRequest && other.language == language;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (language.hashCode);

  @override
  String toString() => 'StartQuestionnaireRequest[language=$language]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'language'] = this.language;
    return json;
  }

  /// Returns a new [StartQuestionnaireRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static StartQuestionnaireRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'language'),
            'Required key "StartQuestionnaireRequest[language]" is missing from JSON.');
        assert(json[r'language'] != null,
            'Required key "StartQuestionnaireRequest[language]" has a null value in JSON.');
        return true;
      }());

      return StartQuestionnaireRequest(
        language: Language.fromJson(json[r'language'])!,
      );
    }
    return null;
  }

  static List<StartQuestionnaireRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <StartQuestionnaireRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = StartQuestionnaireRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, StartQuestionnaireRequest> mapFromJson(dynamic json) {
    final map = <String, StartQuestionnaireRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = StartQuestionnaireRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of StartQuestionnaireRequest-objects as value to a dart map
  static Map<String, List<StartQuestionnaireRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<StartQuestionnaireRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = StartQuestionnaireRequest.listFromJson(
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
  };
}
