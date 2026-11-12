//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class OpenConversationRequest {
  /// Returns a new [OpenConversationRequest] instance.
  OpenConversationRequest({
    this.patientId,
    this.organizationId,
    this.practitionerRoleId,
    this.encounterId,
  });

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? patientId;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? organizationId;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? practitionerRoleId;

  /// The consultation the conversation is about.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? encounterId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is OpenConversationRequest &&
          other.patientId == patientId &&
          other.organizationId == organizationId &&
          other.practitionerRoleId == practitionerRoleId &&
          other.encounterId == encounterId;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (patientId == null ? 0 : patientId!.hashCode) +
      (organizationId == null ? 0 : organizationId!.hashCode) +
      (practitionerRoleId == null ? 0 : practitionerRoleId!.hashCode) +
      (encounterId == null ? 0 : encounterId!.hashCode);

  @override
  String toString() =>
      'OpenConversationRequest[patientId=$patientId, organizationId=$organizationId, practitionerRoleId=$practitionerRoleId, encounterId=$encounterId]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    if (this.patientId != null) {
      json[r'patient_id'] = this.patientId;
    } else {
      json[r'patient_id'] = null;
    }
    if (this.organizationId != null) {
      json[r'organization_id'] = this.organizationId;
    } else {
      json[r'organization_id'] = null;
    }
    if (this.practitionerRoleId != null) {
      json[r'practitioner_role_id'] = this.practitionerRoleId;
    } else {
      json[r'practitioner_role_id'] = null;
    }
    if (this.encounterId != null) {
      json[r'encounter_id'] = this.encounterId;
    } else {
      json[r'encounter_id'] = null;
    }
    return json;
  }

  /// Returns a new [OpenConversationRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static OpenConversationRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        return true;
      }());

      return OpenConversationRequest(
        patientId: mapValueOfType<String>(json, r'patient_id'),
        organizationId: mapValueOfType<String>(json, r'organization_id'),
        practitionerRoleId:
            mapValueOfType<String>(json, r'practitioner_role_id'),
        encounterId: mapValueOfType<String>(json, r'encounter_id'),
      );
    }
    return null;
  }

  static List<OpenConversationRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <OpenConversationRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = OpenConversationRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, OpenConversationRequest> mapFromJson(dynamic json) {
    final map = <String, OpenConversationRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = OpenConversationRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of OpenConversationRequest-objects as value to a dart map
  static Map<String, List<OpenConversationRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<OpenConversationRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = OpenConversationRequest.listFromJson(
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
