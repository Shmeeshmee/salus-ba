//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class NotificationChannels {
  /// Returns a new [NotificationChannels] instance.
  NotificationChannels({
    required this.push,
    required this.email,
    required this.sms,
  });

  bool push;

  bool email;

  /// Opt-in; needs a phone in the profile and a clinic that offers SMS.
  bool sms;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is NotificationChannels &&
          other.push == push &&
          other.email == email &&
          other.sms == sms;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (push.hashCode) + (email.hashCode) + (sms.hashCode);

  @override
  String toString() =>
      'NotificationChannels[push=$push, email=$email, sms=$sms]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'push'] = this.push;
    json[r'email'] = this.email;
    json[r'sms'] = this.sms;
    return json;
  }

  /// Returns a new [NotificationChannels] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static NotificationChannels? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'push'),
            'Required key "NotificationChannels[push]" is missing from JSON.');
        assert(json[r'push'] != null,
            'Required key "NotificationChannels[push]" has a null value in JSON.');
        assert(json.containsKey(r'email'),
            'Required key "NotificationChannels[email]" is missing from JSON.');
        assert(json[r'email'] != null,
            'Required key "NotificationChannels[email]" has a null value in JSON.');
        assert(json.containsKey(r'sms'),
            'Required key "NotificationChannels[sms]" is missing from JSON.');
        assert(json[r'sms'] != null,
            'Required key "NotificationChannels[sms]" has a null value in JSON.');
        return true;
      }());

      return NotificationChannels(
        push: mapValueOfType<bool>(json, r'push')!,
        email: mapValueOfType<bool>(json, r'email')!,
        sms: mapValueOfType<bool>(json, r'sms')!,
      );
    }
    return null;
  }

  static List<NotificationChannels> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <NotificationChannels>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = NotificationChannels.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, NotificationChannels> mapFromJson(dynamic json) {
    final map = <String, NotificationChannels>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = NotificationChannels.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of NotificationChannels-objects as value to a dart map
  static Map<String, List<NotificationChannels>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<NotificationChannels>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = NotificationChannels.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'push',
    'email',
    'sms',
  };
}
