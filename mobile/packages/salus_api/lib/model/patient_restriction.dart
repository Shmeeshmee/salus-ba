//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class PatientRestriction {
  /// Returns a new [PatientRestriction] instance.
  PatientRestriction({
    required this.restricted,
    this.restrictedAt,
    this.by,
  });

  bool restricted;

  /// While restricted.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? restrictedAt;

  /// While restricted.
  PatientRestrictionByEnum? by;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is PatientRestriction &&
          other.restricted == restricted &&
          other.restrictedAt == restrictedAt &&
          other.by == by;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (restricted.hashCode) +
      (restrictedAt == null ? 0 : restrictedAt!.hashCode) +
      (by == null ? 0 : by!.hashCode);

  @override
  String toString() =>
      'PatientRestriction[restricted=$restricted, restrictedAt=$restrictedAt, by=$by]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'restricted'] = this.restricted;
    if (this.restrictedAt != null) {
      json[r'restrictedAt'] = this.restrictedAt!.toUtc().toIso8601String();
    } else {
      json[r'restrictedAt'] = null;
    }
    if (this.by != null) {
      json[r'by'] = this.by;
    } else {
      json[r'by'] = null;
    }
    return json;
  }

  /// Returns a new [PatientRestriction] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static PatientRestriction? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'restricted'),
            'Required key "PatientRestriction[restricted]" is missing from JSON.');
        assert(json[r'restricted'] != null,
            'Required key "PatientRestriction[restricted]" has a null value in JSON.');
        return true;
      }());

      return PatientRestriction(
        restricted: mapValueOfType<bool>(json, r'restricted')!,
        restrictedAt: mapDateTime(json, r'restrictedAt', r''),
        by: PatientRestrictionByEnum.fromJson(json[r'by']),
      );
    }
    return null;
  }

  static List<PatientRestriction> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <PatientRestriction>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = PatientRestriction.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, PatientRestriction> mapFromJson(dynamic json) {
    final map = <String, PatientRestriction>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = PatientRestriction.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of PatientRestriction-objects as value to a dart map
  static Map<String, List<PatientRestriction>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<PatientRestriction>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = PatientRestriction.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'restricted',
  };
}

/// While restricted.
enum PatientRestrictionByEnum {
  clinic._(r'clinic'),
  erasure._(r'erasure'),
  ;

  /// Instantiate a new enum with the provided value.
  const PatientRestrictionByEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [PatientRestrictionByEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static PatientRestrictionByEnum? fromJson(dynamic value) =>
      PatientRestrictionByEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [PatientRestrictionByEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<PatientRestrictionByEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <PatientRestrictionByEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = PatientRestrictionByEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [PatientRestrictionByEnum] to String,
/// and [decode] dynamic data back to [PatientRestrictionByEnum].
class PatientRestrictionByEnumTypeTransformer {
  factory PatientRestrictionByEnumTypeTransformer() =>
      _instance ??= const PatientRestrictionByEnumTypeTransformer._();

  const PatientRestrictionByEnumTypeTransformer._();

  String encode(PatientRestrictionByEnum data) => data._value;

  /// Returns the instance of [PatientRestrictionByEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  PatientRestrictionByEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is PatientRestrictionByEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'clinic':
          return PatientRestrictionByEnum.clinic;
        case r'erasure':
          return PatientRestrictionByEnum.erasure;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static PatientRestrictionByEnumTypeTransformer? _instance;
}
