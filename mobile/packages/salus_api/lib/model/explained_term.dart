//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ExplainedTerm {
  /// Returns a new [ExplainedTerm] instance.
  ExplainedTerm({
    required this.termId,
    required this.language,
    required this.text,
    required this.reviewStatus,
  });

  String termId;

  /// The language of `text`; English when the viewer's language has no explanation of the term.
  TermLanguage language;

  String text;

  /// An explanation that needs changes is not given.
  ExplainedTermReviewStatusEnum reviewStatus;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ExplainedTerm &&
          other.termId == termId &&
          other.language == language &&
          other.text == text &&
          other.reviewStatus == reviewStatus;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (termId.hashCode) +
      (language.hashCode) +
      (text.hashCode) +
      (reviewStatus.hashCode);

  @override
  String toString() =>
      'ExplainedTerm[termId=$termId, language=$language, text=$text, reviewStatus=$reviewStatus]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'termId'] = this.termId;
    json[r'language'] = this.language;
    json[r'text'] = this.text;
    json[r'reviewStatus'] = this.reviewStatus;
    return json;
  }

  /// Returns a new [ExplainedTerm] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ExplainedTerm? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'termId'),
            'Required key "ExplainedTerm[termId]" is missing from JSON.');
        assert(json[r'termId'] != null,
            'Required key "ExplainedTerm[termId]" has a null value in JSON.');
        assert(json.containsKey(r'language'),
            'Required key "ExplainedTerm[language]" is missing from JSON.');
        assert(json[r'language'] != null,
            'Required key "ExplainedTerm[language]" has a null value in JSON.');
        assert(json.containsKey(r'text'),
            'Required key "ExplainedTerm[text]" is missing from JSON.');
        assert(json[r'text'] != null,
            'Required key "ExplainedTerm[text]" has a null value in JSON.');
        assert(json.containsKey(r'reviewStatus'),
            'Required key "ExplainedTerm[reviewStatus]" is missing from JSON.');
        assert(json[r'reviewStatus'] != null,
            'Required key "ExplainedTerm[reviewStatus]" has a null value in JSON.');
        return true;
      }());

      return ExplainedTerm(
        termId: mapValueOfType<String>(json, r'termId')!,
        language: TermLanguage.fromJson(json[r'language'])!,
        text: mapValueOfType<String>(json, r'text')!,
        reviewStatus:
            ExplainedTermReviewStatusEnum.fromJson(json[r'reviewStatus'])!,
      );
    }
    return null;
  }

  static List<ExplainedTerm> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ExplainedTerm>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ExplainedTerm.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ExplainedTerm> mapFromJson(dynamic json) {
    final map = <String, ExplainedTerm>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ExplainedTerm.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ExplainedTerm-objects as value to a dart map
  static Map<String, List<ExplainedTerm>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ExplainedTerm>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ExplainedTerm.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'termId',
    'language',
    'text',
    'reviewStatus',
  };
}

/// An explanation that needs changes is not given.
enum ExplainedTermReviewStatusEnum {
  unreviewed._(r'unreviewed'),
  approved._(r'approved'),
  ;

  /// Instantiate a new enum with the provided value.
  const ExplainedTermReviewStatusEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ExplainedTermReviewStatusEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ExplainedTermReviewStatusEnum? fromJson(dynamic value) =>
      ExplainedTermReviewStatusEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ExplainedTermReviewStatusEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ExplainedTermReviewStatusEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ExplainedTermReviewStatusEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ExplainedTermReviewStatusEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ExplainedTermReviewStatusEnum] to String,
/// and [decode] dynamic data back to [ExplainedTermReviewStatusEnum].
class ExplainedTermReviewStatusEnumTypeTransformer {
  factory ExplainedTermReviewStatusEnumTypeTransformer() =>
      _instance ??= const ExplainedTermReviewStatusEnumTypeTransformer._();

  const ExplainedTermReviewStatusEnumTypeTransformer._();

  String encode(ExplainedTermReviewStatusEnum data) => data._value;

  /// Returns the instance of [ExplainedTermReviewStatusEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ExplainedTermReviewStatusEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ExplainedTermReviewStatusEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'unreviewed':
          return ExplainedTermReviewStatusEnum.unreviewed;
        case r'approved':
          return ExplainedTermReviewStatusEnum.approved;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ExplainedTermReviewStatusEnumTypeTransformer? _instance;
}
