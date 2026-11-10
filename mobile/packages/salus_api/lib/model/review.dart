//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Review {
  /// Returns a new [Review] instance.
  Review({
    required this.status,
    this.reviewedBy,
    this.reviewedAt,
    this.note,
  });

  ReviewStatus status;

  /// The account that last set the status; absent when the curator's file set it or before the first review.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? reviewedBy;

  /// When it was last approved or marked as needing changes; kept when the entry goes back to unreviewed.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? reviewedAt;

  /// The reviewer's note, e.g. what needs to change.
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
      other is Review &&
          other.status == status &&
          other.reviewedBy == reviewedBy &&
          other.reviewedAt == reviewedAt &&
          other.note == note;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (status.hashCode) +
      (reviewedBy == null ? 0 : reviewedBy!.hashCode) +
      (reviewedAt == null ? 0 : reviewedAt!.hashCode) +
      (note == null ? 0 : note!.hashCode);

  @override
  String toString() =>
      'Review[status=$status, reviewedBy=$reviewedBy, reviewedAt=$reviewedAt, note=$note]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'status'] = this.status;
    if (this.reviewedBy != null) {
      json[r'reviewedBy'] = this.reviewedBy;
    } else {
      json[r'reviewedBy'] = null;
    }
    if (this.reviewedAt != null) {
      json[r'reviewedAt'] = this.reviewedAt!.toUtc().toIso8601String();
    } else {
      json[r'reviewedAt'] = null;
    }
    if (this.note != null) {
      json[r'note'] = this.note;
    } else {
      json[r'note'] = null;
    }
    return json;
  }

  /// Returns a new [Review] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Review? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'status'),
            'Required key "Review[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "Review[status]" has a null value in JSON.');
        return true;
      }());

      return Review(
        status: ReviewStatus.fromJson(json[r'status'])!,
        reviewedBy: mapValueOfType<String>(json, r'reviewedBy'),
        reviewedAt: mapDateTime(json, r'reviewedAt', r''),
        note: mapValueOfType<String>(json, r'note'),
      );
    }
    return null;
  }

  static List<Review> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Review>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Review.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Review> mapFromJson(dynamic json) {
    final map = <String, Review>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Review.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Review-objects as value to a dart map
  static Map<String, List<Review>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Review>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Review.listFromJson(
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
