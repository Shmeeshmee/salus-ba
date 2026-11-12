//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

/// What a history item is; its reference opens the FHIR resource. `communication` is reserved for chats (SAL-26) and not served yet.
enum HistoryItemType {
  visit._(r'visit'),
  document._(r'document'),
  questionnaireResponse._(r'questionnaire_response'),
  attachment._(r'attachment'),
  communication._(r'communication'),
  ;

  /// Instantiate a new enum with the provided value.
  const HistoryItemType._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [HistoryItemType] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static HistoryItemType? fromJson(dynamic value) =>
      HistoryItemTypeTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [HistoryItemType]
  /// that were successfully decoded from the passed [JSON][json].
  static List<HistoryItemType> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <HistoryItemType>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = HistoryItemType.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [HistoryItemType] to String,
/// and [decode] dynamic data back to [HistoryItemType].
class HistoryItemTypeTypeTransformer {
  factory HistoryItemTypeTypeTransformer() =>
      _instance ??= const HistoryItemTypeTypeTransformer._();

  const HistoryItemTypeTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(HistoryItemType data) => data._value;

  /// Returns the instance of [HistoryItemType] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  HistoryItemType? decode(dynamic data, {bool allowNull = true}) {
    if (data is HistoryItemType) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'visit':
          return HistoryItemType.visit;
        case r'document':
          return HistoryItemType.document;
        case r'questionnaire_response':
          return HistoryItemType.questionnaireResponse;
        case r'attachment':
          return HistoryItemType.attachment;
        case r'communication':
          return HistoryItemType.communication;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static HistoryItemTypeTypeTransformer? _instance;
}
