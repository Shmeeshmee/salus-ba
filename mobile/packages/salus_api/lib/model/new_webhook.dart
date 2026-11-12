//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class NewWebhook {
  /// Returns a new [NewWebhook] instance.
  NewWebhook({
    required this.url,
    this.events = const [],
    this.includeClinical = false,
  });

  /// https, to a public host.
  String url;

  List<WebhookEventType> events;

  /// Adds the message's text to each event.
  bool includeClinical;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is NewWebhook &&
          other.url == url &&
          _deepEquality.equals(other.events, events) &&
          other.includeClinical == includeClinical;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (url.hashCode) + (events.hashCode) + (includeClinical.hashCode);

  @override
  String toString() =>
      'NewWebhook[url=$url, events=$events, includeClinical=$includeClinical]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'url'] = this.url;
    json[r'events'] = this.events;
    json[r'include_clinical'] = this.includeClinical;
    return json;
  }

  /// Returns a new [NewWebhook] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static NewWebhook? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'url'),
            'Required key "NewWebhook[url]" is missing from JSON.');
        assert(json[r'url'] != null,
            'Required key "NewWebhook[url]" has a null value in JSON.');
        assert(json.containsKey(r'events'),
            'Required key "NewWebhook[events]" is missing from JSON.');
        assert(json[r'events'] != null,
            'Required key "NewWebhook[events]" has a null value in JSON.');
        return true;
      }());

      return NewWebhook(
        url: mapValueOfType<String>(json, r'url')!,
        events: WebhookEventType.listFromJson(json[r'events']),
        includeClinical:
            mapValueOfType<bool>(json, r'include_clinical') ?? false,
      );
    }
    return null;
  }

  static List<NewWebhook> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <NewWebhook>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = NewWebhook.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, NewWebhook> mapFromJson(dynamic json) {
    final map = <String, NewWebhook>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = NewWebhook.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of NewWebhook-objects as value to a dart map
  static Map<String, List<NewWebhook>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<NewWebhook>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = NewWebhook.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'url',
    'events',
  };
}
