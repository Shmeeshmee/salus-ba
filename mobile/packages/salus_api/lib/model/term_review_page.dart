//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class TermReviewPage {
  /// Returns a new [TermReviewPage] instance.
  TermReviewPage({
    this.entries = const [],
    this.next,
  });

  List<TermReviewEntry> entries;

  /// Pass as `after` for the next page; absent on the last.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? next;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is TermReviewPage &&
          _deepEquality.equals(other.entries, entries) &&
          other.next == next;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (entries.hashCode) + (next == null ? 0 : next!.hashCode);

  @override
  String toString() => 'TermReviewPage[entries=$entries, next=$next]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'entries'] = this.entries;
    if (this.next != null) {
      json[r'next'] = this.next;
    } else {
      json[r'next'] = null;
    }
    return json;
  }

  /// Returns a new [TermReviewPage] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static TermReviewPage? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'entries'),
            'Required key "TermReviewPage[entries]" is missing from JSON.');
        assert(json[r'entries'] != null,
            'Required key "TermReviewPage[entries]" has a null value in JSON.');
        return true;
      }());

      return TermReviewPage(
        entries: TermReviewEntry.listFromJson(json[r'entries']),
        next: mapValueOfType<String>(json, r'next'),
      );
    }
    return null;
  }

  static List<TermReviewPage> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <TermReviewPage>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = TermReviewPage.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, TermReviewPage> mapFromJson(dynamic json) {
    final map = <String, TermReviewPage>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = TermReviewPage.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of TermReviewPage-objects as value to a dart map
  static Map<String, List<TermReviewPage>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<TermReviewPage>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = TermReviewPage.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'entries',
  };
}
