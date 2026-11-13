//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class NoShowFlagsFlagsInner {
  /// Returns a new [NoShowFlagsFlagsInner] instance.
  NoShowFlagsFlagsInner({
    required this.appointmentId,
    required this.noshows,
    required this.windowDays,
  });

  String appointmentId;

  /// Minimum value: 1
  int noshows;

  int windowDays;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is NoShowFlagsFlagsInner &&
          other.appointmentId == appointmentId &&
          other.noshows == noshows &&
          other.windowDays == windowDays;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (appointmentId.hashCode) + (noshows.hashCode) + (windowDays.hashCode);

  @override
  String toString() =>
      'NoShowFlagsFlagsInner[appointmentId=$appointmentId, noshows=$noshows, windowDays=$windowDays]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'appointment_id'] = this.appointmentId;
    json[r'noshows'] = this.noshows;
    json[r'window_days'] = this.windowDays;
    return json;
  }

  /// Returns a new [NoShowFlagsFlagsInner] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static NoShowFlagsFlagsInner? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'appointment_id'),
            'Required key "NoShowFlagsFlagsInner[appointment_id]" is missing from JSON.');
        assert(json[r'appointment_id'] != null,
            'Required key "NoShowFlagsFlagsInner[appointment_id]" has a null value in JSON.');
        assert(json.containsKey(r'noshows'),
            'Required key "NoShowFlagsFlagsInner[noshows]" is missing from JSON.');
        assert(json[r'noshows'] != null,
            'Required key "NoShowFlagsFlagsInner[noshows]" has a null value in JSON.');
        assert(json.containsKey(r'window_days'),
            'Required key "NoShowFlagsFlagsInner[window_days]" is missing from JSON.');
        assert(json[r'window_days'] != null,
            'Required key "NoShowFlagsFlagsInner[window_days]" has a null value in JSON.');
        return true;
      }());

      return NoShowFlagsFlagsInner(
        appointmentId: mapValueOfType<String>(json, r'appointment_id')!,
        noshows: mapValueOfType<int>(json, r'noshows')!,
        windowDays: mapValueOfType<int>(json, r'window_days')!,
      );
    }
    return null;
  }

  static List<NoShowFlagsFlagsInner> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <NoShowFlagsFlagsInner>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = NoShowFlagsFlagsInner.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, NoShowFlagsFlagsInner> mapFromJson(dynamic json) {
    final map = <String, NoShowFlagsFlagsInner>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = NoShowFlagsFlagsInner.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of NoShowFlagsFlagsInner-objects as value to a dart map
  static Map<String, List<NoShowFlagsFlagsInner>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<NoShowFlagsFlagsInner>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = NoShowFlagsFlagsInner.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'appointment_id',
    'noshows',
    'window_days',
  };
}
