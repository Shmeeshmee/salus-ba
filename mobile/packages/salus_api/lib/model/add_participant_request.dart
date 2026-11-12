//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class AddParticipantRequest {
  /// Returns a new [AddParticipantRequest] instance.
  AddParticipantRequest({
    required this.practitionerRoleId,
    required this.organizationId,
    required this.reason,
    required this.history,
    this.fromSeq,
  });

  String practitionerRoleId;

  /// The role's clinic: the conversation's to add, another to invite.
  String organizationId;

  AddParticipantRequestReasonEnum reason;

  /// `full`: the whole history; `from`: the messages after `from_seq`.
  AddParticipantRequestHistoryEnum history;

  /// With `history` `from` only; at most the conversation's `last_seq`.
  ///
  /// Minimum value: 0
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  int? fromSeq;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is AddParticipantRequest &&
          other.practitionerRoleId == practitionerRoleId &&
          other.organizationId == organizationId &&
          other.reason == reason &&
          other.history == history &&
          other.fromSeq == fromSeq;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (practitionerRoleId.hashCode) +
      (organizationId.hashCode) +
      (reason.hashCode) +
      (history.hashCode) +
      (fromSeq == null ? 0 : fromSeq!.hashCode);

  @override
  String toString() =>
      'AddParticipantRequest[practitionerRoleId=$practitionerRoleId, organizationId=$organizationId, reason=$reason, history=$history, fromSeq=$fromSeq]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'practitioner_role_id'] = this.practitionerRoleId;
    json[r'organization_id'] = this.organizationId;
    json[r'reason'] = this.reason;
    json[r'history'] = this.history;
    if (this.fromSeq != null) {
      json[r'from_seq'] = this.fromSeq;
    } else {
      json[r'from_seq'] = null;
    }
    return json;
  }

  /// Returns a new [AddParticipantRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static AddParticipantRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'practitioner_role_id'),
            'Required key "AddParticipantRequest[practitioner_role_id]" is missing from JSON.');
        assert(json[r'practitioner_role_id'] != null,
            'Required key "AddParticipantRequest[practitioner_role_id]" has a null value in JSON.');
        assert(json.containsKey(r'organization_id'),
            'Required key "AddParticipantRequest[organization_id]" is missing from JSON.');
        assert(json[r'organization_id'] != null,
            'Required key "AddParticipantRequest[organization_id]" has a null value in JSON.');
        assert(json.containsKey(r'reason'),
            'Required key "AddParticipantRequest[reason]" is missing from JSON.');
        assert(json[r'reason'] != null,
            'Required key "AddParticipantRequest[reason]" has a null value in JSON.');
        assert(json.containsKey(r'history'),
            'Required key "AddParticipantRequest[history]" is missing from JSON.');
        assert(json[r'history'] != null,
            'Required key "AddParticipantRequest[history]" has a null value in JSON.');
        return true;
      }());

      return AddParticipantRequest(
        practitionerRoleId:
            mapValueOfType<String>(json, r'practitioner_role_id')!,
        organizationId: mapValueOfType<String>(json, r'organization_id')!,
        reason: AddParticipantRequestReasonEnum.fromJson(json[r'reason'])!,
        history: AddParticipantRequestHistoryEnum.fromJson(json[r'history'])!,
        fromSeq: mapValueOfType<int>(json, r'from_seq'),
      );
    }
    return null;
  }

  static List<AddParticipantRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <AddParticipantRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = AddParticipantRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, AddParticipantRequest> mapFromJson(dynamic json) {
    final map = <String, AddParticipantRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = AddParticipantRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of AddParticipantRequest-objects as value to a dart map
  static Map<String, List<AddParticipantRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<AddParticipantRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = AddParticipantRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'practitioner_role_id',
    'organization_id',
    'reason',
    'history',
  };
}

enum AddParticipantRequestReasonEnum {
  secondOpinion._(r'second_opinion'),
  specialistReview._(r'specialist_review'),
  handover._(r'handover'),
  ;

  /// Instantiate a new enum with the provided value.
  const AddParticipantRequestReasonEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [AddParticipantRequestReasonEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static AddParticipantRequestReasonEnum? fromJson(dynamic value) =>
      AddParticipantRequestReasonEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [AddParticipantRequestReasonEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<AddParticipantRequestReasonEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <AddParticipantRequestReasonEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = AddParticipantRequestReasonEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [AddParticipantRequestReasonEnum] to String,
/// and [decode] dynamic data back to [AddParticipantRequestReasonEnum].
class AddParticipantRequestReasonEnumTypeTransformer {
  factory AddParticipantRequestReasonEnumTypeTransformer() =>
      _instance ??= const AddParticipantRequestReasonEnumTypeTransformer._();

  const AddParticipantRequestReasonEnumTypeTransformer._();

  String encode(AddParticipantRequestReasonEnum data) => data._value;

  /// Returns the instance of [AddParticipantRequestReasonEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  AddParticipantRequestReasonEnum? decode(dynamic data,
      {bool allowNull = true}) {
    if (data is AddParticipantRequestReasonEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'second_opinion':
          return AddParticipantRequestReasonEnum.secondOpinion;
        case r'specialist_review':
          return AddParticipantRequestReasonEnum.specialistReview;
        case r'handover':
          return AddParticipantRequestReasonEnum.handover;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static AddParticipantRequestReasonEnumTypeTransformer? _instance;
}

/// `full`: the whole history; `from`: the messages after `from_seq`.
enum AddParticipantRequestHistoryEnum {
  full._(r'full'),
  from._(r'from'),
  ;

  /// Instantiate a new enum with the provided value.
  const AddParticipantRequestHistoryEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [AddParticipantRequestHistoryEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static AddParticipantRequestHistoryEnum? fromJson(dynamic value) =>
      AddParticipantRequestHistoryEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [AddParticipantRequestHistoryEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<AddParticipantRequestHistoryEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <AddParticipantRequestHistoryEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = AddParticipantRequestHistoryEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [AddParticipantRequestHistoryEnum] to String,
/// and [decode] dynamic data back to [AddParticipantRequestHistoryEnum].
class AddParticipantRequestHistoryEnumTypeTransformer {
  factory AddParticipantRequestHistoryEnumTypeTransformer() =>
      _instance ??= const AddParticipantRequestHistoryEnumTypeTransformer._();

  const AddParticipantRequestHistoryEnumTypeTransformer._();

  String encode(AddParticipantRequestHistoryEnum data) => data._value;

  /// Returns the instance of [AddParticipantRequestHistoryEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  AddParticipantRequestHistoryEnum? decode(dynamic data,
      {bool allowNull = true}) {
    if (data is AddParticipantRequestHistoryEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'full':
          return AddParticipantRequestHistoryEnum.full;
        case r'from':
          return AddParticipantRequestHistoryEnum.from;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static AddParticipantRequestHistoryEnumTypeTransformer? _instance;
}
