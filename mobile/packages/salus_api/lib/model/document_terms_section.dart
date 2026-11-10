//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class DocumentTermsSection {
  /// Returns a new [DocumentTermsSection] instance.
  DocumentTermsSection({
    required this.key,
    this.lines = const [],
  });

  /// The field's template key.
  String key;

  List<List<TermSpan>> lines;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DocumentTermsSection &&
          other.key == key &&
          _deepEquality.equals(other.lines, lines);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (key.hashCode) + (lines.hashCode);

  @override
  String toString() => 'DocumentTermsSection[key=$key, lines=$lines]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'key'] = this.key;
    json[r'lines'] = this.lines;
    return json;
  }

  /// Returns a new [DocumentTermsSection] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static DocumentTermsSection? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'key'),
            'Required key "DocumentTermsSection[key]" is missing from JSON.');
        assert(json[r'key'] != null,
            'Required key "DocumentTermsSection[key]" has a null value in JSON.');
        assert(json.containsKey(r'lines'),
            'Required key "DocumentTermsSection[lines]" is missing from JSON.');
        assert(json[r'lines'] != null,
            'Required key "DocumentTermsSection[lines]" has a null value in JSON.');
        return true;
      }());

      return DocumentTermsSection(
        key: mapValueOfType<String>(json, r'key')!,
        lines: json[r'lines'] is List
            ? (json[r'lines'] as List)
                .map((e) =>
                    e == null ? const <TermSpan>[] : TermSpan.listFromJson(e))
                .toList()
            : const [],
      );
    }
    return null;
  }

  static List<DocumentTermsSection> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DocumentTermsSection>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DocumentTermsSection.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, DocumentTermsSection> mapFromJson(dynamic json) {
    final map = <String, DocumentTermsSection>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = DocumentTermsSection.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of DocumentTermsSection-objects as value to a dart map
  static Map<String, List<DocumentTermsSection>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<DocumentTermsSection>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = DocumentTermsSection.listFromJson(
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
    'lines',
  };
}
