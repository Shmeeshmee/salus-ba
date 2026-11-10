//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ReviewDecision {
  /// Returns a new [ReviewDecision] instance.
  ReviewDecision({
    required this.status,
    this.note,
  });

  ReviewDecisionStatusEnum status;

  /// Needed with needs_changes: what has to change.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? note;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ReviewDecision && other.status == status && other.note == note;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (status.hashCode) + (note == null ? 0 : note!.hashCode);

  @override
  String toString() => 'ReviewDecision[status=$status, note=$note]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'status'] = this.status;
    if (this.note != null) {
      json[r'note'] = this.note;
    } else {
      json[r'note'] = null;
    }
    return json;
  }

  /// Returns a new [ReviewDecision] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ReviewDecision? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'status'),
            'Required key "ReviewDecision[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "ReviewDecision[status]" has a null value in JSON.');
        return true;
      }());

      return ReviewDecision(
        status: ReviewDecisionStatusEnum.fromJson(json[r'status'])!,
        note: mapValueOfType<String>(json, r'note'),
      );
    }
    return null;
  }

  static List<ReviewDecision> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ReviewDecision>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ReviewDecision.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ReviewDecision> mapFromJson(dynamic json) {
    final map = <String, ReviewDecision>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ReviewDecision.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ReviewDecision-objects as value to a dart map
  static Map<String, List<ReviewDecision>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ReviewDecision>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ReviewDecision.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'status',
  };
}

enum ReviewDecisionStatusEnum {
  approved._(r'approved'),
  needsChanges._(r'needs_changes'),
  ;

  /// Instantiate a new enum with the provided value.
  const ReviewDecisionStatusEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ReviewDecisionStatusEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ReviewDecisionStatusEnum? fromJson(dynamic value) =>
      ReviewDecisionStatusEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ReviewDecisionStatusEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ReviewDecisionStatusEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ReviewDecisionStatusEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ReviewDecisionStatusEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ReviewDecisionStatusEnum] to String,
/// and [decode] dynamic data back to [ReviewDecisionStatusEnum].
class ReviewDecisionStatusEnumTypeTransformer {
  factory ReviewDecisionStatusEnumTypeTransformer() =>
      _instance ??= const ReviewDecisionStatusEnumTypeTransformer._();

  const ReviewDecisionStatusEnumTypeTransformer._();

  String encode(ReviewDecisionStatusEnum data) => data._value;

  /// Returns the instance of [ReviewDecisionStatusEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ReviewDecisionStatusEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ReviewDecisionStatusEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'approved':
          return ReviewDecisionStatusEnum.approved;
        case r'needs_changes':
          return ReviewDecisionStatusEnum.needsChanges;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ReviewDecisionStatusEnumTypeTransformer? _instance;
}
