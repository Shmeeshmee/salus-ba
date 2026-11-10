//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

/// Where a term, a translation or an explanation is in its review (docs/plans/v1-finish.md \"Contract\"): one status for all three. Editing an approved entry sets it back to `unreviewed`.
enum ReviewStatus {
  unreviewed._(r'unreviewed'),
  approved._(r'approved'),
  needsChanges._(r'needs_changes'),
  ;

  /// Instantiate a new enum with the provided value.
  const ReviewStatus._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ReviewStatus] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ReviewStatus? fromJson(dynamic value) =>
      ReviewStatusTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ReviewStatus]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ReviewStatus> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ReviewStatus>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ReviewStatus.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ReviewStatus] to String,
/// and [decode] dynamic data back to [ReviewStatus].
class ReviewStatusTypeTransformer {
  factory ReviewStatusTypeTransformer() =>
      _instance ??= const ReviewStatusTypeTransformer._();

  const ReviewStatusTypeTransformer._();

  /// Encodes this enum as a value suitable for JSON.
  String encode(ReviewStatus data) => data._value;

  /// Returns the instance of [ReviewStatus] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ReviewStatus? decode(dynamic data, {bool allowNull = true}) {
    if (data is ReviewStatus) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'unreviewed':
          return ReviewStatus.unreviewed;
        case r'approved':
          return ReviewStatus.approved;
        case r'needs_changes':
          return ReviewStatus.needsChanges;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ReviewStatusTypeTransformer? _instance;
}
