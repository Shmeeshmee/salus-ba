//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class CarePlan {
  /// Returns a new [CarePlan] instance.
  CarePlan({
    required this.id,
    required this.organizationId,
    required this.patientId,
    required this.status,
    required this.title,
    this.quoteId,
    this.startsOn,
    this.steps = const [],
    required this.createdAt,
    this.bookedAt,
    this.revokedAt,
    required this.versionId,
  });

  String id;

  String organizationId;

  String patientId;

  CarePlanStatus status;

  String title;

  /// Its quote (package X1); absent for none.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? quoteId;

  /// Day 0 of the steps; absent for a draft.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? startsOn;

  List<CarePlanStep> steps;

  DateTime createdAt;

  /// Absent for a draft.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? bookedAt;

  /// Absent unless cancelled.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? revokedAt;

  int versionId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is CarePlan &&
          other.id == id &&
          other.organizationId == organizationId &&
          other.patientId == patientId &&
          other.status == status &&
          other.title == title &&
          other.quoteId == quoteId &&
          other.startsOn == startsOn &&
          _deepEquality.equals(other.steps, steps) &&
          other.createdAt == createdAt &&
          other.bookedAt == bookedAt &&
          other.revokedAt == revokedAt &&
          other.versionId == versionId;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (organizationId.hashCode) +
      (patientId.hashCode) +
      (status.hashCode) +
      (title.hashCode) +
      (quoteId == null ? 0 : quoteId!.hashCode) +
      (startsOn == null ? 0 : startsOn!.hashCode) +
      (steps.hashCode) +
      (createdAt.hashCode) +
      (bookedAt == null ? 0 : bookedAt!.hashCode) +
      (revokedAt == null ? 0 : revokedAt!.hashCode) +
      (versionId.hashCode);

  @override
  String toString() =>
      'CarePlan[id=$id, organizationId=$organizationId, patientId=$patientId, status=$status, title=$title, quoteId=$quoteId, startsOn=$startsOn, steps=$steps, createdAt=$createdAt, bookedAt=$bookedAt, revokedAt=$revokedAt, versionId=$versionId]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'organizationId'] = this.organizationId;
    json[r'patientId'] = this.patientId;
    json[r'status'] = this.status;
    json[r'title'] = this.title;
    if (this.quoteId != null) {
      json[r'quoteId'] = this.quoteId;
    } else {
      json[r'quoteId'] = null;
    }
    if (this.startsOn != null) {
      json[r'startsOn'] = _dateFormatter.format(this.startsOn!);
    } else {
      json[r'startsOn'] = null;
    }
    json[r'steps'] = this.steps;
    json[r'createdAt'] = this.createdAt.toUtc().toIso8601String();
    if (this.bookedAt != null) {
      json[r'bookedAt'] = this.bookedAt!.toUtc().toIso8601String();
    } else {
      json[r'bookedAt'] = null;
    }
    if (this.revokedAt != null) {
      json[r'revokedAt'] = this.revokedAt!.toUtc().toIso8601String();
    } else {
      json[r'revokedAt'] = null;
    }
    json[r'versionId'] = this.versionId;
    return json;
  }

  /// Returns a new [CarePlan] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static CarePlan? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "CarePlan[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "CarePlan[id]" has a null value in JSON.');
        assert(json.containsKey(r'organizationId'),
            'Required key "CarePlan[organizationId]" is missing from JSON.');
        assert(json[r'organizationId'] != null,
            'Required key "CarePlan[organizationId]" has a null value in JSON.');
        assert(json.containsKey(r'patientId'),
            'Required key "CarePlan[patientId]" is missing from JSON.');
        assert(json[r'patientId'] != null,
            'Required key "CarePlan[patientId]" has a null value in JSON.');
        assert(json.containsKey(r'status'),
            'Required key "CarePlan[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "CarePlan[status]" has a null value in JSON.');
        assert(json.containsKey(r'title'),
            'Required key "CarePlan[title]" is missing from JSON.');
        assert(json[r'title'] != null,
            'Required key "CarePlan[title]" has a null value in JSON.');
        assert(json.containsKey(r'steps'),
            'Required key "CarePlan[steps]" is missing from JSON.');
        assert(json[r'steps'] != null,
            'Required key "CarePlan[steps]" has a null value in JSON.');
        assert(json.containsKey(r'createdAt'),
            'Required key "CarePlan[createdAt]" is missing from JSON.');
        assert(json[r'createdAt'] != null,
            'Required key "CarePlan[createdAt]" has a null value in JSON.');
        assert(json.containsKey(r'versionId'),
            'Required key "CarePlan[versionId]" is missing from JSON.');
        assert(json[r'versionId'] != null,
            'Required key "CarePlan[versionId]" has a null value in JSON.');
        return true;
      }());

      return CarePlan(
        id: mapValueOfType<String>(json, r'id')!,
        organizationId: mapValueOfType<String>(json, r'organizationId')!,
        patientId: mapValueOfType<String>(json, r'patientId')!,
        status: CarePlanStatus.fromJson(json[r'status'])!,
        title: mapValueOfType<String>(json, r'title')!,
        quoteId: mapValueOfType<String>(json, r'quoteId'),
        startsOn: mapDateTime(json, r'startsOn', r''),
        steps: CarePlanStep.listFromJson(json[r'steps']),
        createdAt: mapDateTime(json, r'createdAt', r'')!,
        bookedAt: mapDateTime(json, r'bookedAt', r''),
        revokedAt: mapDateTime(json, r'revokedAt', r''),
        versionId: mapValueOfType<int>(json, r'versionId')!,
      );
    }
    return null;
  }

  static List<CarePlan> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <CarePlan>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = CarePlan.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, CarePlan> mapFromJson(dynamic json) {
    final map = <String, CarePlan>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = CarePlan.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of CarePlan-objects as value to a dart map
  static Map<String, List<CarePlan>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<CarePlan>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = CarePlan.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'id',
    'organizationId',
    'patientId',
    'status',
    'title',
    'steps',
    'createdAt',
    'versionId',
  };
}
