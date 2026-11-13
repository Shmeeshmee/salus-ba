//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

enum AppointmentChangeVia {
  app._(r'app'),
  link._(r'link'),
  staff._(r'staff'),
  system._(r'system'),
  ;

  /// Instantiate a new enum with the provided value.
  const AppointmentChangeVia._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [AppointmentChangeVia] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static AppointmentChangeVia? fromJson(dynamic value) =>
      AppointmentChangeViaTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [AppointmentChangeVia]
  /// that were successfully decoded from the passed [JSON][json].
  static List<AppointmentChangeVia> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <AppointmentChangeVia>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = AppointmentChangeVia.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [AppointmentChangeVia] to String,
/// and [decode] dynamic data back to [AppointmentChangeVia].
class AppointmentChangeViaTypeTransformer {
  factory AppointmentChangeViaTypeTransformer() =>
      _instance ??= const AppointmentChangeViaTypeTransformer._();

  const AppointmentChangeViaTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(AppointmentChangeVia data) => data._value;

  /// Returns the instance of [AppointmentChangeVia] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  AppointmentChangeVia? decode(dynamic data, {bool allowNull = true}) {
    if (data is AppointmentChangeVia) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'app':
          return AppointmentChangeVia.app;
        case r'link':
          return AppointmentChangeVia.link;
        case r'staff':
          return AppointmentChangeVia.staff;
        case r'system':
          return AppointmentChangeVia.system;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static AppointmentChangeViaTypeTransformer? _instance;
}
