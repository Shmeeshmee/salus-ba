//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class PlatformMonths {
  /// Returns a new [PlatformMonths] instance.
  PlatformMonths({
    this.months = const [],
    this.quarters = const [],
    required this.minCount,
  });

  List<String> months;

  List<String> quarters;

  /// k: no published cell holds fewer patients (A4).
  int minCount;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is PlatformMonths &&
          _deepEquality.equals(other.months, months) &&
          _deepEquality.equals(other.quarters, quarters) &&
          other.minCount == minCount;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (months.hashCode) + (quarters.hashCode) + (minCount.hashCode);

  @override
  String toString() =>
      'PlatformMonths[months=$months, quarters=$quarters, minCount=$minCount]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'months'] = this.months;
    json[r'quarters'] = this.quarters;
    json[r'minCount'] = this.minCount;
    return json;
  }

  /// Returns a new [PlatformMonths] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static PlatformMonths? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'months'),
            'Required key "PlatformMonths[months]" is missing from JSON.');
        assert(json[r'months'] != null,
            'Required key "PlatformMonths[months]" has a null value in JSON.');
        assert(json.containsKey(r'quarters'),
            'Required key "PlatformMonths[quarters]" is missing from JSON.');
        assert(json[r'quarters'] != null,
            'Required key "PlatformMonths[quarters]" has a null value in JSON.');
        assert(json.containsKey(r'minCount'),
            'Required key "PlatformMonths[minCount]" is missing from JSON.');
        assert(json[r'minCount'] != null,
            'Required key "PlatformMonths[minCount]" has a null value in JSON.');
        return true;
      }());

      return PlatformMonths(
        months: json[r'months'] is Iterable
            ? (json[r'months'] as Iterable)
                .cast<String>()
                .toList(growable: false)
            : const [],
        quarters: json[r'quarters'] is Iterable
            ? (json[r'quarters'] as Iterable)
                .cast<String>()
                .toList(growable: false)
            : const [],
        minCount: mapValueOfType<int>(json, r'minCount')!,
      );
    }
    return null;
  }

  static List<PlatformMonths> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <PlatformMonths>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = PlatformMonths.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, PlatformMonths> mapFromJson(dynamic json) {
    final map = <String, PlatformMonths>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = PlatformMonths.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of PlatformMonths-objects as value to a dart map
  static Map<String, List<PlatformMonths>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<PlatformMonths>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = PlatformMonths.listFromJson(
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
    'quarters',
    'minCount',
  };
}
