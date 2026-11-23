//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

/// Moves only forward. `past_due`: a banner for the admin from 14 days; `suspended` from 30: new bookings stop, care never does.
enum SubscriptionStatus {
  trial._(r'trial'),
  active._(r'active'),
  pastDue._(r'past_due'),
  suspended._(r'suspended'),
  cancelled._(r'cancelled'),
  ;

  /// Instantiate a new enum with the provided value.
  const SubscriptionStatus._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [SubscriptionStatus] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static SubscriptionStatus? fromJson(dynamic value) =>
      SubscriptionStatusTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [SubscriptionStatus]
  /// that were successfully decoded from the passed [JSON][json].
  static List<SubscriptionStatus> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SubscriptionStatus>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SubscriptionStatus.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [SubscriptionStatus] to String,
/// and [decode] dynamic data back to [SubscriptionStatus].
class SubscriptionStatusTypeTransformer {
  factory SubscriptionStatusTypeTransformer() =>
      _instance ??= const SubscriptionStatusTypeTransformer._();

  const SubscriptionStatusTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(SubscriptionStatus data) => data._value;

  /// Returns the instance of [SubscriptionStatus] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  SubscriptionStatus? decode(dynamic data, {bool allowNull = true}) {
    if (data is SubscriptionStatus) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'trial':
          return SubscriptionStatus.trial;
        case r'active':
          return SubscriptionStatus.active;
        case r'past_due':
          return SubscriptionStatus.pastDue;
        case r'suspended':
          return SubscriptionStatus.suspended;
        case r'cancelled':
          return SubscriptionStatus.cancelled;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static SubscriptionStatusTypeTransformer? _instance;
}
