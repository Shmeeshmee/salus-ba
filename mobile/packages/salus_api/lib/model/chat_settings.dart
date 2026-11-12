//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ChatSettings {
  /// Returns a new [ChatSettings] instance.
  ChatSettings({
    this.hours = const [],
    required this.responseTimeHours,
    required this.emailAfterMinutes,
  });

  List<ChatHour> hours;

  /// Minimum value: 1
  /// Maximum value: 168
  int responseTimeHours;

  /// How long a message stays unread before its reader gets an e-mail.
  ///
  /// Minimum value: 5
  /// Maximum value: 10080
  int emailAfterMinutes;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ChatSettings &&
          _deepEquality.equals(other.hours, hours) &&
          other.responseTimeHours == responseTimeHours &&
          other.emailAfterMinutes == emailAfterMinutes;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (hours.hashCode) +
      (responseTimeHours.hashCode) +
      (emailAfterMinutes.hashCode);

  @override
  String toString() =>
      'ChatSettings[hours=$hours, responseTimeHours=$responseTimeHours, emailAfterMinutes=$emailAfterMinutes]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'hours'] = this.hours;
    json[r'response_time_hours'] = this.responseTimeHours;
    json[r'email_after_minutes'] = this.emailAfterMinutes;
    return json;
  }

  /// Returns a new [ChatSettings] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ChatSettings? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'hours'),
            'Required key "ChatSettings[hours]" is missing from JSON.');
        assert(json[r'hours'] != null,
            'Required key "ChatSettings[hours]" has a null value in JSON.');
        assert(json.containsKey(r'response_time_hours'),
            'Required key "ChatSettings[response_time_hours]" is missing from JSON.');
        assert(json[r'response_time_hours'] != null,
            'Required key "ChatSettings[response_time_hours]" has a null value in JSON.');
        assert(json.containsKey(r'email_after_minutes'),
            'Required key "ChatSettings[email_after_minutes]" is missing from JSON.');
        assert(json[r'email_after_minutes'] != null,
            'Required key "ChatSettings[email_after_minutes]" has a null value in JSON.');
        return true;
      }());

      return ChatSettings(
        hours: ChatHour.listFromJson(json[r'hours']),
        responseTimeHours: mapValueOfType<int>(json, r'response_time_hours')!,
        emailAfterMinutes: mapValueOfType<int>(json, r'email_after_minutes')!,
      );
    }
    return null;
  }

  static List<ChatSettings> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatSettings>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatSettings.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ChatSettings> mapFromJson(dynamic json) {
    final map = <String, ChatSettings>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ChatSettings.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ChatSettings-objects as value to a dart map
  static Map<String, List<ChatSettings>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ChatSettings>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ChatSettings.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'hours',
    'response_time_hours',
    'email_after_minutes',
  };
}
