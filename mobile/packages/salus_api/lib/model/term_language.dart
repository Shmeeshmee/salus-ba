//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

/// A label's language (docs/terminology/SPEC.md §3 \"Database\"), Latin then the ten app languages.
enum TermLanguage {
  la._(r'la'),
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
  const TermLanguage._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [TermLanguage] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static TermLanguage? fromJson(dynamic value) =>
      TermLanguageTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [TermLanguage]
  /// that were successfully decoded from the passed [JSON][json].
  static List<TermLanguage> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <TermLanguage>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = TermLanguage.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [TermLanguage] to String,
/// and [decode] dynamic data back to [TermLanguage].
class TermLanguageTypeTransformer {
  factory TermLanguageTypeTransformer() =>
      _instance ??= const TermLanguageTypeTransformer._();

  const TermLanguageTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(TermLanguage data) => data._value;

  /// Returns the instance of [TermLanguage] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  TermLanguage? decode(dynamic data, {bool allowNull = true}) {
    if (data is TermLanguage) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'la':
          return TermLanguage.la;
        case r'bs':
          return TermLanguage.bs;
        case r'hr':
          return TermLanguage.hr;
        case r'sr-Latn':
          return TermLanguage.srLatn;
        case r'sr-Cyrl':
          return TermLanguage.srCyrl;
        case r'en':
          return TermLanguage.en;
        case r'de':
          return TermLanguage.de;
        case r'ar':
          return TermLanguage.ar;
        case r'tr':
          return TermLanguage.tr;
        case r'it':
          return TermLanguage.it;
        case r'fr':
          return TermLanguage.fr;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static TermLanguageTypeTransformer? _instance;
}
