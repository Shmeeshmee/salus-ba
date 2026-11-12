//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class WebhookEvent {
  /// Returns a new [WebhookEvent] instance.
  WebhookEvent({
    required this.id,
    required this.type,
    required this.organizationId,
    required this.data,
  });

  /// The event's id, the same on every attempt.
  String id;

  WebhookEventType type;

  String organizationId;

  WebhookEventData data;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is WebhookEvent &&
          other.id == id &&
          other.type == type &&
          other.organizationId == organizationId &&
          other.data == data;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (type.hashCode) +
      (organizationId.hashCode) +
      (data.hashCode);

  @override
  String toString() =>
      'WebhookEvent[id=$id, type=$type, organizationId=$organizationId, data=$data]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'type'] = this.type;
    json[r'organization_id'] = this.organizationId;
    json[r'data'] = this.data;
    return json;
  }

  /// Returns a new [WebhookEvent] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static WebhookEvent? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "WebhookEvent[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "WebhookEvent[id]" has a null value in JSON.');
        assert(json.containsKey(r'type'),
            'Required key "WebhookEvent[type]" is missing from JSON.');
        assert(json[r'type'] != null,
            'Required key "WebhookEvent[type]" has a null value in JSON.');
        assert(json.containsKey(r'organization_id'),
            'Required key "WebhookEvent[organization_id]" is missing from JSON.');
        assert(json[r'organization_id'] != null,
            'Required key "WebhookEvent[organization_id]" has a null value in JSON.');
        assert(json.containsKey(r'data'),
            'Required key "WebhookEvent[data]" is missing from JSON.');
        assert(json[r'data'] != null,
            'Required key "WebhookEvent[data]" has a null value in JSON.');
        return true;
      }());

      return WebhookEvent(
        id: mapValueOfType<String>(json, r'id')!,
        type: WebhookEventType.fromJson(json[r'type'])!,
        organizationId: mapValueOfType<String>(json, r'organization_id')!,
        data: WebhookEventData.fromJson(json[r'data'])!,
      );
    }
    return null;
  }

  static List<WebhookEvent> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <WebhookEvent>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = WebhookEvent.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, WebhookEvent> mapFromJson(dynamic json) {
    final map = <String, WebhookEvent>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = WebhookEvent.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of WebhookEvent-objects as value to a dart map
  static Map<String, List<WebhookEvent>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<WebhookEvent>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = WebhookEvent.listFromJson(
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
    'type',
    'organization_id',
    'data',
  };
}
