//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class DraftField {
  /// Returns a new [DraftField] instance.
  DraftField({
    required this.key,
    required this.value,
    this.sourceSegmentIds = const [],
    required this.missing,
  });

  String key;

  String value;

  List<String> sourceSegmentIds;

  bool missing;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DraftField &&
          other.key == key &&
          other.value == value &&
          _deepEquality.equals(other.sourceSegmentIds, sourceSegmentIds) &&
          other.missing == missing;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (key.hashCode) +
      (value.hashCode) +
      (sourceSegmentIds.hashCode) +
      (missing.hashCode);

  @override
  String toString() =>
      'DraftField[key=$key, value=$value, sourceSegmentIds=$sourceSegmentIds, missing=$missing]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'key'] = this.key;
    json[r'value'] = this.value;
    json[r'source_segment_ids'] = this.sourceSegmentIds;
    json[r'missing'] = this.missing;
    return json;
  }

  /// Returns a new [DraftField] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static DraftField? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'key'),
            'Required key "DraftField[key]" is missing from JSON.');
        assert(json[r'key'] != null,
            'Required key "DraftField[key]" has a null value in JSON.');
        assert(json.containsKey(r'value'),
            'Required key "DraftField[value]" is missing from JSON.');
        assert(json[r'value'] != null,
            'Required key "DraftField[value]" has a null value in JSON.');
        assert(json.containsKey(r'missing'),
            'Required key "DraftField[missing]" is missing from JSON.');
        assert(json[r'missing'] != null,
            'Required key "DraftField[missing]" has a null value in JSON.');
        return true;
      }());

      return DraftField(
        key: mapValueOfType<String>(json, r'key')!,
        value: mapValueOfType<String>(json, r'value')!,
        sourceSegmentIds: json[r'source_segment_ids'] is Iterable
            ? (json[r'source_segment_ids'] as Iterable)
                .cast<String>()
                .toList(growable: false)
            : const [],
        missing: mapValueOfType<bool>(json, r'missing')!,
      );
    }
    return null;
  }

  static List<DraftField> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DraftField>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DraftField.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, DraftField> mapFromJson(dynamic json) {
    final map = <String, DraftField>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = DraftField.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of DraftField-objects as value to a dart map
  static Map<String, List<DraftField>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<DraftField>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = DraftField.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'key',
    'value',
    'missing',
  };
}
