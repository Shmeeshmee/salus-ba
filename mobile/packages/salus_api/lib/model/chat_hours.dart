//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ChatHours {
  /// Returns a new [ChatHours] instance.
  ChatHours({
    required this.timeZone,
    this.hours = const [],
    required this.responseTimeHours,
    required this.openNow,
  });

  /// The clinic's IANA time zone, which the hours are in.
  String timeZone;

  List<ChatHour> hours;

  /// The response time the clinic states.
  ///
  /// Minimum value: 1
  /// Maximum value: 168
  int responseTimeHours;

  /// Within the hours now, in the clinic's zone; true when it states none.
  bool openNow;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ChatHours &&
          other.timeZone == timeZone &&
          _deepEquality.equals(other.hours, hours) &&
          other.responseTimeHours == responseTimeHours &&
          other.openNow == openNow;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (timeZone.hashCode) +
      (hours.hashCode) +
      (responseTimeHours.hashCode) +
      (openNow.hashCode);

  @override
  String toString() =>
      'ChatHours[timeZone=$timeZone, hours=$hours, responseTimeHours=$responseTimeHours, openNow=$openNow]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'time_zone'] = this.timeZone;
    json[r'hours'] = this.hours;
    json[r'response_time_hours'] = this.responseTimeHours;
    json[r'open_now'] = this.openNow;
    return json;
  }

  /// Returns a new [ChatHours] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ChatHours? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'time_zone'),
            'Required key "ChatHours[time_zone]" is missing from JSON.');
        assert(json[r'time_zone'] != null,
            'Required key "ChatHours[time_zone]" has a null value in JSON.');
        assert(json.containsKey(r'hours'),
            'Required key "ChatHours[hours]" is missing from JSON.');
        assert(json[r'hours'] != null,
            'Required key "ChatHours[hours]" has a null value in JSON.');
        assert(json.containsKey(r'response_time_hours'),
            'Required key "ChatHours[response_time_hours]" is missing from JSON.');
        assert(json[r'response_time_hours'] != null,
            'Required key "ChatHours[response_time_hours]" has a null value in JSON.');
        assert(json.containsKey(r'open_now'),
            'Required key "ChatHours[open_now]" is missing from JSON.');
        assert(json[r'open_now'] != null,
            'Required key "ChatHours[open_now]" has a null value in JSON.');
        return true;
      }());

      return ChatHours(
        timeZone: mapValueOfType<String>(json, r'time_zone')!,
        hours: ChatHour.listFromJson(json[r'hours']),
        responseTimeHours: mapValueOfType<int>(json, r'response_time_hours')!,
        openNow: mapValueOfType<bool>(json, r'open_now')!,
      );
    }
    return null;
  }

  static List<ChatHours> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ChatHours>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ChatHours.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ChatHours> mapFromJson(dynamic json) {
    final map = <String, ChatHours>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ChatHours.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ChatHours-objects as value to a dart map
  static Map<String, List<ChatHours>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ChatHours>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ChatHours.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'time_zone',
    'hours',
    'response_time_hours',
    'open_now',
  };
}
