//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ClinicProfileTranslation {
  /// Returns a new [ClinicProfileTranslation] instance.
  ClinicProfileTranslation({
    required this.language,
    required this.description,
  });

  Language language;

  String description;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ClinicProfileTranslation &&
          other.language == language &&
          other.description == description;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (language.hashCode) + (description.hashCode);

  @override
  String toString() =>
      'ClinicProfileTranslation[language=$language, description=$description]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'language'] = this.language;
    json[r'description'] = this.description;
    return json;
  }

  /// Returns a new [ClinicProfileTranslation] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ClinicProfileTranslation? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'language'),
            'Required key "ClinicProfileTranslation[language]" is missing from JSON.');
        assert(json[r'language'] != null,
            'Required key "ClinicProfileTranslation[language]" has a null value in JSON.');
        assert(json.containsKey(r'description'),
            'Required key "ClinicProfileTranslation[description]" is missing from JSON.');
        assert(json[r'description'] != null,
            'Required key "ClinicProfileTranslation[description]" has a null value in JSON.');
        return true;
      }());

      return ClinicProfileTranslation(
        language: Language.fromJson(json[r'language'])!,
        description: mapValueOfType<String>(json, r'description')!,
      );
    }
    return null;
  }

  static List<ClinicProfileTranslation> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ClinicProfileTranslation>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ClinicProfileTranslation.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ClinicProfileTranslation> mapFromJson(dynamic json) {
    final map = <String, ClinicProfileTranslation>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ClinicProfileTranslation.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ClinicProfileTranslation-objects as value to a dart map
  static Map<String, List<ClinicProfileTranslation>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ClinicProfileTranslation>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ClinicProfileTranslation.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'language',
    'description',
  };
}
