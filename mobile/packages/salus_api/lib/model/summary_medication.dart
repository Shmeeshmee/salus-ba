//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SummaryMedication {
  /// Returns a new [SummaryMedication] instance.
  SummaryMedication({
    required this.dosage,
    required this.reference,
    required this.text,
    required this.confirmed,
  });

  Dosage dosage;

  /// MedicationStatement/id
  String reference;

  String text;

  bool confirmed;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SummaryMedication &&
          other.dosage == dosage &&
          other.reference == reference &&
          other.text == text &&
          other.confirmed == confirmed;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (dosage.hashCode) +
      (reference.hashCode) +
      (text.hashCode) +
      (confirmed.hashCode);

  @override
  String toString() =>
      'SummaryMedication[dosage=$dosage, reference=$reference, text=$text, confirmed=$confirmed]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'dosage'] = this.dosage;
    json[r'reference'] = this.reference;
    json[r'text'] = this.text;
    json[r'confirmed'] = this.confirmed;
    return json;
  }

  /// Returns a new [SummaryMedication] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SummaryMedication? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'dosage'),
            'Required key "SummaryMedication[dosage]" is missing from JSON.');
        assert(json[r'dosage'] != null,
            'Required key "SummaryMedication[dosage]" has a null value in JSON.');
        assert(json.containsKey(r'reference'),
            'Required key "SummaryMedication[reference]" is missing from JSON.');
        assert(json[r'reference'] != null,
            'Required key "SummaryMedication[reference]" has a null value in JSON.');
        assert(json.containsKey(r'text'),
            'Required key "SummaryMedication[text]" is missing from JSON.');
        assert(json[r'text'] != null,
            'Required key "SummaryMedication[text]" has a null value in JSON.');
        assert(json.containsKey(r'confirmed'),
            'Required key "SummaryMedication[confirmed]" is missing from JSON.');
        assert(json[r'confirmed'] != null,
            'Required key "SummaryMedication[confirmed]" has a null value in JSON.');
        return true;
      }());

      return SummaryMedication(
        dosage: Dosage.fromJson(json[r'dosage'])!,
        reference: mapValueOfType<String>(json, r'reference')!,
        text: mapValueOfType<String>(json, r'text')!,
        confirmed: mapValueOfType<bool>(json, r'confirmed')!,
      );
    }
    return null;
  }

  static List<SummaryMedication> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SummaryMedication>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SummaryMedication.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SummaryMedication> mapFromJson(dynamic json) {
    final map = <String, SummaryMedication>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SummaryMedication.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SummaryMedication-objects as value to a dart map
  static Map<String, List<SummaryMedication>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SummaryMedication>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SummaryMedication.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'dosage',
    'reference',
    'text',
    'confirmed',
  };
}
