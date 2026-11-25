//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class CarePlanStep {
  /// Returns a new [CarePlanStep] instance.
  CarePlanStep({
    this.id,
    this.position,
    required this.serviceId,
    required this.dayOffset,
    required this.visitMode,
    this.appointmentId,
  });

  /// In a response only.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? id;

  /// In a response only.
  ///
  /// Minimum value: 1
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  int? position;

  /// A service of the clinic's price list.
  String serviceId;

  /// Days from the plan's start.
  ///
  /// Minimum value: 0
  /// Maximum value: 730
  int dayOffset;

  VisitMode visitMode;

  /// Its visit once booked; in a response only.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? appointmentId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is CarePlanStep &&
          other.id == id &&
          other.position == position &&
          other.serviceId == serviceId &&
          other.dayOffset == dayOffset &&
          other.visitMode == visitMode &&
          other.appointmentId == appointmentId;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id == null ? 0 : id!.hashCode) +
      (position == null ? 0 : position!.hashCode) +
      (serviceId.hashCode) +
      (dayOffset.hashCode) +
      (visitMode.hashCode) +
      (appointmentId == null ? 0 : appointmentId!.hashCode);

  @override
  String toString() =>
      'CarePlanStep[id=$id, position=$position, serviceId=$serviceId, dayOffset=$dayOffset, visitMode=$visitMode, appointmentId=$appointmentId]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    if (this.id != null) {
      json[r'id'] = this.id;
    } else {
      json[r'id'] = null;
    }
    if (this.position != null) {
      json[r'position'] = this.position;
    } else {
      json[r'position'] = null;
    }
    json[r'serviceId'] = this.serviceId;
    json[r'dayOffset'] = this.dayOffset;
    json[r'visitMode'] = this.visitMode;
    if (this.appointmentId != null) {
      json[r'appointmentId'] = this.appointmentId;
    } else {
      json[r'appointmentId'] = null;
    }
    return json;
  }

  /// Returns a new [CarePlanStep] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static CarePlanStep? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'serviceId'),
            'Required key "CarePlanStep[serviceId]" is missing from JSON.');
        assert(json[r'serviceId'] != null,
            'Required key "CarePlanStep[serviceId]" has a null value in JSON.');
        assert(json.containsKey(r'dayOffset'),
            'Required key "CarePlanStep[dayOffset]" is missing from JSON.');
        assert(json[r'dayOffset'] != null,
            'Required key "CarePlanStep[dayOffset]" has a null value in JSON.');
        assert(json.containsKey(r'visitMode'),
            'Required key "CarePlanStep[visitMode]" is missing from JSON.');
        assert(json[r'visitMode'] != null,
            'Required key "CarePlanStep[visitMode]" has a null value in JSON.');
        return true;
      }());

      return CarePlanStep(
        id: mapValueOfType<String>(json, r'id'),
        position: mapValueOfType<int>(json, r'position'),
        serviceId: mapValueOfType<String>(json, r'serviceId')!,
        dayOffset: mapValueOfType<int>(json, r'dayOffset')!,
        visitMode: VisitMode.fromJson(json[r'visitMode'])!,
        appointmentId: mapValueOfType<String>(json, r'appointmentId'),
      );
    }
    return null;
  }

  static List<CarePlanStep> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <CarePlanStep>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = CarePlanStep.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, CarePlanStep> mapFromJson(dynamic json) {
    final map = <String, CarePlanStep>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = CarePlanStep.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of CarePlanStep-objects as value to a dart map
  static Map<String, List<CarePlanStep>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<CarePlanStep>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = CarePlanStep.listFromJson(
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
