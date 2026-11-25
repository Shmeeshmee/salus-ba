//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

/// http://terminology.hl7.org/CodeSystem/v3-ObservationInterpretation: normal, low, high, critically low, critically high, abnormal.
enum ObservationInterpretation {
  N._(r'N'),
  L._(r'L'),
  H._(r'H'),
  LL._(r'LL'),
  HH._(r'HH'),
  A._(r'A'),
  ;

  /// Instantiate a new enum with the provided value.
  const ObservationInterpretation._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ObservationInterpretation] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ObservationInterpretation? fromJson(dynamic value) =>
      ObservationInterpretationTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ObservationInterpretation]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ObservationInterpretation> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ObservationInterpretation>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ObservationInterpretation.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ObservationInterpretation] to String,
/// and [decode] dynamic data back to [ObservationInterpretation].
class ObservationInterpretationTypeTransformer {
  factory ObservationInterpretationTypeTransformer() =>
      _instance ??= const ObservationInterpretationTypeTransformer._();

  const ObservationInterpretationTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(ObservationInterpretation data) => data._value;

  /// Returns the instance of [ObservationInterpretation] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ObservationInterpretation? decode(dynamic data, {bool allowNull = true}) {
    if (data is ObservationInterpretation) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'N':
          return ObservationInterpretation.N;
        case r'L':
          return ObservationInterpretation.L;
        case r'H':
          return ObservationInterpretation.H;
        case r'LL':
          return ObservationInterpretation.LL;
        case r'HH':
          return ObservationInterpretation.HH;
        case r'A':
          return ObservationInterpretation.A;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ObservationInterpretationTypeTransformer? _instance;
}
