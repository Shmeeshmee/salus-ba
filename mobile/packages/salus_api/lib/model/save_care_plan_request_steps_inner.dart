//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SaveCarePlanRequestStepsInner {
  /// Returns a new [SaveCarePlanRequestStepsInner] instance.
  SaveCarePlanRequestStepsInner({
    required this.serviceId,
    required this.dayOffset,
    required this.visitMode,
  });

  String serviceId;

  /// Minimum value: 0
  /// Maximum value: 730
  int dayOffset;

  VisitMode visitMode;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SaveCarePlanRequestStepsInner &&
          other.serviceId == serviceId &&
          other.dayOffset == dayOffset &&
          other.visitMode == visitMode;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (serviceId.hashCode) + (dayOffset.hashCode) + (visitMode.hashCode);

  @override
  String toString() =>
      'SaveCarePlanRequestStepsInner[serviceId=$serviceId, dayOffset=$dayOffset, visitMode=$visitMode]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'serviceId'] = this.serviceId;
    json[r'dayOffset'] = this.dayOffset;
    json[r'visitMode'] = this.visitMode;
    return json;
  }

  /// Returns a new [SaveCarePlanRequestStepsInner] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SaveCarePlanRequestStepsInner? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'serviceId'),
            'Required key "SaveCarePlanRequestStepsInner[serviceId]" is missing from JSON.');
        assert(json[r'serviceId'] != null,
            'Required key "SaveCarePlanRequestStepsInner[serviceId]" has a null value in JSON.');
        assert(json.containsKey(r'dayOffset'),
            'Required key "SaveCarePlanRequestStepsInner[dayOffset]" is missing from JSON.');
        assert(json[r'dayOffset'] != null,
            'Required key "SaveCarePlanRequestStepsInner[dayOffset]" has a null value in JSON.');
        assert(json.containsKey(r'visitMode'),
            'Required key "SaveCarePlanRequestStepsInner[visitMode]" is missing from JSON.');
        assert(json[r'visitMode'] != null,
            'Required key "SaveCarePlanRequestStepsInner[visitMode]" has a null value in JSON.');
        return true;
      }());

      return SaveCarePlanRequestStepsInner(
        serviceId: mapValueOfType<String>(json, r'serviceId')!,
        dayOffset: mapValueOfType<int>(json, r'dayOffset')!,
        visitMode: VisitMode.fromJson(json[r'visitMode'])!,
      );
    }
    return null;
  }

  static List<SaveCarePlanRequestStepsInner> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SaveCarePlanRequestStepsInner>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SaveCarePlanRequestStepsInner.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SaveCarePlanRequestStepsInner> mapFromJson(dynamic json) {
    final map = <String, SaveCarePlanRequestStepsInner>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SaveCarePlanRequestStepsInner.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SaveCarePlanRequestStepsInner-objects as value to a dart map
  static Map<String, List<SaveCarePlanRequestStepsInner>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SaveCarePlanRequestStepsInner>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SaveCarePlanRequestStepsInner.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'serviceId',
    'dayOffset',
    'visitMode',
  };
}
