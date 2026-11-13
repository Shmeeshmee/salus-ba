//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SummaryRequest {
  /// Returns a new [SummaryRequest] instance.
  SummaryRequest({
    this.lang,
  });

  /// The language to write the summary in.
  SummaryRequestLangEnum? lang;

  @override
  bool operator ==(Object other) =>
      identical(this, other) || other is SummaryRequest && other.lang == lang;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (lang == null ? 0 : lang!.hashCode);

  @override
  String toString() => 'SummaryRequest[lang=$lang]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    if (this.lang != null) {
      json[r'lang'] = this.lang;
    } else {
      json[r'lang'] = null;
    }
    return json;
  }

  /// Returns a new [SummaryRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SummaryRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        return true;
      }());

      return SummaryRequest(
        lang: SummaryRequestLangEnum.fromJson(json[r'lang']),
      );
    }
    return null;
  }

  static List<SummaryRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SummaryRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SummaryRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SummaryRequest> mapFromJson(dynamic json) {
    final map = <String, SummaryRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SummaryRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SummaryRequest-objects as value to a dart map
  static Map<String, List<SummaryRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SummaryRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SummaryRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{};
}

/// The language to write the summary in.
enum SummaryRequestLangEnum {
  bs._(r'bs'),
  hr._(r'hr'),
  srLatn._(r'sr-Latn'),
  srCyrl._(r'sr-Cyrl'),
  en._(r'en'),
  ;

  /// Instantiate a new enum with the provided value.
  const SummaryRequestLangEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [SummaryRequestLangEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static SummaryRequestLangEnum? fromJson(dynamic value) =>
      SummaryRequestLangEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [SummaryRequestLangEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<SummaryRequestLangEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SummaryRequestLangEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SummaryRequestLangEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [SummaryRequestLangEnum] to String,
/// and [decode] dynamic data back to [SummaryRequestLangEnum].
class SummaryRequestLangEnumTypeTransformer {
  factory SummaryRequestLangEnumTypeTransformer() =>
      _instance ??= const SummaryRequestLangEnumTypeTransformer._();

  const SummaryRequestLangEnumTypeTransformer._();

  String encode(SummaryRequestLangEnum data) => data._value;

  /// Returns the instance of [SummaryRequestLangEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  SummaryRequestLangEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is SummaryRequestLangEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'bs':
          return SummaryRequestLangEnum.bs;
        case r'hr':
          return SummaryRequestLangEnum.hr;
        case r'sr-Latn':
          return SummaryRequestLangEnum.srLatn;
        case r'sr-Cyrl':
          return SummaryRequestLangEnum.srCyrl;
        case r'en':
          return SummaryRequestLangEnum.en;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static SummaryRequestLangEnumTypeTransformer? _instance;
}
