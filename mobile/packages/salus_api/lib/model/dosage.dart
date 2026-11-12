//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Dosage {
  /// Returns a new [Dosage] instance.
  Dosage({
    this.doseUnit,
    this.periodUnit,
    this.doseValue,
    this.period,
    this.frequency,
    this.asNeeded,
  });

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DoseUnit? doseUnit;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  PeriodUnit? periodUnit;

  /// Minimum value: 0
  /// Maximum value: 100000
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  num? doseValue;

  /// Minimum value: 0
  /// Maximum value: 1000
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  num? period;

  /// Minimum value: 1
  /// Maximum value: 100
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  int? frequency;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  bool? asNeeded;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Dosage &&
          other.doseUnit == doseUnit &&
          other.periodUnit == periodUnit &&
          other.doseValue == doseValue &&
          other.period == period &&
          other.frequency == frequency &&
          other.asNeeded == asNeeded;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (doseUnit == null ? 0 : doseUnit!.hashCode) +
      (periodUnit == null ? 0 : periodUnit!.hashCode) +
      (doseValue == null ? 0 : doseValue!.hashCode) +
      (period == null ? 0 : period!.hashCode) +
      (frequency == null ? 0 : frequency!.hashCode) +
      (asNeeded == null ? 0 : asNeeded!.hashCode);

  @override
  String toString() =>
      'Dosage[doseUnit=$doseUnit, periodUnit=$periodUnit, doseValue=$doseValue, period=$period, frequency=$frequency, asNeeded=$asNeeded]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    if (this.doseUnit != null) {
      json[r'doseUnit'] = this.doseUnit;
    } else {
      json[r'doseUnit'] = null;
    }
    if (this.periodUnit != null) {
      json[r'periodUnit'] = this.periodUnit;
    } else {
      json[r'periodUnit'] = null;
    }
    if (this.doseValue != null) {
      json[r'doseValue'] = this.doseValue;
    } else {
      json[r'doseValue'] = null;
    }
    if (this.period != null) {
      json[r'period'] = this.period;
    } else {
      json[r'period'] = null;
    }
    if (this.frequency != null) {
      json[r'frequency'] = this.frequency;
    } else {
      json[r'frequency'] = null;
    }
    if (this.asNeeded != null) {
      json[r'asNeeded'] = this.asNeeded;
    } else {
      json[r'asNeeded'] = null;
    }
    return json;
  }

  /// Returns a new [Dosage] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Dosage? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        return true;
      }());

      return Dosage(
        doseUnit: DoseUnit.fromJson(json[r'doseUnit']),
        periodUnit: PeriodUnit.fromJson(json[r'periodUnit']),
        doseValue: json[r'doseValue'] == null
            ? null
            : num.parse('${json[r'doseValue']}'),
        period:
            json[r'period'] == null ? null : num.parse('${json[r'period']}'),
        frequency: mapValueOfType<int>(json, r'frequency'),
        asNeeded: mapValueOfType<bool>(json, r'asNeeded'),
      );
    }
    return null;
  }

  static List<Dosage> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Dosage>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Dosage.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Dosage> mapFromJson(dynamic json) {
    final map = <String, Dosage>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Dosage.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Dosage-objects as value to a dart map
  static Map<String, List<Dosage>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Dosage>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Dosage.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{};
}
