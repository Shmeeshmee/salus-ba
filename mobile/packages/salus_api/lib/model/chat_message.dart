//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ChatMessage {
  /// Returns a new [ChatMessage] instance.
  ChatMessage({
    required this.id,
    required this.conversationId,
    required this.seq,
    required this.cursor,
    this.senderParticipantId,
    required this.senderRole,
    required this.kind,
    this.body,
    this.system,
    required this.clientMessageId,
    required this.sentAt,
    this.editedAt,
    required this.versionId,
  });

  String id;

  String conversationId;

  /// The message's number in the conversation
  int seq;

  /// Opaque: the `before` of a page, the `after` of the socket.
  String cursor;

  /// Absent for a system message.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? senderParticipantId;

  ChatMessageSenderRoleEnum senderRole;

  ChatMessageKindEnum kind;

  /// A text's body, 1 to 4000 characters.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? body;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  SystemMessage? system;

  String clientMessageId;

  DateTime sentAt;

  /// Absent for a message never edited.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? editedAt;

  int versionId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ChatMessage &&
          other.id == id &&
          other.conversationId == conversationId &&
          other.seq == seq &&
          other.cursor == cursor &&
          other.senderParticipantId == senderParticipantId &&
          other.senderRole == senderRole &&
          other.kind == kind &&
          other.body == body &&
          other.system == system &&
          other.clientMessageId == clientMessageId &&
          other.sentAt == sentAt &&
          other.editedAt == editedAt &&
          other.versionId == versionId;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (conversationId.hashCode) +
      (seq.hashCode) +
      (cursor.hashCode) +
      (senderParticipantId == null ? 0 : senderParticipantId!.hashCode) +
      (senderRole.hashCode) +
      (kind.hashCode) +
      (body == null ? 0 : body!.hashCode) +
      (system == null ? 0 : system!.hashCode) +
      (clientMessageId.hashCode) +
      (sentAt.hashCode) +
      (editedAt == null ? 0 : editedAt!.hashCode) +
      (versionId.hashCode);

  @override
  String toString() =>
      'ChatMessage[id=$id, conversationId=$conversationId, seq=$seq, cursor=$cursor, senderParticipantId=$senderParticipantId, senderRole=$senderRole, kind=$kind, body=$body, system=$system, clientMessageId=$clientMessageId, sentAt=$sentAt, editedAt=$editedAt, versionId=$versionId]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'conversation_id'] = this.conversationId;
    json[r'seq'] = this.seq;
    json[r'cursor'] = this.cursor;
    if (this.senderParticipantId != null) {
      json[r'sender_participant_id'] = this.senderParticipantId;
    } else {
      json[r'sender_participant_id'] = null;
    }
    json[r'sender_role'] = this.senderRole;
    json[r'kind'] = this.kind;
    if (this.body != null) {
      json[r'body'] = this.body;
    } else {
      json[r'body'] = null;
    }
    if (this.system != null) {
      json[r'system'] = this.system;
    } else {
      json[r'system'] = null;
    }
    json[r'client_message_id'] = this.clientMessageId;
    json[r'sent_at'] = this.sentAt.toUtc().toIso8601String();
    if (this.editedAt != null) {
      json[r'edited_at'] = this.editedAt!.toUtc().toIso8601String();
    } else {
      json[r'edited_at'] = null;
    }
    json[r'version_id'] = this.versionId;
    return json;
  }

  /// Returns a new [ChatMessage] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ChatMessage? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "ChatMessage[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "ChatMessage[id]" has a null value in JSON.');
        assert(json.containsKey(r'conversation_id'),
            'Required key "ChatMessage[conversation_id]" is missing from JSON.');
        assert(json[r'conversation_id'] != null,
            'Required key "ChatMessage[conversation_id]" has a null value in JSON.');
        assert(json.containsKey(r'seq'),
            'Required key "ChatMessage[seq]" is missing from JSON.');
        assert(json[r'seq'] != null,
            'Required key "ChatMessage[seq]" has a null value in JSON.');
        assert(json.containsKey(r'cursor'),
            'Required key "ChatMessage[cursor]" is missing from JSON.');
        assert(json[r'cursor'] != null,
            'Required key "ChatMessage[cursor]" has a null value in JSON.');
        assert(json.containsKey(r'sender_role'),
            'Required key "ChatMessage[sender_role]" is missing from JSON.');
        assert(json[r'sender_role'] != null,
            'Required key "ChatMessage[sender_role]" has a null value in JSON.');
        assert(json.containsKey(r'kind'),
            'Required key "ChatMessage[kind]" is missing from JSON.');
        assert(json[r'kind'] != null,
            'Required key "ChatMessage[kind]" has a null value in JSON.');
        assert(json.containsKey(r'client_message_id'),
            'Required key "ChatMessage[client_message_id]" is missing from JSON.');
        assert(json[r'client_message_id'] != null,
            'Required key "ChatMessage[client_message_id]" has a null value in JSON.');
        assert(json.containsKey(r'sent_at'),
            'Required key "ChatMessage[sent_at]" is missing from JSON.');
        assert(json[r'sent_at'] != null,
            'Required key "ChatMessage[sent_at]" has a null value in JSON.');
        assert(json.containsKey(r'version_id'),
            'Required key "ChatMessage[version_id]" is missing from JSON.');
        assert(json[r'version_id'] != null,
            'Required key "ChatMessage[version_id]" has a null value in JSON.');
        return true;
      }());

      return ChatMessage(
        id: mapValueOfType<String>(json, r'id')!,
        conversationId: mapValueOfType<String>(json, r'conversation_id')!,
        seq: mapValueOfType<int>(json, r'seq')!,
        cursor: mapValueOfType<String>(json, r'cursor')!,
        senderParticipantId:
            mapValueOfType<String>(json, r'sender_participant_id'),
        senderRole: ChatMessageSenderRoleEnum.fromJson(json[r'sender_role'])!,
        kind: ChatMessageKindEnum.fromJson(json[r'kind'])!,
        body: mapValueOfType<String>(json, r'body'),
        system: SystemMessage.fromJson(json[r'system']),
        clientMessageId: mapValueOfType<String>(json, r'client_message_id')!,
        sentAt: mapDateTime(json, r'sent_at', r'')!,
        editedAt: mapDateTime(json, r'edited_at', r''),
        versionId: mapValueOfType<int>(json, r'version_id')!,
      );
    }
    return null;
  }

  static List<ChatMessage> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatMessage>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatMessage.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ChatMessage> mapFromJson(dynamic json) {
    final map = <String, ChatMessage>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ChatMessage.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ChatMessage-objects as value to a dart map
  static Map<String, List<ChatMessage>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ChatMessage>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ChatMessage.listFromJson(
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
    'seq',
    'cursor',
    'sender_role',
    'kind',
    'client_message_id',
    'sent_at',
    'version_id',
  };
}

