//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class NotificationSettings {
  /// Returns a new [NotificationSettings] instance.
  NotificationSettings({
    required this.channels,
    this.quietHours,
    this.timeZone,
  });

  NotificationChannels channels;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  QuietHours? quietHours;

  /// An IANA zone, e.g. Asia/Dubai.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? timeZone;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is NotificationSettings &&
          other.channels == channels &&
          other.quietHours == quietHours &&
          other.timeZone == timeZone;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (channels.hashCode) +
      (quietHours == null ? 0 : quietHours!.hashCode) +
      (timeZone == null ? 0 : timeZone!.hashCode);

  @override
  String toString() =>
      'NotificationSettings[channels=$channels, quietHours=$quietHours, timeZone=$timeZone]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'channels'] = this.channels;
    if (this.quietHours != null) {
      json[r'quiet_hours'] = this.quietHours;
    } else {
      json[r'quiet_hours'] = null;
    }
    if (this.timeZone != null) {
      json[r'time_zone'] = this.timeZone;
    } else {
      json[r'time_zone'] = null;
    }
    return json;
  }

  /// Returns a new [NotificationSettings] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static NotificationSettings? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'channels'),
            'Required key "NotificationSettings[channels]" is missing from JSON.');
        assert(json[r'channels'] != null,
            'Required key "NotificationSettings[channels]" has a null value in JSON.');
        return true;
      }());

      return NotificationSettings(
        channels: NotificationChannels.fromJson(json[r'channels'])!,
        quietHours: QuietHours.fromJson(json[r'quiet_hours']),
        timeZone: mapValueOfType<String>(json, r'time_zone'),
      );
    }
    return null;
  }

  static List<NotificationSettings> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <NotificationSettings>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = NotificationSettings.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, NotificationSettings> mapFromJson(dynamic json) {
    final map = <String, NotificationSettings>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = NotificationSettings.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of NotificationSettings-objects as value to a dart map
  static Map<String, List<NotificationSettings>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<NotificationSettings>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = NotificationSettings.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'channels',
  };
}
