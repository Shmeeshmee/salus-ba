//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

enum AppointmentChangeKind {
  confirmed._(r'confirmed'),
  rescheduled._(r'rescheduled'),
  statusSet._(r'status_set'),
  statusCorrected._(r'status_corrected'),
  ;

  /// Instantiate a new enum with the provided value.
  const AppointmentChangeKind._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [AppointmentChangeKind] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static AppointmentChangeKind? fromJson(dynamic value) =>
      AppointmentChangeKindTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [AppointmentChangeKind]
  /// that were successfully decoded from the passed [JSON][json].
  static List<AppointmentChangeKind> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <AppointmentChangeKind>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = AppointmentChangeKind.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [AppointmentChangeKind] to String,
/// and [decode] dynamic data back to [AppointmentChangeKind].
class AppointmentChangeKindTypeTransformer {
  factory AppointmentChangeKindTypeTransformer() =>
      _instance ??= const AppointmentChangeKindTypeTransformer._();

  const AppointmentChangeKindTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(AppointmentChangeKind data) => data._value;

  /// Returns the instance of [AppointmentChangeKind] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  AppointmentChangeKind? decode(dynamic data, {bool allowNull = true}) {
    if (data is AppointmentChangeKind) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'confirmed':
          return AppointmentChangeKind.confirmed;
        case r'rescheduled':
          return AppointmentChangeKind.rescheduled;
        case r'status_set':
          return AppointmentChangeKind.statusSet;
        case r'status_corrected':
          return AppointmentChangeKind.statusCorrected;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static AppointmentChangeKindTypeTransformer? _instance;
}
