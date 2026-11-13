//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SmsUsage {
  /// Returns a new [SmsUsage] instance.
  SmsUsage({
    this.months = const [],
  });

  List<SmsUsageMonthsInner> months;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SmsUsage && _deepEquality.equals(other.months, months);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (months.hashCode);

  @override
  String toString() => 'SmsUsage[months=$months]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'months'] = this.months;
    return json;
  }

  /// Returns a new [SmsUsage] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SmsUsage? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'months'),
            'Required key "SmsUsage[months]" is missing from JSON.');
        assert(json[r'months'] != null,
            'Required key "SmsUsage[months]" has a null value in JSON.');
        return true;
      }());

      return SmsUsage(
        months: SmsUsageMonthsInner.listFromJson(json[r'months']),
      );
    }
    return null;
  }

  static List<SmsUsage> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SmsUsage>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SmsUsage.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SmsUsage> mapFromJson(dynamic json) {
    final map = <String, SmsUsage>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SmsUsage.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SmsUsage-objects as value to a dart map
  static Map<String, List<SmsUsage>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SmsUsage>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SmsUsage.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'months',
  };
}
