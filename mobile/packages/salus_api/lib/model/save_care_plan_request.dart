//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SaveCarePlanRequest {
  /// Returns a new [SaveCarePlanRequest] instance.
  SaveCarePlanRequest({
    this.patientId,
    required this.title,
    this.quoteId,
    this.steps = const [],
  });

  /// Required when writing a new plan; ignored on a change.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? patientId;

  String title;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? quoteId;

  List<SaveCarePlanRequestStepsInner> steps;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SaveCarePlanRequest &&
          other.patientId == patientId &&
          other.title == title &&
          other.quoteId == quoteId &&
          _deepEquality.equals(other.steps, steps);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (patientId == null ? 0 : patientId!.hashCode) +
      (title.hashCode) +
      (quoteId == null ? 0 : quoteId!.hashCode) +
      (steps.hashCode);

  @override
  String toString() =>
      'SaveCarePlanRequest[patientId=$patientId, title=$title, quoteId=$quoteId, steps=$steps]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    if (this.patientId != null) {
      json[r'patientId'] = this.patientId;
    } else {
      json[r'patientId'] = null;
    }
    json[r'title'] = this.title;
    if (this.quoteId != null) {
      json[r'quoteId'] = this.quoteId;
    } else {
      json[r'quoteId'] = null;
    }
    json[r'steps'] = this.steps;
    return json;
  }

  /// Returns a new [SaveCarePlanRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SaveCarePlanRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'title'),
            'Required key "SaveCarePlanRequest[title]" is missing from JSON.');
        assert(json[r'title'] != null,
            'Required key "SaveCarePlanRequest[title]" has a null value in JSON.');
        assert(json.containsKey(r'steps'),
            'Required key "SaveCarePlanRequest[steps]" is missing from JSON.');
        assert(json[r'steps'] != null,
            'Required key "SaveCarePlanRequest[steps]" has a null value in JSON.');
        return true;
      }());

      return SaveCarePlanRequest(
        patientId: mapValueOfType<String>(json, r'patientId'),
        title: mapValueOfType<String>(json, r'title')!,
        quoteId: mapValueOfType<String>(json, r'quoteId'),
        steps: SaveCarePlanRequestStepsInner.listFromJson(json[r'steps']),
      );
    }
    return null;
  }

  static List<SaveCarePlanRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SaveCarePlanRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SaveCarePlanRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SaveCarePlanRequest> mapFromJson(dynamic json) {
    final map = <String, SaveCarePlanRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SaveCarePlanRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SaveCarePlanRequest-objects as value to a dart map
  static Map<String, List<SaveCarePlanRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SaveCarePlanRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SaveCarePlanRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'title',
    'steps',
  };
}
