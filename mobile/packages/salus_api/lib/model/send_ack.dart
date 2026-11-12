//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SendAck {
  /// Returns a new [SendAck] instance.
  SendAck({
    required this.clientMessageId,
    required this.message,
  });

  String clientMessageId;

  ChatMessage message;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SendAck &&
          other.clientMessageId == clientMessageId &&
          other.message == message;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (clientMessageId.hashCode) + (message.hashCode);

  @override
  String toString() =>
      'SendAck[clientMessageId=$clientMessageId, message=$message]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'client_message_id'] = this.clientMessageId;
    json[r'message'] = this.message;
    return json;
  }

  /// Returns a new [SendAck] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SendAck? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'client_message_id'),
            'Required key "SendAck[client_message_id]" is missing from JSON.');
        assert(json[r'client_message_id'] != null,
            'Required key "SendAck[client_message_id]" has a null value in JSON.');
        assert(json.containsKey(r'message'),
            'Required key "SendAck[message]" is missing from JSON.');
        assert(json[r'message'] != null,
            'Required key "SendAck[message]" has a null value in JSON.');
        return true;
      }());

      return SendAck(
        clientMessageId: mapValueOfType<String>(json, r'client_message_id')!,
        message: ChatMessage.fromJson(json[r'message'])!,
      );
    }
    return null;
  }

  static List<SendAck> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SendAck>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SendAck.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SendAck> mapFromJson(dynamic json) {
    final map = <String, SendAck>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SendAck.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SendAck-objects as value to a dart map
  static Map<String, List<SendAck>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SendAck>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SendAck.listFromJson(
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
    'message',
  };
}
