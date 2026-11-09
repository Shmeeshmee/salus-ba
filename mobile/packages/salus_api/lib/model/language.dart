//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

/// A BCP 47 tag (docs/i18n/SPEC.md §3 \"Tags\"). Staff accounts and a clinic's default language take only the staff languages, the first five: bs, hr, sr-Latn, sr-Cyrl, en. Serbian is ijekavian in both scripts.
enum Language {
  bs._(r'bs'),
  hr._(r'hr'),
  srLatn._(r'sr-Latn'),
  srCyrl._(r'sr-Cyrl'),
  en._(r'en'),
  de._(r'de'),
  ar._(r'ar'),
  tr._(r'tr'),
  it._(r'it'),
  fr._(r'fr'),
  ;

  /// Instantiate a new enum with the provided value.
  const Language._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [Language] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static Language? fromJson(dynamic value) =>
      LanguageTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [Language]
  /// that were successfully decoded from the passed [JSON][json].
  static List<Language> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Language>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Language.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [Language] to String,
/// and [decode] dynamic data back to [Language].
class LanguageTypeTransformer {
  factory LanguageTypeTransformer() =>
      _instance ??= const LanguageTypeTransformer._();

  const LanguageTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(Language data) => data._value;

  /// Returns the instance of [Language] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  Language? decode(dynamic data, {bool allowNull = true}) {
    if (data is Language) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'bs':
          return Language.bs;
        case r'hr':
          return Language.hr;
        case r'sr-Latn':
          return Language.srLatn;
        case r'sr-Cyrl':
          return Language.srCyrl;
        case r'en':
          return Language.en;
        case r'de':
          return Language.de;
        case r'ar':
          return Language.ar;
        case r'tr':
          return Language.tr;
        case r'it':
          return Language.it;
        case r'fr':
          return Language.fr;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static LanguageTypeTransformer? _instance;
}
