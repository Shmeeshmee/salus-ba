//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class RequestQuestionnaireRequest {
  /// Returns a new [RequestQuestionnaireRequest] instance.
  RequestQuestionnaireRequest({
    required this.questionnaireId,
  });

  /// The active version to ask.
  String questionnaireId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is RequestQuestionnaireRequest &&
          other.questionnaireId == questionnaireId;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (questionnaireId.hashCode);

  @override
  String toString() =>
      'RequestQuestionnaireRequest[questionnaireId=$questionnaireId]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'questionnaireId'] = this.questionnaireId;
    return json;
  }

  /// Returns a new [RequestQuestionnaireRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static RequestQuestionnaireRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'questionnaireId'),
            'Required key "RequestQuestionnaireRequest[questionnaireId]" is missing from JSON.');
        assert(json[r'questionnaireId'] != null,
            'Required key "RequestQuestionnaireRequest[questionnaireId]" has a null value in JSON.');
        return true;
      }());

      return RequestQuestionnaireRequest(
        questionnaireId: mapValueOfType<String>(json, r'questionnaireId')!,
      );
    }
    return null;
  }

  static List<RequestQuestionnaireRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <RequestQuestionnaireRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = RequestQuestionnaireRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, RequestQuestionnaireRequest> mapFromJson(dynamic json) {
    final map = <String, RequestQuestionnaireRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = RequestQuestionnaireRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of RequestQuestionnaireRequest-objects as value to a dart map
  static Map<String, List<RequestQuestionnaireRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<RequestQuestionnaireRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = RequestQuestionnaireRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'questionnaireId',
  };
}
