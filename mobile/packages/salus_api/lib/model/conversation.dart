//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Conversation {
  /// Returns a new [Conversation] instance.
  Conversation({
    required this.id,
    required this.organizationId,
    required this.patientId,
    required this.practitionerRoleId,
    this.encounterId,
    this.parentId,
    required this.createdAt,
    this.lastSentAt,
    required this.lastSeq,
    required this.unread,
    required this.staffOnly,
    this.participants = const [],
  });

  String id;

  String organizationId;

  String patientId;

  /// The treating practitioner.
  String practitionerRoleId;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? encounterId;

  /// A doctor-only side thread's conversation (SAL-28).
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? parentId;

  DateTime createdAt;

  /// Absent before the first message.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? lastSentAt;

  /// The number of the latest message; 0 before the first.
  int lastSeq;

  /// The caller's messages after their read position.
  int unread;

  /// A doctor-only side thread (SAL-28); never shown to the patient.
  bool staffOnly;

  /// The active participants, the patient first; absent in a list.
  List<ConversationParticipant> participants;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Conversation &&
          other.id == id &&
          other.organizationId == organizationId &&
          other.patientId == patientId &&
          other.practitionerRoleId == practitionerRoleId &&
          other.encounterId == encounterId &&
          other.parentId == parentId &&
          other.createdAt == createdAt &&
          other.lastSentAt == lastSentAt &&
          other.lastSeq == lastSeq &&
          other.unread == unread &&
          other.staffOnly == staffOnly &&
          _deepEquality.equals(other.participants, participants);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (organizationId.hashCode) +
      (patientId.hashCode) +
      (practitionerRoleId.hashCode) +
      (encounterId == null ? 0 : encounterId!.hashCode) +
      (parentId == null ? 0 : parentId!.hashCode) +
      (createdAt.hashCode) +
      (lastSentAt == null ? 0 : lastSentAt!.hashCode) +
      (lastSeq.hashCode) +
      (unread.hashCode) +
      (staffOnly.hashCode) +
      (participants.hashCode);

  @override
  String toString() =>
      'Conversation[id=$id, organizationId=$organizationId, patientId=$patientId, practitionerRoleId=$practitionerRoleId, encounterId=$encounterId, parentId=$parentId, createdAt=$createdAt, lastSentAt=$lastSentAt, lastSeq=$lastSeq, unread=$unread, staffOnly=$staffOnly, participants=$participants]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'organization_id'] = this.organizationId;
    json[r'patient_id'] = this.patientId;
    json[r'practitioner_role_id'] = this.practitionerRoleId;
    if (this.encounterId != null) {
      json[r'encounter_id'] = this.encounterId;
    } else {
      json[r'encounter_id'] = null;
    }
    if (this.parentId != null) {
      json[r'parent_id'] = this.parentId;
    } else {
      json[r'parent_id'] = null;
    }
    json[r'created_at'] = this.createdAt.toUtc().toIso8601String();
    if (this.lastSentAt != null) {
      json[r'last_sent_at'] = this.lastSentAt!.toUtc().toIso8601String();
    } else {
      json[r'last_sent_at'] = null;
    }
    json[r'last_seq'] = this.lastSeq;
    json[r'unread'] = this.unread;
    json[r'staff_only'] = this.staffOnly;
    json[r'participants'] = this.participants;
    return json;
  }

  /// Returns a new [Conversation] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Conversation? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "Conversation[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "Conversation[id]" has a null value in JSON.');
        assert(json.containsKey(r'organization_id'),
            'Required key "Conversation[organization_id]" is missing from JSON.');
        assert(json[r'organization_id'] != null,
            'Required key "Conversation[organization_id]" has a null value in JSON.');
        assert(json.containsKey(r'patient_id'),
            'Required key "Conversation[patient_id]" is missing from JSON.');
        assert(json[r'patient_id'] != null,
            'Required key "Conversation[patient_id]" has a null value in JSON.');
        assert(json.containsKey(r'practitioner_role_id'),
            'Required key "Conversation[practitioner_role_id]" is missing from JSON.');
        assert(json[r'practitioner_role_id'] != null,
            'Required key "Conversation[practitioner_role_id]" has a null value in JSON.');
        assert(json.containsKey(r'created_at'),
            'Required key "Conversation[created_at]" is missing from JSON.');
        assert(json[r'created_at'] != null,
            'Required key "Conversation[created_at]" has a null value in JSON.');
        assert(json.containsKey(r'last_seq'),
            'Required key "Conversation[last_seq]" is missing from JSON.');
        assert(json[r'last_seq'] != null,
            'Required key "Conversation[last_seq]" has a null value in JSON.');
        assert(json.containsKey(r'unread'),
            'Required key "Conversation[unread]" is missing from JSON.');
        assert(json[r'unread'] != null,
            'Required key "Conversation[unread]" has a null value in JSON.');
        assert(json.containsKey(r'staff_only'),
            'Required key "Conversation[staff_only]" is missing from JSON.');
        assert(json[r'staff_only'] != null,
            'Required key "Conversation[staff_only]" has a null value in JSON.');
        return true;
      }());

      return Conversation(
        id: mapValueOfType<String>(json, r'id')!,
        organizationId: mapValueOfType<String>(json, r'organization_id')!,
        patientId: mapValueOfType<String>(json, r'patient_id')!,
        practitionerRoleId:
            mapValueOfType<String>(json, r'practitioner_role_id')!,
        encounterId: mapValueOfType<String>(json, r'encounter_id'),
        parentId: mapValueOfType<String>(json, r'parent_id'),
        createdAt: mapDateTime(json, r'created_at', r'')!,
        lastSentAt: mapDateTime(json, r'last_sent_at', r''),
        lastSeq: mapValueOfType<int>(json, r'last_seq')!,
        unread: mapValueOfType<int>(json, r'unread')!,
        staffOnly: mapValueOfType<bool>(json, r'staff_only')!,
        participants:
            ConversationParticipant.listFromJson(json[r'participants']),
      );
    }
    return null;
  }

  static List<Conversation> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Conversation>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Conversation.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Conversation> mapFromJson(dynamic json) {
    final map = <String, Conversation>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Conversation.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Conversation-objects as value to a dart map
  static Map<String, List<Conversation>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Conversation>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Conversation.listFromJson(
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
    'organization_id',
    'patient_id',
    'practitioner_role_id',
    'created_at',
    'last_seq',
    'unread',
    'staff_only',
  };
}
