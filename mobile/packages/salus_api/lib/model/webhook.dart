//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Webhook {
  /// Returns a new [Webhook] instance.
  Webhook({
    required this.createdAt,
    required this.id,
    required this.url,
    this.events = const [],
    required this.includeClinical,
  });

  DateTime createdAt;

  String id;

  String url;

  List<WebhookEventType> events;

  bool includeClinical;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Webhook &&
          other.createdAt == createdAt &&
          other.id == id &&
          other.url == url &&
          _deepEquality.equals(other.events, events) &&
          other.includeClinical == includeClinical;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (createdAt.hashCode) +
      (id.hashCode) +
      (url.hashCode) +
      (events.hashCode) +
      (includeClinical.hashCode);

  @override
  String toString() =>
      'Webhook[createdAt=$createdAt, id=$id, url=$url, events=$events, includeClinical=$includeClinical]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'created_at'] = this.createdAt.toUtc().toIso8601String();
    json[r'id'] = this.id;
    json[r'url'] = this.url;
    json[r'events'] = this.events;
    json[r'include_clinical'] = this.includeClinical;
    return json;
  }

  /// Returns a new [Webhook] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Webhook? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'created_at'),
            'Required key "Webhook[created_at]" is missing from JSON.');
        assert(json[r'created_at'] != null,
            'Required key "Webhook[created_at]" has a null value in JSON.');
        assert(json.containsKey(r'id'),
            'Required key "Webhook[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "Webhook[id]" has a null value in JSON.');
        assert(json.containsKey(r'url'),
            'Required key "Webhook[url]" is missing from JSON.');
        assert(json[r'url'] != null,
            'Required key "Webhook[url]" has a null value in JSON.');
        assert(json.containsKey(r'events'),
            'Required key "Webhook[events]" is missing from JSON.');
        assert(json[r'events'] != null,
            'Required key "Webhook[events]" has a null value in JSON.');
        assert(json.containsKey(r'include_clinical'),
            'Required key "Webhook[include_clinical]" is missing from JSON.');
        assert(json[r'include_clinical'] != null,
            'Required key "Webhook[include_clinical]" has a null value in JSON.');
        return true;
      }());

      return Webhook(
        createdAt: mapDateTime(json, r'created_at', r'')!,
        id: mapValueOfType<String>(json, r'id')!,
        url: mapValueOfType<String>(json, r'url')!,
        events: WebhookEventType.listFromJson(json[r'events']),
        includeClinical: mapValueOfType<bool>(json, r'include_clinical')!,
      );
    }
    return null;
  }

  static List<Webhook> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Webhook>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Webhook.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Webhook> mapFromJson(dynamic json) {
    final map = <String, Webhook>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Webhook.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Webhook-objects as value to a dart map
  static Map<String, List<Webhook>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Webhook>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Webhook.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'created_at',
    'id',
    'url',
    'events',
    'include_clinical',
  };
}
