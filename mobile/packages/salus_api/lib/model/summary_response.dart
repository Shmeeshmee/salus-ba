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
  bool aiGenerated;

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
        aiGenerated: mapValueOfType<bool>(json, r'ai_generated')!,
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
