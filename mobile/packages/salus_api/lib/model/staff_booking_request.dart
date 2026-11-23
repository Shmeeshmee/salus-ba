//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class StaffBookingRequest {
  /// Returns a new [StaffBookingRequest] instance.
  StaffBookingRequest({
    required this.patientId,
    required this.slotId,
    this.serviceId,
  });

  String patientId;

  String slotId;

  /// The price-list service; omitted for none.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? serviceId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is StaffBookingRequest &&
          other.patientId == patientId &&
          other.slotId == slotId &&
          other.serviceId == serviceId;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (patientId.hashCode) +
      (slotId.hashCode) +
      (serviceId == null ? 0 : serviceId!.hashCode);

  @override
  String toString() =>
      'StaffBookingRequest[patientId=$patientId, slotId=$slotId, serviceId=$serviceId]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'patientId'] = this.patientId;
    json[r'slotId'] = this.slotId;
    if (this.serviceId != null) {
      json[r'serviceId'] = this.serviceId;
    } else {
      json[r'serviceId'] = null;
    }
    return json;
  }

  /// Returns a new [StaffBookingRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static StaffBookingRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'patientId'),
            'Required key "StaffBookingRequest[patientId]" is missing from JSON.');
        assert(json[r'patientId'] != null,
            'Required key "StaffBookingRequest[patientId]" has a null value in JSON.');
        assert(json.containsKey(r'slotId'),
            'Required key "StaffBookingRequest[slotId]" is missing from JSON.');
        assert(json[r'slotId'] != null,
            'Required key "StaffBookingRequest[slotId]" has a null value in JSON.');
        return true;
      }());

      return StaffBookingRequest(
        patientId: mapValueOfType<String>(json, r'patientId')!,
        slotId: mapValueOfType<String>(json, r'slotId')!,
        serviceId: mapValueOfType<String>(json, r'serviceId'),
      );
    }
    return null;
  }

  static List<StaffBookingRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <StaffBookingRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = StaffBookingRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, StaffBookingRequest> mapFromJson(dynamic json) {
    final map = <String, StaffBookingRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = StaffBookingRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of StaffBookingRequest-objects as value to a dart map
  static Map<String, List<StaffBookingRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<StaffBookingRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = StaffBookingRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'patientId',
    'slotId',
  };
}
