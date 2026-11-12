//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ConversationParticipant {
  /// Returns a new [ConversationParticipant] instance.
  ConversationParticipant({
    required this.id,
    required this.role,
    required this.status,
    this.patientId,
    this.practitionerRoleId,
    this.organizationId,
    this.reason,
    required this.addedAt,
    required this.historyFromSeq,
  });

  String id;

  ConversationParticipantRoleEnum role;

  /// `pending_consent`: a doctor of another clinic waiting for the patient's consent, who sees nothing until then (SAL-28).
  ConversationParticipantStatusEnum status;

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
  String? practitionerRoleId;

  /// A practitioner's clinic: another than the conversation's for an invited doctor (SAL-28).
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? organizationId;

  /// Why the practitioner was added; absent for the patient and the treating practitioner.
  ConversationParticipantReasonEnum? reason;

  DateTime addedAt;

  /// The participant reads the messages after this number; 0 for all.
  int historyFromSeq;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ConversationParticipant &&
          other.id == id &&
          other.role == role &&
          other.status == status &&
          other.patientId == patientId &&
          other.practitionerRoleId == practitionerRoleId &&
          other.organizationId == organizationId &&
          other.reason == reason &&
          other.addedAt == addedAt &&
          other.historyFromSeq == historyFromSeq;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (role.hashCode) +
      (status.hashCode) +
      (patientId == null ? 0 : patientId!.hashCode) +
      (practitionerRoleId == null ? 0 : practitionerRoleId!.hashCode) +
      (organizationId == null ? 0 : organizationId!.hashCode) +
      (reason == null ? 0 : reason!.hashCode) +
      (addedAt.hashCode) +
      (historyFromSeq.hashCode);

  @override
  String toString() =>
      'ConversationParticipant[id=$id, role=$role, status=$status, patientId=$patientId, practitionerRoleId=$practitionerRoleId, organizationId=$organizationId, reason=$reason, addedAt=$addedAt, historyFromSeq=$historyFromSeq]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'role'] = this.role;
    json[r'status'] = this.status;
    if (this.patientId != null) {
      json[r'patient_id'] = this.patientId;
    } else {
      json[r'patient_id'] = null;
    }
    if (this.practitionerRoleId != null) {
      json[r'practitioner_role_id'] = this.practitionerRoleId;
    } else {
      json[r'practitioner_role_id'] = null;
    }
    if (this.organizationId != null) {
      json[r'organization_id'] = this.organizationId;
    } else {
      json[r'organization_id'] = null;
    }
    if (this.reason != null) {
      json[r'reason'] = this.reason;
    } else {
      json[r'reason'] = null;
    }
    json[r'added_at'] = this.addedAt.toUtc().toIso8601String();
    json[r'history_from_seq'] = this.historyFromSeq;
    return json;
  }

  /// Returns a new [ConversationParticipant] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ConversationParticipant? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "ConversationParticipant[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "ConversationParticipant[id]" has a null value in JSON.');
        assert(json.containsKey(r'role'),
            'Required key "ConversationParticipant[role]" is missing from JSON.');
        assert(json[r'role'] != null,
            'Required key "ConversationParticipant[role]" has a null value in JSON.');
        assert(json.containsKey(r'status'),
            'Required key "ConversationParticipant[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "ConversationParticipant[status]" has a null value in JSON.');
        assert(json.containsKey(r'added_at'),
            'Required key "ConversationParticipant[added_at]" is missing from JSON.');
        assert(json[r'added_at'] != null,
            'Required key "ConversationParticipant[added_at]" has a null value in JSON.');
        assert(json.containsKey(r'history_from_seq'),
            'Required key "ConversationParticipant[history_from_seq]" is missing from JSON.');
        assert(json[r'history_from_seq'] != null,
            'Required key "ConversationParticipant[history_from_seq]" has a null value in JSON.');
        return true;
      }());

      return ConversationParticipant(
        id: mapValueOfType<String>(json, r'id')!,
        role: ConversationParticipantRoleEnum.fromJson(json[r'role'])!,
        status: ConversationParticipantStatusEnum.fromJson(json[r'status'])!,
        patientId: mapValueOfType<String>(json, r'patient_id'),
        practitionerRoleId:
            mapValueOfType<String>(json, r'practitioner_role_id'),
        organizationId: mapValueOfType<String>(json, r'organization_id'),
        reason: ConversationParticipantReasonEnum.fromJson(json[r'reason']),
        addedAt: mapDateTime(json, r'added_at', r'')!,
        historyFromSeq: mapValueOfType<int>(json, r'history_from_seq')!,
      );
    }
    return null;
  }

  static List<ConversationParticipant> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ConversationParticipant>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ConversationParticipant.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ConversationParticipant> mapFromJson(dynamic json) {
    final map = <String, ConversationParticipant>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ConversationParticipant.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ConversationParticipant-objects as value to a dart map
  static Map<String, List<ConversationParticipant>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ConversationParticipant>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ConversationParticipant.listFromJson(
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
    'role',
    'status',
    'added_at',
    'history_from_seq',
  };
}

