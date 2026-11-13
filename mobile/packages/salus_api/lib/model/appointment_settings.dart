//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class AppointmentSettings {
  /// Returns a new [AppointmentSettings] instance.
  AppointmentSettings({
    this.reminderOffsetsMinutes = const {},
    required this.smsReminders,
    required this.rescheduleNoticeHours,
    required this.rescheduleMax,
    required this.waitlistEnabled,
    required this.waitlistHoldMinutes,
    required this.noshowFlagThreshold,
    required this.noshowFlagWindowDays,
  });

  /// When reminders go out, in minutes before the start; empty for none.
  Set<int> reminderOffsetsMinutes;

  /// The paid SMS option: reminders and waitlist offers also by SMS, to patients who opted in.
  bool smsReminders;

  /// A patient reschedules until this many hours before the start.
  ///
  /// Minimum value: 0
  /// Maximum value: 168
  int rescheduleNoticeHours;

  /// How often an appointment may be rescheduled; 0 turns self-service rescheduling off.
  ///
  /// Minimum value: 0
  /// Maximum value: 5
  int rescheduleMax;

  bool waitlistEnabled;

  /// How long a freed slot is held for the patient it is offered to.
  ///
  /// Minimum value: 5
  /// Maximum value: 120
  int waitlistHoldMinutes;

  /// Staff see a flag from this many no-shows in the window; 0 for off.
  ///
  /// Minimum value: 0
  /// Maximum value: 10
  int noshowFlagThreshold;

  /// Minimum value: 30
  /// Maximum value: 730
  int noshowFlagWindowDays;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is AppointmentSettings &&
          _deepEquality.equals(
              other.reminderOffsetsMinutes, reminderOffsetsMinutes) &&
          other.smsReminders == smsReminders &&
          other.rescheduleNoticeHours == rescheduleNoticeHours &&
          other.rescheduleMax == rescheduleMax &&
          other.waitlistEnabled == waitlistEnabled &&
          other.waitlistHoldMinutes == waitlistHoldMinutes &&
          other.noshowFlagThreshold == noshowFlagThreshold &&
          other.noshowFlagWindowDays == noshowFlagWindowDays;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (reminderOffsetsMinutes.hashCode) +
      (smsReminders.hashCode) +
      (rescheduleNoticeHours.hashCode) +
      (rescheduleMax.hashCode) +
      (waitlistEnabled.hashCode) +
      (waitlistHoldMinutes.hashCode) +
      (noshowFlagThreshold.hashCode) +
      (noshowFlagWindowDays.hashCode);

  @override
  String toString() =>
      'AppointmentSettings[reminderOffsetsMinutes=$reminderOffsetsMinutes, smsReminders=$smsReminders, rescheduleNoticeHours=$rescheduleNoticeHours, rescheduleMax=$rescheduleMax, waitlistEnabled=$waitlistEnabled, waitlistHoldMinutes=$waitlistHoldMinutes, noshowFlagThreshold=$noshowFlagThreshold, noshowFlagWindowDays=$noshowFlagWindowDays]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'reminder_offsets_minutes'] =
        this.reminderOffsetsMinutes.toList(growable: false);
    json[r'sms_reminders'] = this.smsReminders;
    json[r'reschedule_notice_hours'] = this.rescheduleNoticeHours;
    json[r'reschedule_max'] = this.rescheduleMax;
    json[r'waitlist_enabled'] = this.waitlistEnabled;
    json[r'waitlist_hold_minutes'] = this.waitlistHoldMinutes;
    json[r'noshow_flag_threshold'] = this.noshowFlagThreshold;
    json[r'noshow_flag_window_days'] = this.noshowFlagWindowDays;
    return json;
  }

  /// Returns a new [AppointmentSettings] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static AppointmentSettings? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'reminder_offsets_minutes'),
            'Required key "AppointmentSettings[reminder_offsets_minutes]" is missing from JSON.');
        assert(json[r'reminder_offsets_minutes'] != null,
            'Required key "AppointmentSettings[reminder_offsets_minutes]" has a null value in JSON.');
        assert(json.containsKey(r'sms_reminders'),
            'Required key "AppointmentSettings[sms_reminders]" is missing from JSON.');
        assert(json[r'sms_reminders'] != null,
            'Required key "AppointmentSettings[sms_reminders]" has a null value in JSON.');
        assert(json.containsKey(r'reschedule_notice_hours'),
            'Required key "AppointmentSettings[reschedule_notice_hours]" is missing from JSON.');
        assert(json[r'reschedule_notice_hours'] != null,
            'Required key "AppointmentSettings[reschedule_notice_hours]" has a null value in JSON.');
        assert(json.containsKey(r'reschedule_max'),
            'Required key "AppointmentSettings[reschedule_max]" is missing from JSON.');
        assert(json[r'reschedule_max'] != null,
            'Required key "AppointmentSettings[reschedule_max]" has a null value in JSON.');
        assert(json.containsKey(r'waitlist_enabled'),
            'Required key "AppointmentSettings[waitlist_enabled]" is missing from JSON.');
        assert(json[r'waitlist_enabled'] != null,
            'Required key "AppointmentSettings[waitlist_enabled]" has a null value in JSON.');
        assert(json.containsKey(r'waitlist_hold_minutes'),
            'Required key "AppointmentSettings[waitlist_hold_minutes]" is missing from JSON.');
        assert(json[r'waitlist_hold_minutes'] != null,
            'Required key "AppointmentSettings[waitlist_hold_minutes]" has a null value in JSON.');
        assert(json.containsKey(r'noshow_flag_threshold'),
            'Required key "AppointmentSettings[noshow_flag_threshold]" is missing from JSON.');
        assert(json[r'noshow_flag_threshold'] != null,
            'Required key "AppointmentSettings[noshow_flag_threshold]" has a null value in JSON.');
        assert(json.containsKey(r'noshow_flag_window_days'),
            'Required key "AppointmentSettings[noshow_flag_window_days]" is missing from JSON.');
        assert(json[r'noshow_flag_window_days'] != null,
            'Required key "AppointmentSettings[noshow_flag_window_days]" has a null value in JSON.');
        return true;
      }());

      return AppointmentSettings(
        reminderOffsetsMinutes: json[r'reminder_offsets_minutes'] is Iterable
            ? (json[r'reminder_offsets_minutes'] as Iterable)
                .cast<int>()
                .toSet()
            : const {},
        smsReminders: mapValueOfType<bool>(json, r'sms_reminders')!,
        rescheduleNoticeHours:
            mapValueOfType<int>(json, r'reschedule_notice_hours')!,
        rescheduleMax: mapValueOfType<int>(json, r'reschedule_max')!,
        waitlistEnabled: mapValueOfType<bool>(json, r'waitlist_enabled')!,
        waitlistHoldMinutes:
            mapValueOfType<int>(json, r'waitlist_hold_minutes')!,
        noshowFlagThreshold:
            mapValueOfType<int>(json, r'noshow_flag_threshold')!,
        noshowFlagWindowDays:
            mapValueOfType<int>(json, r'noshow_flag_window_days')!,
      );
    }
    return null;
  }

  static List<AppointmentSettings> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <AppointmentSettings>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = AppointmentSettings.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, AppointmentSettings> mapFromJson(dynamic json) {
    final map = <String, AppointmentSettings>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = AppointmentSettings.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of AppointmentSettings-objects as value to a dart map
  static Map<String, List<AppointmentSettings>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<AppointmentSettings>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = AppointmentSettings.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'reminder_offsets_minutes',
    'sms_reminders',
    'reschedule_notice_hours',
    'reschedule_max',
    'waitlist_enabled',
    'waitlist_hold_minutes',
    'noshow_flag_threshold',
    'noshow_flag_window_days',
  };
}
