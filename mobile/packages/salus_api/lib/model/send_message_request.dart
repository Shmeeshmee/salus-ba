//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SendMessageRequest {
  /// Returns a new [SendMessageRequest] instance.
  SendMessageRequest({
    required this.clientMessageId,
    required this.body,
    this.attachmentIds = const [],
  });

  /// The client's id of the message; a retry repeats it.
  String clientMessageId;

  /// 1 to 4000 characters after trimming; may be empty with `attachment_ids`.
  String body;

  /// The sender's uploads to the conversation (`uploadChatAttachment`), scanning or clean, not yet sent; at most 10.
  List<String> attachmentIds;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SendMessageRequest &&
          other.clientMessageId == clientMessageId &&
          other.body == body &&
          _deepEquality.equals(other.attachmentIds, attachmentIds);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (clientMessageId.hashCode) + (body.hashCode) + (attachmentIds.hashCode);

  @override
  String toString() =>
      'SendMessageRequest[clientMessageId=$clientMessageId, body=$body, attachmentIds=$attachmentIds]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'client_message_id'] = this.clientMessageId;
    json[r'body'] = this.body;
    json[r'attachment_ids'] = this.attachmentIds;
    return json;
  }

  /// Returns a new [SendMessageRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SendMessageRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'client_message_id'),
            'Required key "SendMessageRequest[client_message_id]" is missing from JSON.');
        assert(json[r'client_message_id'] != null,
            'Required key "SendMessageRequest[client_message_id]" has a null value in JSON.');
        assert(json.containsKey(r'body'),
            'Required key "SendMessageRequest[body]" is missing from JSON.');
        assert(json[r'body'] != null,
            'Required key "SendMessageRequest[body]" has a null value in JSON.');
        return true;
      }());

      return SendMessageRequest(
        clientMessageId: mapValueOfType<String>(json, r'client_message_id')!,
        body: mapValueOfType<String>(json, r'body')!,
        attachmentIds: json[r'attachment_ids'] is Iterable
            ? (json[r'attachment_ids'] as Iterable)
                .cast<String>()
                .toList(growable: false)
            : const [],
      );
    }
    return null;
  }

  static List<SendMessageRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SendMessageRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SendMessageRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SendMessageRequest> mapFromJson(dynamic json) {
    final map = <String, SendMessageRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SendMessageRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SendMessageRequest-objects as value to a dart map
  static Map<String, List<SendMessageRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SendMessageRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SendMessageRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'client_message_id',
    'body',
  };
}
