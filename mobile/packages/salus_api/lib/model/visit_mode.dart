//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

/// How the patient attends: at the clinic or by video (ADR 0016). In FHIR, the visit-mode extension of Slot and Appointment.
enum VisitMode {
  inPerson._(r'in-person'),
  video._(r'video'),
  ;

  /// Instantiate a new enum with the provided value.
  const VisitMode._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [VisitMode] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static VisitMode? fromJson(dynamic value) =>
      VisitModeTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [VisitMode]
  /// that were successfully decoded from the passed [JSON][json].
  static List<VisitMode> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <VisitMode>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = VisitMode.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [VisitMode] to String,
/// and [decode] dynamic data back to [VisitMode].
class VisitModeTypeTransformer {
  factory VisitModeTypeTransformer() =>
      _instance ??= const VisitModeTypeTransformer._();

  const VisitModeTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(VisitMode data) => data._value;

  /// Returns the instance of [VisitMode] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  VisitMode? decode(dynamic data, {bool allowNull = true}) {
    if (data is VisitMode) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'in-person':
          return VisitMode.inPerson;
        case r'video':
          return VisitMode.video;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static VisitModeTypeTransformer? _instance;
}
