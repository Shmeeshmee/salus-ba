//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

enum AllergySeverity {
  mild._(r'mild'),
  moderate._(r'moderate'),
  severe._(r'severe'),
  ;

  /// Instantiate a new enum with the provided value.
  const AllergySeverity._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [AllergySeverity] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static AllergySeverity? fromJson(dynamic value) =>
      AllergySeverityTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [AllergySeverity]
  /// that were successfully decoded from the passed [JSON][json].
  static List<AllergySeverity> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <AllergySeverity>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = AllergySeverity.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [AllergySeverity] to String,
/// and [decode] dynamic data back to [AllergySeverity].
class AllergySeverityTypeTransformer {
  factory AllergySeverityTypeTransformer() =>
      _instance ??= const AllergySeverityTypeTransformer._();

  const AllergySeverityTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(AllergySeverity data) => data._value;

  /// Returns the instance of [AllergySeverity] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  AllergySeverity? decode(dynamic data, {bool allowNull = true}) {
    if (data is AllergySeverity) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'mild':
          return AllergySeverity.mild;
        case r'moderate':
          return AllergySeverity.moderate;
        case r'severe':
          return AllergySeverity.severe;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static AllergySeverityTypeTransformer? _instance;
}
