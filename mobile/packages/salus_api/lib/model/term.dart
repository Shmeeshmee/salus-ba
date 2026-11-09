//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Term {
  /// Returns a new [Term] instance.
  Term({
    required this.id,
    required this.category,
    required this.reviewStatus,
    required this.codes,
    this.labels = const [],
  });

  String id;

  TermCategory category;

  /// `draft` until the curator has reviewed the term (the UI marks it \"Not reviewed yet\").
  TermReviewStatusEnum reviewStatus;

  TermCodes codes;

  /// One entry per requested language that has a label, in the order requested. A language with no label is absent: gaps are never filled by machine translation.
  List<TermLabels> labels;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Term &&
          other.id == id &&
          other.category == category &&
          other.reviewStatus == reviewStatus &&
          other.codes == codes &&
          _deepEquality.equals(other.labels, labels);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (category.hashCode) +
      (reviewStatus.hashCode) +
      (codes.hashCode) +
      (labels.hashCode);

  @override
  String toString() =>
      'Term[id=$id, category=$category, reviewStatus=$reviewStatus, codes=$codes, labels=$labels]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'category'] = this.category;
    json[r'reviewStatus'] = this.reviewStatus;
    json[r'codes'] = this.codes;
    json[r'labels'] = this.labels;
    return json;
  }

  /// Returns a new [Term] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Term? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "Term[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "Term[id]" has a null value in JSON.');
        assert(json.containsKey(r'category'),
            'Required key "Term[category]" is missing from JSON.');
        assert(json[r'category'] != null,
            'Required key "Term[category]" has a null value in JSON.');
        assert(json.containsKey(r'reviewStatus'),
            'Required key "Term[reviewStatus]" is missing from JSON.');
        assert(json[r'reviewStatus'] != null,
            'Required key "Term[reviewStatus]" has a null value in JSON.');
        assert(json.containsKey(r'codes'),
            'Required key "Term[codes]" is missing from JSON.');
        assert(json[r'codes'] != null,
            'Required key "Term[codes]" has a null value in JSON.');
        assert(json.containsKey(r'labels'),
            'Required key "Term[labels]" is missing from JSON.');
        assert(json[r'labels'] != null,
            'Required key "Term[labels]" has a null value in JSON.');
        return true;
      }());

      return Term(
        id: mapValueOfType<String>(json, r'id')!,
        category: TermCategory.fromJson(json[r'category'])!,
        reviewStatus: TermReviewStatusEnum.fromJson(json[r'reviewStatus'])!,
        codes: TermCodes.fromJson(json[r'codes'])!,
        labels: TermLabels.listFromJson(json[r'labels']),
      );
    }
    return null;
  }

  static List<Term> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Term>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Term.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Term> mapFromJson(dynamic json) {
    final map = <String, Term>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Term.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Term-objects as value to a dart map
  static Map<String, List<Term>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Term>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Term.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'id',
    'category',
    'reviewStatus',
    'codes',
    'labels',
  };
}

/// `draft` until the curator has reviewed the term (the UI marks it \"Not reviewed yet\").
enum TermReviewStatusEnum {
  draft._(r'draft'),
  reviewed._(r'reviewed'),
  ;

  /// Instantiate a new enum with the provided value.
  const TermReviewStatusEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [TermReviewStatusEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static TermReviewStatusEnum? fromJson(dynamic value) =>
      TermReviewStatusEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [TermReviewStatusEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<TermReviewStatusEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <TermReviewStatusEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = TermReviewStatusEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [TermReviewStatusEnum] to String,
/// and [decode] dynamic data back to [TermReviewStatusEnum].
class TermReviewStatusEnumTypeTransformer {
  factory TermReviewStatusEnumTypeTransformer() =>
      _instance ??= const TermReviewStatusEnumTypeTransformer._();

  const TermReviewStatusEnumTypeTransformer._();

  String encode(TermReviewStatusEnum data) => data._value;

  /// Returns the instance of [TermReviewStatusEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  TermReviewStatusEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is TermReviewStatusEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'draft':
          return TermReviewStatusEnum.draft;
        case r'reviewed':
          return TermReviewStatusEnum.reviewed;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static TermReviewStatusEnumTypeTransformer? _instance;
}
