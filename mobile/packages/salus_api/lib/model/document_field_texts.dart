//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class DocumentFieldTexts {
  /// Returns a new [DocumentFieldTexts] instance.
  DocumentFieldTexts({
    required this.label,
    this.instruction,
  });

  String label;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? instruction;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DocumentFieldTexts &&
          other.label == label &&
          other.instruction == instruction;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (label.hashCode) + (instruction == null ? 0 : instruction!.hashCode);

  @override
  String toString() =>
      'DocumentFieldTexts[label=$label, instruction=$instruction]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'label'] = this.label;
    if (this.instruction != null) {
      json[r'instruction'] = this.instruction;
    } else {
      json[r'instruction'] = null;
    }
    return json;
  }

  /// Returns a new [DocumentFieldTexts] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static DocumentFieldTexts? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'label'),
            'Required key "DocumentFieldTexts[label]" is missing from JSON.');
        assert(json[r'label'] != null,
            'Required key "DocumentFieldTexts[label]" has a null value in JSON.');
        return true;
      }());

      return DocumentFieldTexts(
        label: mapValueOfType<String>(json, r'label')!,
        instruction: mapValueOfType<String>(json, r'instruction'),
      );
    }
    return null;
  }

  static List<DocumentFieldTexts> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DocumentFieldTexts>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DocumentFieldTexts.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, DocumentFieldTexts> mapFromJson(dynamic json) {
    final map = <String, DocumentFieldTexts>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = DocumentFieldTexts.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of DocumentFieldTexts-objects as value to a dart map
  static Map<String, List<DocumentFieldTexts>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<DocumentFieldTexts>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = DocumentFieldTexts.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'label',
  };
}