enum ConversationParticipantRoleEnum {
  patient._(r'patient'),
  practitioner._(r'practitioner'),
  ;

  /// Instantiate a new enum with the provided value.
  const ConversationParticipantRoleEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ConversationParticipantRoleEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ConversationParticipantRoleEnum? fromJson(dynamic value) =>
      ConversationParticipantRoleEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ConversationParticipantRoleEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ConversationParticipantRoleEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ConversationParticipantRoleEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ConversationParticipantRoleEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ConversationParticipantRoleEnum] to String,
/// and [decode] dynamic data back to [ConversationParticipantRoleEnum].
class ConversationParticipantRoleEnumTypeTransformer {
  factory ConversationParticipantRoleEnumTypeTransformer() =>
      _instance ??= const ConversationParticipantRoleEnumTypeTransformer._();

  const ConversationParticipantRoleEnumTypeTransformer._();

  String encode(ConversationParticipantRoleEnum data) => data._value;

  /// Returns the instance of [ConversationParticipantRoleEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ConversationParticipantRoleEnum? decode(dynamic data,
      {bool allowNull = true}) {
    if (data is ConversationParticipantRoleEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'patient':
          return ConversationParticipantRoleEnum.patient;
        case r'practitioner':
          return ConversationParticipantRoleEnum.practitioner;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ConversationParticipantRoleEnumTypeTransformer? _instance;
}

/// `pending_consent`: a doctor of another clinic waiting for the patient's consent, who sees nothing until then (SAL-28).
enum ConversationParticipantStatusEnum {
  active._(r'active'),
  pendingConsent._(r'pending_consent'),
  ;

  /// Instantiate a new enum with the provided value.
  const ConversationParticipantStatusEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ConversationParticipantStatusEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ConversationParticipantStatusEnum? fromJson(dynamic value) =>
      ConversationParticipantStatusEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ConversationParticipantStatusEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ConversationParticipantStatusEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ConversationParticipantStatusEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ConversationParticipantStatusEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ConversationParticipantStatusEnum] to String,
/// and [decode] dynamic data back to [ConversationParticipantStatusEnum].
class ConversationParticipantStatusEnumTypeTransformer {
  factory ConversationParticipantStatusEnumTypeTransformer() =>
      _instance ??= const ConversationParticipantStatusEnumTypeTransformer._();

  const ConversationParticipantStatusEnumTypeTransformer._();

  String encode(ConversationParticipantStatusEnum data) => data._value;

  /// Returns the instance of [ConversationParticipantStatusEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ConversationParticipantStatusEnum? decode(dynamic data,
      {bool allowNull = true}) {
    if (data is ConversationParticipantStatusEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'active':
          return ConversationParticipantStatusEnum.active;
        case r'pending_consent':
          return ConversationParticipantStatusEnum.pendingConsent;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ConversationParticipantStatusEnumTypeTransformer? _instance;
}

/// Why the practitioner was added; absent for the patient and the treating practitioner.
enum ConversationParticipantReasonEnum {
  secondOpinion._(r'second_opinion'),
  specialistReview._(r'specialist_review'),
  handover._(r'handover'),
  ;

  /// Instantiate a new enum with the provided value.
  const ConversationParticipantReasonEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ConversationParticipantReasonEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ConversationParticipantReasonEnum? fromJson(dynamic value) =>
      ConversationParticipantReasonEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ConversationParticipantReasonEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ConversationParticipantReasonEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ConversationParticipantReasonEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ConversationParticipantReasonEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ConversationParticipantReasonEnum] to String,
/// and [decode] dynamic data back to [ConversationParticipantReasonEnum].
class ConversationParticipantReasonEnumTypeTransformer {
  factory ConversationParticipantReasonEnumTypeTransformer() =>
      _instance ??= const ConversationParticipantReasonEnumTypeTransformer._();

  const ConversationParticipantReasonEnumTypeTransformer._();

  String encode(ConversationParticipantReasonEnum data) => data._value;

  /// Returns the instance of [ConversationParticipantReasonEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ConversationParticipantReasonEnum? decode(dynamic data,
      {bool allowNull = true}) {
    if (data is ConversationParticipantReasonEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'second_opinion':
          return ConversationParticipantReasonEnum.secondOpinion;
        case r'specialist_review':
          return ConversationParticipantReasonEnum.specialistReview;
        case r'handover':
          return ConversationParticipantReasonEnum.handover;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ConversationParticipantReasonEnumTypeTransformer? _instance;
}