enum ChatMessageSenderRoleEnum {
  patient._(r'patient'),
  practitioner._(r'practitioner'),
  system._(r'system'),
  ;

  /// Instantiate a new enum with the provided value.
  const ChatMessageSenderRoleEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ChatMessageSenderRoleEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ChatMessageSenderRoleEnum? fromJson(dynamic value) =>
      ChatMessageSenderRoleEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ChatMessageSenderRoleEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ChatMessageSenderRoleEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatMessageSenderRoleEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatMessageSenderRoleEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ChatMessageSenderRoleEnum] to String,
/// and [decode] dynamic data back to [ChatMessageSenderRoleEnum].
class ChatMessageSenderRoleEnumTypeTransformer {
  factory ChatMessageSenderRoleEnumTypeTransformer() =>
      _instance ??= const ChatMessageSenderRoleEnumTypeTransformer._();

  const ChatMessageSenderRoleEnumTypeTransformer._();

  String encode(ChatMessageSenderRoleEnum data) => data._value;

  /// Returns the instance of [ChatMessageSenderRoleEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ChatMessageSenderRoleEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ChatMessageSenderRoleEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'patient':
          return ChatMessageSenderRoleEnum.patient;
        case r'practitioner':
          return ChatMessageSenderRoleEnum.practitioner;
        case r'system':
          return ChatMessageSenderRoleEnum.system;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ChatMessageSenderRoleEnumTypeTransformer? _instance;
}

enum ChatMessageKindEnum {
  text._(r'text'),
  system._(r'system'),
  ;

  /// Instantiate a new enum with the provided value.
  const ChatMessageKindEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ChatMessageKindEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ChatMessageKindEnum? fromJson(dynamic value) =>
      ChatMessageKindEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ChatMessageKindEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ChatMessageKindEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatMessageKindEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatMessageKindEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ChatMessageKindEnum] to String,
/// and [decode] dynamic data back to [ChatMessageKindEnum].
class ChatMessageKindEnumTypeTransformer {
  factory ChatMessageKindEnumTypeTransformer() =>
      _instance ??= const ChatMessageKindEnumTypeTransformer._();

  const ChatMessageKindEnumTypeTransformer._();

  String encode(ChatMessageKindEnum data) => data._value;

  /// Returns the instance of [ChatMessageKindEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ChatMessageKindEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ChatMessageKindEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'text':
          return ChatMessageKindEnum.text;
        case r'system':
          return ChatMessageKindEnum.system;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ChatMessageKindEnumTypeTransformer? _instance;
}
