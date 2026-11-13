//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

/// What the consent is for: `care`, sharing the profile and appointments with the clinic; `transcription`, transcription and translation of the consultations with the clinic, with AI assistance for the practitioner (suggestions, document drafts), used only with an active care consent (SAL-12); `history`, sharing chosen visits at other clinics with the clinic (SAL-42), granted and changed only through `PUT /history-sharing/{organizationId}` (a grant of it here is `invalid_request`), listed and revoked here like the others.
enum ConsentKind {
  care._(r'care'),
  transcription._(r'transcription'),
  history._(r'history'),
  ;

  /// Instantiate a new enum with the provided value.
  const ConsentKind._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ConsentKind] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ConsentKind? fromJson(dynamic value) =>
      ConsentKindTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ConsentKind]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ConsentKind> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ConsentKind>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ConsentKind.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ConsentKind] to String,
/// and [decode] dynamic data back to [ConsentKind].
class ConsentKindTypeTransformer {
  factory ConsentKindTypeTransformer() =>
      _instance ??= const ConsentKindTypeTransformer._();

  const ConsentKindTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(ConsentKind data) => data._value;

  /// Returns the instance of [ConsentKind] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ConsentKind? decode(dynamic data, {bool allowNull = true}) {
    if (data is ConsentKind) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'care':
          return ConsentKind.care;
        case r'transcription':
          return ConsentKind.transcription;
        case r'history':
          return ConsentKind.history;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ConsentKindTypeTransformer? _instance;
}
