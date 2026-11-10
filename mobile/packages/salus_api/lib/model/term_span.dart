//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class TermSpan {
  /// Returns a new [TermSpan] instance.
  TermSpan({
    required this.termId,
    required this.start,
    required this.end,
  });

  String termId;

  /// Minimum value: 0
  int start;

  /// Minimum value: 1
  int end;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is TermSpan &&
          other.termId == termId &&
          other.start == start &&
          other.end == end;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (termId.hashCode) + (start.hashCode) + (end.hashCode);

  @override
  String toString() => 'TermSpan[termId=$termId, start=$start, end=$end]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'term_id'] = this.termId;
    json[r'start'] = this.start;
    json[r'end'] = this.end;
    return json;
  }

  /// Returns a new [TermSpan] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static TermSpan? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'term_id'),
            'Required key "TermSpan[term_id]" is missing from JSON.');
        assert(json[r'term_id'] != null,
            'Required key "TermSpan[term_id]" has a null value in JSON.');
        assert(json.containsKey(r'start'),
            'Required key "TermSpan[start]" is missing from JSON.');
        assert(json[r'start'] != null,
            'Required key "TermSpan[start]" has a null value in JSON.');
        assert(json.containsKey(r'end'),
            'Required key "TermSpan[end]" is missing from JSON.');
        assert(json[r'end'] != null,
            'Required key "TermSpan[end]" has a null value in JSON.');
        return true;
      }());

      return TermSpan(
        termId: mapValueOfType<String>(json, r'term_id')!,
        start: mapValueOfType<int>(json, r'start')!,
        end: mapValueOfType<int>(json, r'end')!,
      );
    }
    return null;
  }

  static List<TermSpan> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <TermSpan>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = TermSpan.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, TermSpan> mapFromJson(dynamic json) {
    final map = <String, TermSpan>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = TermSpan.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of TermSpan-objects as value to a dart map
  static Map<String, List<TermSpan>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<TermSpan>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = TermSpan.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'term_id',
    'start',
    'end',
  };
}
