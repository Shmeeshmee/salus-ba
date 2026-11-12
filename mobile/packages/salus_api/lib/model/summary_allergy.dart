//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SummaryAllergy {
  /// Returns a new [SummaryAllergy] instance.
  SummaryAllergy({
    required this.reference,
    required this.text,
    this.reaction,
    this.severity,
    required this.confirmed,
  });

  /// AllergyIntolerance/id
  String reference;

  String text;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? reaction;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  AllergySeverity? severity;

  bool confirmed;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SummaryAllergy &&
          other.reference == reference &&
          other.text == text &&
          other.reaction == reaction &&
          other.severity == severity &&
          other.confirmed == confirmed;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (reference.hashCode) +
      (text.hashCode) +
      (reaction == null ? 0 : reaction!.hashCode) +
      (severity == null ? 0 : severity!.hashCode) +
      (confirmed.hashCode);

  @override
  String toString() =>
      'SummaryAllergy[reference=$reference, text=$text, reaction=$reaction, severity=$severity, confirmed=$confirmed]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'reference'] = this.reference;
    json[r'text'] = this.text;
    if (this.reaction != null) {
      json[r'reaction'] = this.reaction;
    } else {
      json[r'reaction'] = null;
    }
    if (this.severity != null) {
      json[r'severity'] = this.severity;
    } else {
      json[r'severity'] = null;
    }
    json[r'confirmed'] = this.confirmed;
    return json;
  }

  /// Returns a new [SummaryAllergy] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SummaryAllergy? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'reference'),
            'Required key "SummaryAllergy[reference]" is missing from JSON.');
        assert(json[r'reference'] != null,
            'Required key "SummaryAllergy[reference]" has a null value in JSON.');
        assert(json.containsKey(r'text'),
            'Required key "SummaryAllergy[text]" is missing from JSON.');
        assert(json[r'text'] != null,
            'Required key "SummaryAllergy[text]" has a null value in JSON.');
        assert(json.containsKey(r'confirmed'),
            'Required key "SummaryAllergy[confirmed]" is missing from JSON.');
        assert(json[r'confirmed'] != null,
            'Required key "SummaryAllergy[confirmed]" has a null value in JSON.');
        return true;
      }());

      return SummaryAllergy(
        reference: mapValueOfType<String>(json, r'reference')!,
        text: mapValueOfType<String>(json, r'text')!,
        reaction: mapValueOfType<String>(json, r'reaction'),
        severity: AllergySeverity.fromJson(json[r'severity']),
        confirmed: mapValueOfType<bool>(json, r'confirmed')!,
      );
    }
    return null;
  }

  static List<SummaryAllergy> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SummaryAllergy>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SummaryAllergy.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SummaryAllergy> mapFromJson(dynamic json) {
    final map = <String, SummaryAllergy>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SummaryAllergy.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SummaryAllergy-objects as value to a dart map
  static Map<String, List<SummaryAllergy>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SummaryAllergy>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SummaryAllergy.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'reference',
    'text',
    'confirmed',
  };
}
