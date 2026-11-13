//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class AppointmentActions {
  /// Returns a new [AppointmentActions] instance.
  AppointmentActions({
    required this.confirm,
    required this.cancel,
    this.cancelUntil,
    required this.reschedule,
    this.rescheduleUntil,
    required this.reschedulesLeft,
  });

  bool confirm;

  bool cancel;

  /// Absent when cancelling is not possible.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? cancelUntil;

  bool reschedule;

  /// Absent when rescheduling is not possible.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? rescheduleUntil;

  /// Minimum value: 0
  int reschedulesLeft;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is AppointmentActions &&
          other.confirm == confirm &&
          other.cancel == cancel &&
          other.cancelUntil == cancelUntil &&
          other.reschedule == reschedule &&
          other.rescheduleUntil == rescheduleUntil &&
          other.reschedulesLeft == reschedulesLeft;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (confirm.hashCode) +
      (cancel.hashCode) +
      (cancelUntil == null ? 0 : cancelUntil!.hashCode) +
      (reschedule.hashCode) +
      (rescheduleUntil == null ? 0 : rescheduleUntil!.hashCode) +
      (reschedulesLeft.hashCode);

  @override
  String toString() =>
      'AppointmentActions[confirm=$confirm, cancel=$cancel, cancelUntil=$cancelUntil, reschedule=$reschedule, rescheduleUntil=$rescheduleUntil, reschedulesLeft=$reschedulesLeft]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'confirm'] = this.confirm;
    json[r'cancel'] = this.cancel;
    if (this.cancelUntil != null) {
      json[r'cancel_until'] = this.cancelUntil!.toUtc().toIso8601String();
    } else {
      json[r'cancel_until'] = null;
    }
    json[r'reschedule'] = this.reschedule;
    if (this.rescheduleUntil != null) {
      json[r'reschedule_until'] =
          this.rescheduleUntil!.toUtc().toIso8601String();
    } else {
      json[r'reschedule_until'] = null;
    }
    json[r'reschedules_left'] = this.reschedulesLeft;
    return json;
  }

  /// Returns a new [AppointmentActions] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static AppointmentActions? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'confirm'),
            'Required key "AppointmentActions[confirm]" is missing from JSON.');
        assert(json[r'confirm'] != null,
            'Required key "AppointmentActions[confirm]" has a null value in JSON.');
        assert(json.containsKey(r'cancel'),
            'Required key "AppointmentActions[cancel]" is missing from JSON.');
        assert(json[r'cancel'] != null,
            'Required key "AppointmentActions[cancel]" has a null value in JSON.');
        assert(json.containsKey(r'reschedule'),
            'Required key "AppointmentActions[reschedule]" is missing from JSON.');
        assert(json[r'reschedule'] != null,
            'Required key "AppointmentActions[reschedule]" has a null value in JSON.');
        assert(json.containsKey(r'reschedules_left'),
            'Required key "AppointmentActions[reschedules_left]" is missing from JSON.');
        assert(json[r'reschedules_left'] != null,
            'Required key "AppointmentActions[reschedules_left]" has a null value in JSON.');
        return true;
      }());

      return AppointmentActions(
        confirm: mapValueOfType<bool>(json, r'confirm')!,
        cancel: mapValueOfType<bool>(json, r'cancel')!,
        cancelUntil: mapDateTime(json, r'cancel_until', r''),
        reschedule: mapValueOfType<bool>(json, r'reschedule')!,
        rescheduleUntil: mapDateTime(json, r'reschedule_until', r''),
        reschedulesLeft: mapValueOfType<int>(json, r'reschedules_left')!,
      );
    }
    return null;
  }

  static List<AppointmentActions> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <AppointmentActions>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = AppointmentActions.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, AppointmentActions> mapFromJson(dynamic json) {
    final map = <String, AppointmentActions>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = AppointmentActions.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of AppointmentActions-objects as value to a dart map
  static Map<String, List<AppointmentActions>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<AppointmentActions>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = AppointmentActions.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'confirm',
    'cancel',
    'reschedule',
    'reschedules_left',
  };
}
