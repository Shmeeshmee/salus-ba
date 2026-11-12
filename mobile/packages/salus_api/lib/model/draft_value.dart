//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class DraftValue {
  /// Returns a new [DraftValue] instance.
  DraftValue({
    required this.key,
    required this.value,
    this.confirmed,
  });

  String key;

  String value;

  /// The practitioner confirmed the value of a coded field (an ICD-10 diagnosis), whether it was suggested by the drafting or typed by them. Required for a filled coded field; ignored for the other types.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  bool? confirmed;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DraftValue &&
          other.key == key &&
          other.value == value &&
          other.confirmed == confirmed;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (key.hashCode) +
      (value.hashCode) +
      (confirmed == null ? 0 : confirmed!.hashCode);

  @override
  String toString() =>
      'DraftValue[key=$key, value=$value, confirmed=$confirmed]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'key'] = this.key;
    json[r'value'] = this.value;
    if (this.confirmed != null) {
      json[r'confirmed'] = this.confirmed;
    } else {
      json[r'confirmed'] = null;
    }
    return json;
  }

  /// Returns a new [DraftValue] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static DraftValue? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'key'),
            'Required key "DraftValue[key]" is missing from JSON.');
        assert(json[r'key'] != null,
            'Required key "DraftValue[key]" has a null value in JSON.');
        assert(json.containsKey(r'value'),
            'Required key "DraftValue[value]" is missing from JSON.');
        assert(json[r'value'] != null,
            'Required key "DraftValue[value]" has a null value in JSON.');
        return true;
      }());

      return DraftValue(
        key: mapValueOfType<String>(json, r'key')!,
        value: mapValueOfType<String>(json, r'value')!,
        confirmed: mapValueOfType<bool>(json, r'confirmed'),
      );
    }
    return null;
  }

  static List<DraftValue> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DraftValue>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DraftValue.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, DraftValue> mapFromJson(dynamic json) {
    final map = <String, DraftValue>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = DraftValue.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of DraftValue-objects as value to a dart map
  static Map<String, List<DraftValue>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<DraftValue>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = DraftValue.listFromJson(
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
  };
}
