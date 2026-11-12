//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

enum AllergyClinicalStatus {
  active._(r'active'),
  inactive._(r'inactive'),
  resolved._(r'resolved'),
  ;

  /// Instantiate a new enum with the provided value.
  const AllergyClinicalStatus._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [AllergyClinicalStatus] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static AllergyClinicalStatus? fromJson(dynamic value) =>
      AllergyClinicalStatusTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [AllergyClinicalStatus]
  /// that were successfully decoded from the passed [JSON][json].
  static List<AllergyClinicalStatus> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <AllergyClinicalStatus>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = AllergyClinicalStatus.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [AllergyClinicalStatus] to String,
/// and [decode] dynamic data back to [AllergyClinicalStatus].
class AllergyClinicalStatusTypeTransformer {
  factory AllergyClinicalStatusTypeTransformer() =>
      _instance ??= const AllergyClinicalStatusTypeTransformer._();

  const AllergyClinicalStatusTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(AllergyClinicalStatus data) => data._value;

  /// Returns the instance of [AllergyClinicalStatus] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  AllergyClinicalStatus? decode(dynamic data, {bool allowNull = true}) {
    if (data is AllergyClinicalStatus) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'active':
          return AllergyClinicalStatus.active;
        case r'inactive':
          return AllergyClinicalStatus.inactive;
        case r'resolved':
          return AllergyClinicalStatus.resolved;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static AllergyClinicalStatusTypeTransformer? _instance;
}
