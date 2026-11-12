//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

/// A UCUM code, as FHIR Timing.repeat.periodUnit.
enum PeriodUnit {
  h._(r'h'),
  d._(r'd'),
  wk._(r'wk'),
  mo._(r'mo'),
  ;

  /// Instantiate a new enum with the provided value.
  const PeriodUnit._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [PeriodUnit] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static PeriodUnit? fromJson(dynamic value) =>
      PeriodUnitTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [PeriodUnit]
  /// that were successfully decoded from the passed [JSON][json].
  static List<PeriodUnit> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <PeriodUnit>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = PeriodUnit.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [PeriodUnit] to String,
/// and [decode] dynamic data back to [PeriodUnit].
class PeriodUnitTypeTransformer {
  factory PeriodUnitTypeTransformer() =>
      _instance ??= const PeriodUnitTypeTransformer._();

  const PeriodUnitTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(PeriodUnit data) => data._value;

  /// Returns the instance of [PeriodUnit] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  PeriodUnit? decode(dynamic data, {bool allowNull = true}) {
    if (data is PeriodUnit) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'h':
          return PeriodUnit.h;
        case r'd':
          return PeriodUnit.d;
        case r'wk':
          return PeriodUnit.wk;
        case r'mo':
          return PeriodUnit.mo;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static PeriodUnitTypeTransformer? _instance;
}
