//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

/// `done` and `done_partly_kept` (done; what the medical-record retention keeps is kept) fulfil a request; the others refuse it.
enum DataRequestAnswerCode {
  done._(r'done'),
  donePartlyKept._(r'done_partly_kept'),
  identityUnverified._(r'identity_unverified'),
  manifestlyUnfounded._(r'manifestly_unfounded'),
  excessive._(r'excessive'),
  retentionLaw._(r'retention_law'),
  noData._(r'no_data'),
  ;

  /// Instantiate a new enum with the provided value.
  const DataRequestAnswerCode._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [DataRequestAnswerCode] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static DataRequestAnswerCode? fromJson(dynamic value) =>
      DataRequestAnswerCodeTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [DataRequestAnswerCode]
  /// that were successfully decoded from the passed [JSON][json].
  static List<DataRequestAnswerCode> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DataRequestAnswerCode>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DataRequestAnswerCode.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [DataRequestAnswerCode] to String,
/// and [decode] dynamic data back to [DataRequestAnswerCode].
class DataRequestAnswerCodeTypeTransformer {
  factory DataRequestAnswerCodeTypeTransformer() =>
      _instance ??= const DataRequestAnswerCodeTypeTransformer._();

  const DataRequestAnswerCodeTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(DataRequestAnswerCode data) => data._value;

  /// Returns the instance of [DataRequestAnswerCode] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  DataRequestAnswerCode? decode(dynamic data, {bool allowNull = true}) {
    if (data is DataRequestAnswerCode) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'done':
          return DataRequestAnswerCode.done;
        case r'done_partly_kept':
          return DataRequestAnswerCode.donePartlyKept;
        case r'identity_unverified':
          return DataRequestAnswerCode.identityUnverified;
        case r'manifestly_unfounded':
          return DataRequestAnswerCode.manifestlyUnfounded;
        case r'excessive':
          return DataRequestAnswerCode.excessive;
        case r'retention_law':
          return DataRequestAnswerCode.retentionLaw;
        case r'no_data':
          return DataRequestAnswerCode.noData;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static DataRequestAnswerCodeTypeTransformer? _instance;
}
