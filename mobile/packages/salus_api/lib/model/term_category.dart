//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

enum TermCategory {
  anatomy._(r'anatomy'),
  diagnosis._(r'diagnosis'),
  symptom._(r'symptom'),
  procedure._(r'procedure'),
  drug._(r'drug'),
  ;

  /// Instantiate a new enum with the provided value.
  const TermCategory._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [TermCategory] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static TermCategory? fromJson(dynamic value) =>
      TermCategoryTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [TermCategory]
  /// that were successfully decoded from the passed [JSON][json].
  static List<TermCategory> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <TermCategory>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = TermCategory.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [TermCategory] to String,
/// and [decode] dynamic data back to [TermCategory].
class TermCategoryTypeTransformer {
  factory TermCategoryTypeTransformer() =>
      _instance ??= const TermCategoryTypeTransformer._();

  const TermCategoryTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(TermCategory data) => data._value;

  /// Returns the instance of [TermCategory] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  TermCategory? decode(dynamic data, {bool allowNull = true}) {
    if (data is TermCategory) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'anatomy':
          return TermCategory.anatomy;
        case r'diagnosis':
          return TermCategory.diagnosis;
        case r'symptom':
          return TermCategory.symptom;
        case r'procedure':
          return TermCategory.procedure;
        case r'drug':
          return TermCategory.drug;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static TermCategoryTypeTransformer? _instance;
}
