//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

/// http://hl7.org/fhir/request-status: `draft` laid out, `active` booked, `completed` every visit past, `revoked` cancelled.
enum CarePlanStatus {
  draft._(r'draft'),
  active._(r'active'),
  completed._(r'completed'),
  revoked._(r'revoked'),
  ;

  /// Instantiate a new enum with the provided value.
  const CarePlanStatus._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [CarePlanStatus] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static CarePlanStatus? fromJson(dynamic value) =>
      CarePlanStatusTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [CarePlanStatus]
  /// that were successfully decoded from the passed [JSON][json].
  static List<CarePlanStatus> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <CarePlanStatus>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = CarePlanStatus.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [CarePlanStatus] to String,
/// and [decode] dynamic data back to [CarePlanStatus].
class CarePlanStatusTypeTransformer {
  factory CarePlanStatusTypeTransformer() =>
      _instance ??= const CarePlanStatusTypeTransformer._();

  const CarePlanStatusTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(CarePlanStatus data) => data._value;

  /// Returns the instance of [CarePlanStatus] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  CarePlanStatus? decode(dynamic data, {bool allowNull = true}) {
    if (data is CarePlanStatus) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'draft':
          return CarePlanStatus.draft;
        case r'active':
          return CarePlanStatus.active;
        case r'completed':
          return CarePlanStatus.completed;
        case r'revoked':
          return CarePlanStatus.revoked;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static CarePlanStatusTypeTransformer? _instance;
}
