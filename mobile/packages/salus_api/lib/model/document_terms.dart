//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class DocumentTerms {
  /// Returns a new [DocumentTerms] instance.
  DocumentTerms({
    this.sections = const [],
  });

  /// One per field, in the Composition's section order.
  List<DocumentTermsSection> sections;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DocumentTerms && _deepEquality.equals(other.sections, sections);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (sections.hashCode);

  @override
  String toString() => 'DocumentTerms[sections=$sections]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'sections'] = this.sections;
    return json;
  }

  /// Returns a new [DocumentTerms] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static DocumentTerms? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'sections'),
            'Required key "DocumentTerms[sections]" is missing from JSON.');
        assert(json[r'sections'] != null,
            'Required key "DocumentTerms[sections]" has a null value in JSON.');
        return true;
      }());

      return DocumentTerms(
        sections: DocumentTermsSection.listFromJson(json[r'sections']),
      );
    }
    return null;
  }

  static List<DocumentTerms> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DocumentTerms>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DocumentTerms.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, DocumentTerms> mapFromJson(dynamic json) {
    final map = <String, DocumentTerms>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = DocumentTerms.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of DocumentTerms-objects as value to a dart map
  static Map<String, List<DocumentTerms>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<DocumentTerms>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = DocumentTerms.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'sections',
  };
}
