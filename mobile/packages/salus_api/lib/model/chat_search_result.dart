//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ChatSearchResult {
  /// Returns a new [ChatSearchResult] instance.
  ChatSearchResult({
    required this.conversationId,
    required this.message,
  });

  String conversationId;

  ChatMessage message;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ChatSearchResult &&
          other.conversationId == conversationId &&
          other.message == message;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (conversationId.hashCode) + (message.hashCode);

  @override
  String toString() =>
      'ChatSearchResult[conversationId=$conversationId, message=$message]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'conversation_id'] = this.conversationId;
    json[r'message'] = this.message;
    return json;
  }

  /// Returns a new [ChatSearchResult] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ChatSearchResult? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'conversation_id'),
            'Required key "ChatSearchResult[conversation_id]" is missing from JSON.');
        assert(json[r'conversation_id'] != null,
            'Required key "ChatSearchResult[conversation_id]" has a null value in JSON.');
        assert(json.containsKey(r'message'),
            'Required key "ChatSearchResult[message]" is missing from JSON.');
        assert(json[r'message'] != null,
            'Required key "ChatSearchResult[message]" has a null value in JSON.');
        return true;
      }());

      return ChatSearchResult(
        conversationId: mapValueOfType<String>(json, r'conversation_id')!,
        message: ChatMessage.fromJson(json[r'message'])!,
      );
    }
    return null;
  }

  static List<ChatSearchResult> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatSearchResult>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatSearchResult.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ChatSearchResult> mapFromJson(dynamic json) {
    final map = <String, ChatSearchResult>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ChatSearchResult.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ChatSearchResult-objects as value to a dart map
  static Map<String, List<ChatSearchResult>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ChatSearchResult>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ChatSearchResult.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'conversation_id',
    'message',
  };
}
