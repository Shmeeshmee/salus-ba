//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SummaryResponse {
  /// Returns a new [SummaryResponse] instance.
  SummaryResponse({
    required this.summary,
    required this.generatedAt,
    required this.upToSeq,
    required this.aiGenerated,
  });

  /// The summary's text; never a diagnosis.
  String summary;

  DateTime generatedAt;

  /// The number of the latest message it covers.
  int upToSeq;

  /// Always true; every client labels the summary AI generated.
  SummaryResponseAiGeneratedEnum aiGenerated;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SummaryResponse &&
          other.summary == summary &&
          other.generatedAt == generatedAt &&
          other.upToSeq == upToSeq &&
          other.aiGenerated == aiGenerated;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (summary.hashCode) +
      (generatedAt.hashCode) +
      (upToSeq.hashCode) +
      (aiGenerated.hashCode);

  @override
  String toString() =>
      'SummaryResponse[summary=$summary, generatedAt=$generatedAt, upToSeq=$upToSeq, aiGenerated=$aiGenerated]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'summary'] = this.summary;
    json[r'generated_at'] = this.generatedAt.toUtc().toIso8601String();
    json[r'up_to_seq'] = this.upToSeq;
    json[r'ai_generated'] = this.aiGenerated;
    return json;
  }

  /// Returns a new [SummaryResponse] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SummaryResponse? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'summary'),
            'Required key "SummaryResponse[summary]" is missing from JSON.');
        assert(json[r'summary'] != null,
            'Required key "SummaryResponse[summary]" has a null value in JSON.');
        assert(json.containsKey(r'generated_at'),
            'Required key "SummaryResponse[generated_at]" is missing from JSON.');
        assert(json[r'generated_at'] != null,
            'Required key "SummaryResponse[generated_at]" has a null value in JSON.');
        assert(json.containsKey(r'up_to_seq'),
            'Required key "SummaryResponse[up_to_seq]" is missing from JSON.');
        assert(json[r'up_to_seq'] != null,
            'Required key "SummaryResponse[up_to_seq]" has a null value in JSON.');
        assert(json.containsKey(r'ai_generated'),
            'Required key "SummaryResponse[ai_generated]" is missing from JSON.');
        assert(json[r'ai_generated'] != null,
            'Required key "SummaryResponse[ai_generated]" has a null value in JSON.');
        return true;
      }());

      return SummaryResponse(
        summary: mapValueOfType<String>(json, r'summary')!,
        generatedAt: mapDateTime(json, r'generated_at', r'')!,
        upToSeq: mapValueOfType<int>(json, r'up_to_seq')!,
        aiGenerated:
            SummaryResponseAiGeneratedEnum.fromJson(json[r'ai_generated'])!,
      );
    }
    return null;
  }

  static List<SummaryResponse> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SummaryResponse>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SummaryResponse.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SummaryResponse> mapFromJson(dynamic json) {
    final map = <String, SummaryResponse>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SummaryResponse.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SummaryResponse-objects as value to a dart map
  static Map<String, List<SummaryResponse>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SummaryResponse>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SummaryResponse.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'summary',
    'generated_at',
    'up_to_seq',
    'ai_generated',
  };
}

/// Always true; every client labels the summary AI generated.
enum SummaryResponseAiGeneratedEnum {
  true_._('true'),
  ;

  /// Instantiate a new enum with the provided value.
  const SummaryResponseAiGeneratedEnum._(this._value);

  /// The underlying value of this enum member.
  final bool _value;

  @override
  String toString() => _value.toString();

  /// Encodes this enum as a value suitable for JSON.
  bool toJson() => _value;

  /// Returns the instance of [SummaryResponseAiGeneratedEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static SummaryResponseAiGeneratedEnum? fromJson(dynamic value) =>
      SummaryResponseAiGeneratedEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [SummaryResponseAiGeneratedEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<SummaryResponseAiGeneratedEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SummaryResponseAiGeneratedEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SummaryResponseAiGeneratedEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [SummaryResponseAiGeneratedEnum] to bool,
/// and [decode] dynamic data back to [SummaryResponseAiGeneratedEnum].
class SummaryResponseAiGeneratedEnumTypeTransformer {
  factory SummaryResponseAiGeneratedEnumTypeTransformer() =>
      _instance ??= const SummaryResponseAiGeneratedEnumTypeTransformer._();

  const SummaryResponseAiGeneratedEnumTypeTransformer._();

  bool encode(SummaryResponseAiGeneratedEnum data) => data._value;

  /// Returns the instance of [SummaryResponseAiGeneratedEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  SummaryResponseAiGeneratedEnum? decode(dynamic data,
      {bool allowNull = true}) {
    if (data is SummaryResponseAiGeneratedEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case 'true':
          return SummaryResponseAiGeneratedEnum.true_;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static SummaryResponseAiGeneratedEnumTypeTransformer? _instance;
}
