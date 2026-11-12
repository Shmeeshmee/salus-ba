//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ChatInvitation {
  /// Returns a new [ChatInvitation] instance.
  ChatInvitation({
    required this.id,
    required this.conversationId,
    required this.organizationId,
    required this.practitionerRoleId,
    required this.reason,
    required this.invitedAt,
  });

  /// The invitation's id (the invited participant's).
  String id;

  String conversationId;

  /// The invited doctor's clinic.
  String organizationId;

  String practitionerRoleId;

  ChatInvitationReasonEnum reason;

  DateTime invitedAt;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ChatInvitation &&
          other.id == id &&
          other.conversationId == conversationId &&
          other.organizationId == organizationId &&
          other.practitionerRoleId == practitionerRoleId &&
          other.reason == reason &&
          other.invitedAt == invitedAt;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (conversationId.hashCode) +
      (organizationId.hashCode) +
      (practitionerRoleId.hashCode) +
      (reason.hashCode) +
      (invitedAt.hashCode);

  @override
  String toString() =>
      'ChatInvitation[id=$id, conversationId=$conversationId, organizationId=$organizationId, practitionerRoleId=$practitionerRoleId, reason=$reason, invitedAt=$invitedAt]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'conversation_id'] = this.conversationId;
    json[r'organization_id'] = this.organizationId;
    json[r'practitioner_role_id'] = this.practitionerRoleId;
    json[r'reason'] = this.reason;
    json[r'invited_at'] = this.invitedAt.toUtc().toIso8601String();
    return json;
  }

  /// Returns a new [ChatInvitation] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ChatInvitation? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "ChatInvitation[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "ChatInvitation[id]" has a null value in JSON.');
        assert(json.containsKey(r'conversation_id'),
            'Required key "ChatInvitation[conversation_id]" is missing from JSON.');
        assert(json[r'conversation_id'] != null,
            'Required key "ChatInvitation[conversation_id]" has a null value in JSON.');
        assert(json.containsKey(r'organization_id'),
            'Required key "ChatInvitation[organization_id]" is missing from JSON.');
        assert(json[r'organization_id'] != null,
            'Required key "ChatInvitation[organization_id]" has a null value in JSON.');
        assert(json.containsKey(r'practitioner_role_id'),
            'Required key "ChatInvitation[practitioner_role_id]" is missing from JSON.');
        assert(json[r'practitioner_role_id'] != null,
            'Required key "ChatInvitation[practitioner_role_id]" has a null value in JSON.');
        assert(json.containsKey(r'reason'),
            'Required key "ChatInvitation[reason]" is missing from JSON.');
        assert(json[r'reason'] != null,
            'Required key "ChatInvitation[reason]" has a null value in JSON.');
        assert(json.containsKey(r'invited_at'),
            'Required key "ChatInvitation[invited_at]" is missing from JSON.');
        assert(json[r'invited_at'] != null,
            'Required key "ChatInvitation[invited_at]" has a null value in JSON.');
        return true;
      }());

      return ChatInvitation(
        id: mapValueOfType<String>(json, r'id')!,
        conversationId: mapValueOfType<String>(json, r'conversation_id')!,
        organizationId: mapValueOfType<String>(json, r'organization_id')!,
        practitionerRoleId:
            mapValueOfType<String>(json, r'practitioner_role_id')!,
        reason: ChatInvitationReasonEnum.fromJson(json[r'reason'])!,
        invitedAt: mapDateTime(json, r'invited_at', r'')!,
      );
    }
    return null;
  }

  static List<ChatInvitation> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatInvitation>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatInvitation.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ChatInvitation> mapFromJson(dynamic json) {
    final map = <String, ChatInvitation>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ChatInvitation.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ChatInvitation-objects as value to a dart map
  static Map<String, List<ChatInvitation>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ChatInvitation>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ChatInvitation.listFromJson(
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
    'conversation_id',
    'organization_id',
    'practitioner_role_id',
    'reason',
    'invited_at',
  };
}

enum ChatInvitationReasonEnum {
  secondOpinion._(r'second_opinion'),
  specialistReview._(r'specialist_review'),
  handover._(r'handover'),
  ;

  /// Instantiate a new enum with the provided value.
  const ChatInvitationReasonEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ChatInvitationReasonEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ChatInvitationReasonEnum? fromJson(dynamic value) =>
      ChatInvitationReasonEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ChatInvitationReasonEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ChatInvitationReasonEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatInvitationReasonEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatInvitationReasonEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ChatInvitationReasonEnum] to String,
/// and [decode] dynamic data back to [ChatInvitationReasonEnum].
class ChatInvitationReasonEnumTypeTransformer {
  factory ChatInvitationReasonEnumTypeTransformer() =>
      _instance ??= const ChatInvitationReasonEnumTypeTransformer._();

  const ChatInvitationReasonEnumTypeTransformer._();

  String encode(ChatInvitationReasonEnum data) => data._value;

  /// Returns the instance of [ChatInvitationReasonEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ChatInvitationReasonEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ChatInvitationReasonEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'second_opinion':
          return ChatInvitationReasonEnum.secondOpinion;
        case r'specialist_review':
          return ChatInvitationReasonEnum.specialistReview;
        case r'handover':
          return ChatInvitationReasonEnum.handover;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ChatInvitationReasonEnumTypeTransformer? _instance;
}
