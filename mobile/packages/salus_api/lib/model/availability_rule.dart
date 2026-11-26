//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class AvailabilityRule {
  /// Returns a new [AvailabilityRule] instance.
  AvailabilityRule({
    required this.day,
    required this.start,
    required this.end,
    this.mode,
    this.locationId,
    this.roomId,
    this.roomBusy,
  });

  /// ISO weekday, 1 Monday to 7 Sunday.
  ///
  /// Minimum value: 1
  /// Maximum value: 7
  int day;

  String start;

  /// After start.
  String end;

  /// The visit mode of the rule's slots; always in a response, `video` for every rule made before the clinic system. A PUT may name it (x-salus-package cms-a); a rule that names none is a video rule. The slots generated from the rule carry it, and an appointment keeps the mode of the slot it was booked in.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  VisitMode? mode;

  /// The location of an in-person rule, a location of the clinic (U2 of docs/clinic-cms/WAVE6.md, x-salus-package cms-lc1); never on a video rule. Every in-person rule made before wave 6 is at the clinic's first location. Its slots and the visits booked in them carry it.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? locationId;

  /// Optional: a room of the rule's location (U3, x-salus-package cms-rm1). Two practitioners' in-person hours never overlap in one room: a PUT that would is refused with `resource_busy`.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? roomId;

  /// In a read only, and only when true (x-salus-package cms-rd6): another active practitioner role's in-person hours hold the rule's room at an overlapping time on the same day (a role brought back into use is not re-checked), so some of the rule's slots are not generated. Never sent in a PUT.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  bool? roomBusy;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is AvailabilityRule &&
          other.day == day &&
          other.start == start &&
          other.end == end &&
          other.mode == mode &&
          other.locationId == locationId &&
          other.roomId == roomId &&
          other.roomBusy == roomBusy;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (day.hashCode) +
      (start.hashCode) +
      (end.hashCode) +
      (mode == null ? 0 : mode!.hashCode) +
      (locationId == null ? 0 : locationId!.hashCode) +
      (roomId == null ? 0 : roomId!.hashCode) +
      (roomBusy == null ? 0 : roomBusy!.hashCode);

  @override
  String toString() =>
      'AvailabilityRule[day=$day, start=$start, end=$end, mode=$mode, locationId=$locationId, roomId=$roomId, roomBusy=$roomBusy]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'day'] = this.day;
    json[r'start'] = this.start;
    json[r'end'] = this.end;
    if (this.mode != null) {
      json[r'mode'] = this.mode;
    } else {
      json[r'mode'] = null;
    }
    if (this.locationId != null) {
      json[r'locationId'] = this.locationId;
    } else {
      json[r'locationId'] = null;
    }
    if (this.roomId != null) {
      json[r'roomId'] = this.roomId;
    } else {
      json[r'roomId'] = null;
    }
    if (this.roomBusy != null) {
      json[r'roomBusy'] = this.roomBusy;
    } else {
      json[r'roomBusy'] = null;
    }
    return json;
  }

  /// Returns a new [AvailabilityRule] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static AvailabilityRule? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'day'),
            'Required key "AvailabilityRule[day]" is missing from JSON.');
        assert(json[r'day'] != null,
            'Required key "AvailabilityRule[day]" has a null value in JSON.');
        assert(json.containsKey(r'start'),
            'Required key "AvailabilityRule[start]" is missing from JSON.');
        assert(json[r'start'] != null,
            'Required key "AvailabilityRule[start]" has a null value in JSON.');
        assert(json.containsKey(r'end'),
            'Required key "AvailabilityRule[end]" is missing from JSON.');
        assert(json[r'end'] != null,
            'Required key "AvailabilityRule[end]" has a null value in JSON.');
        return true;
      }());

      return AvailabilityRule(
        day: mapValueOfType<int>(json, r'day')!,
        start: mapValueOfType<String>(json, r'start')!,
        end: mapValueOfType<String>(json, r'end')!,
        mode: VisitMode.fromJson(json[r'mode']),
        locationId: mapValueOfType<String>(json, r'locationId'),
        roomId: mapValueOfType<String>(json, r'roomId'),
        roomBusy: mapValueOfType<bool>(json, r'roomBusy'),
      );
    }
    return null;
  }

  static List<AvailabilityRule> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <AvailabilityRule>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = AvailabilityRule.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, AvailabilityRule> mapFromJson(dynamic json) {
    final map = <String, AvailabilityRule>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = AvailabilityRule.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of AvailabilityRule-objects as value to a dart map
  static Map<String, List<AvailabilityRule>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<AvailabilityRule>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = AvailabilityRule.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'day',
    'start',
    'end',
  };
}
