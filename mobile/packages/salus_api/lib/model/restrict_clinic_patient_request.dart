//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class RestrictClinicPatientRequest {
  /// Returns a new [RestrictClinicPatientRequest] instance.
  RestrictClinicPatientRequest({
    this.dataRequestId,
  });

  /// The request it answers; absent for none.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? dataRequestId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is RestrictClinicPatientRequest &&
          other.dataRequestId == dataRequestId;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (dataRequestId == null ? 0 : dataRequestId!.hashCode);

  @override
  String toString() =>
      'RestrictClinicPatientRequest[dataRequestId=$dataRequestId]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    if (this.dataRequestId != null) {
      json[r'dataRequestId'] = this.dataRequestId;
    } else {
      json[r'dataRequestId'] = null;
    }
    return json;
  }

  /// Returns a new [RestrictClinicPatientRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static RestrictClinicPatientRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        return true;
      }());

      return RestrictClinicPatientRequest(
        dataRequestId: mapValueOfType<String>(json, r'dataRequestId'),
      );
    }
    return null;
  }

  static List<RestrictClinicPatientRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <RestrictClinicPatientRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = RestrictClinicPatientRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, RestrictClinicPatientRequest> mapFromJson(dynamic json) {
    final map = <String, RestrictClinicPatientRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = RestrictClinicPatientRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of RestrictClinicPatientRequest-objects as value to a dart map
  static Map<String, List<RestrictClinicPatientRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<RestrictClinicPatientRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = RestrictClinicPatientRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{};
}
