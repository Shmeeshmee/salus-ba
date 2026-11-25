//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

/// http://hl7.org/fhir/diagnostic-report-status: `preliminary` attached and not released, `final` released to the patient (never changed after).
enum DiagnosticReportStatus {
  preliminary._(r'preliminary'),
  final_._(r'final'),
  ;

  /// Instantiate a new enum with the provided value.
  const DiagnosticReportStatus._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [DiagnosticReportStatus] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static DiagnosticReportStatus? fromJson(dynamic value) =>
      DiagnosticReportStatusTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [DiagnosticReportStatus]
  /// that were successfully decoded from the passed [JSON][json].
  static List<DiagnosticReportStatus> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DiagnosticReportStatus>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DiagnosticReportStatus.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [DiagnosticReportStatus] to String,
/// and [decode] dynamic data back to [DiagnosticReportStatus].
class DiagnosticReportStatusTypeTransformer {
  factory DiagnosticReportStatusTypeTransformer() =>
      _instance ??= const DiagnosticReportStatusTypeTransformer._();

  const DiagnosticReportStatusTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(DiagnosticReportStatus data) => data._value;

  /// Returns the instance of [DiagnosticReportStatus] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  DiagnosticReportStatus? decode(dynamic data, {bool allowNull = true}) {
    if (data is DiagnosticReportStatus) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'preliminary':
          return DiagnosticReportStatus.preliminary;
        case r'final':
          return DiagnosticReportStatus.final_;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static DiagnosticReportStatusTypeTransformer? _instance;
}
