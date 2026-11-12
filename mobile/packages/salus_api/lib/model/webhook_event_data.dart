//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class WebhookEventData {
  /// Returns a new [WebhookEventData] instance.
  WebhookEventData({
    required this.conversationId,
    required this.messageId,
    this.message,
  });

  String conversationId;

  String messageId;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  WebhookEventDataMessage? message;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is WebhookEventData &&
          other.conversationId == conversationId &&
          other.messageId == messageId &&
          other.message == message;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (conversationId.hashCode) +
      (messageId.hashCode) +
      (message == null ? 0 : message!.hashCode);

  @override
  String toString() =>
      'WebhookEventData[conversationId=$conversationId, messageId=$messageId, message=$message]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'conversation_id'] = this.conversationId;
    json[r'message_id'] = this.messageId;
    if (this.message != null) {
      json[r'message'] = this.message;
    } else {
      json[r'message'] = null;
    }
    return json;
  }

  /// Returns a new [WebhookEventData] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static WebhookEventData? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'conversation_id'),
            'Required key "WebhookEventData[conversation_id]" is missing from JSON.');
        assert(json[r'conversation_id'] != null,
            'Required key "WebhookEventData[conversation_id]" has a null value in JSON.');
        assert(json.containsKey(r'message_id'),
            'Required key "WebhookEventData[message_id]" is missing from JSON.');
        assert(json[r'message_id'] != null,
            'Required key "WebhookEventData[message_id]" has a null value in JSON.');
        return true;
      }());

      return WebhookEventData(
        conversationId: mapValueOfType<String>(json, r'conversation_id')!,
        messageId: mapValueOfType<String>(json, r'message_id')!,
        message: WebhookEventDataMessage.fromJson(json[r'message']),
      );
    }
    return null;
  }

  static List<WebhookEventData> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <WebhookEventData>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = WebhookEventData.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, WebhookEventData> mapFromJson(dynamic json) {
    final map = <String, WebhookEventData>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = WebhookEventData.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of WebhookEventData-objects as value to a dart map
  static Map<String, List<WebhookEventData>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<WebhookEventData>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = WebhookEventData.listFromJson(
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
    'message_id',
  };
}
