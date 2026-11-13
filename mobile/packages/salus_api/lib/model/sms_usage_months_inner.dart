//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SmsUsageMonthsInner {
  /// Returns a new [SmsUsageMonthsInner] instance.
  SmsUsageMonthsInner({
    required this.month,
    required this.messages,
    required this.segments,
  });

  String month;

  /// Minimum value: 0
  int messages;

  /// Minimum value: 0
  int segments;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SmsUsageMonthsInner &&
          other.month == month &&
          other.messages == messages &&
          other.segments == segments;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (month.hashCode) + (messages.hashCode) + (segments.hashCode);

  @override
  String toString() =>
      'SmsUsageMonthsInner[month=$month, messages=$messages, segments=$segments]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'month'] = this.month;
    json[r'messages'] = this.messages;
    json[r'segments'] = this.segments;
    return json;
  }

  /// Returns a new [SmsUsageMonthsInner] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SmsUsageMonthsInner? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'month'),
            'Required key "SmsUsageMonthsInner[month]" is missing from JSON.');
        assert(json[r'month'] != null,
            'Required key "SmsUsageMonthsInner[month]" has a null value in JSON.');
        assert(json.containsKey(r'messages'),
            'Required key "SmsUsageMonthsInner[messages]" is missing from JSON.');
        assert(json[r'messages'] != null,
            'Required key "SmsUsageMonthsInner[messages]" has a null value in JSON.');
        assert(json.containsKey(r'segments'),
            'Required key "SmsUsageMonthsInner[segments]" is missing from JSON.');
        assert(json[r'segments'] != null,
            'Required key "SmsUsageMonthsInner[segments]" has a null value in JSON.');
        return true;
      }());

      return SmsUsageMonthsInner(
        month: mapValueOfType<String>(json, r'month')!,
        messages: mapValueOfType<int>(json, r'messages')!,
        segments: mapValueOfType<int>(json, r'segments')!,
      );
    }
    return null;
  }

  static List<SmsUsageMonthsInner> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SmsUsageMonthsInner>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SmsUsageMonthsInner.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SmsUsageMonthsInner> mapFromJson(dynamic json) {
    final map = <String, SmsUsageMonthsInner>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SmsUsageMonthsInner.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SmsUsageMonthsInner-objects as value to a dart map
  static Map<String, List<SmsUsageMonthsInner>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SmsUsageMonthsInner>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SmsUsageMonthsInner.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'month',
    'messages',
    'segments',
  };
}
