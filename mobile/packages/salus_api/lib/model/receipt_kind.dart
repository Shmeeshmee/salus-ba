//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

/// A receipt to a person, or an invoice to a company.
enum ReceiptKind {
  receipt._(r'receipt'),
  invoice._(r'invoice'),
  ;

  /// Instantiate a new enum with the provided value.
  const ReceiptKind._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ReceiptKind] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ReceiptKind? fromJson(dynamic value) =>
      ReceiptKindTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ReceiptKind]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ReceiptKind> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ReceiptKind>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ReceiptKind.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ReceiptKind] to String,
/// and [decode] dynamic data back to [ReceiptKind].
class ReceiptKindTypeTransformer {
  factory ReceiptKindTypeTransformer() =>
      _instance ??= const ReceiptKindTypeTransformer._();

  const ReceiptKindTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(ReceiptKind data) => data._value;

  /// Returns the instance of [ReceiptKind] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ReceiptKind? decode(dynamic data, {bool allowNull = true}) {
    if (data is ReceiptKind) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'receipt':
          return ReceiptKind.receipt;
        case r'invoice':
          return ReceiptKind.invoice;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ReceiptKindTypeTransformer? _instance;
}
