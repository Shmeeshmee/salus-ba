//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ClinicProfile {
  /// Returns a new [ClinicProfile] instance.
  ClinicProfile({
    this.languages = const {},
    required this.description,
    this.translations = const [],
    required this.welcomesInternational,
  });

  /// The languages the clinic's staff speak.
  Set<Language> languages;

  /// In the clinic's default language; empty for none.
  String description;

  List<ClinicProfileTranslation> translations;

  bool welcomesInternational;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ClinicProfile &&
          _deepEquality.equals(other.languages, languages) &&
          other.description == description &&
          _deepEquality.equals(other.translations, translations) &&
          other.welcomesInternational == welcomesInternational;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (languages.hashCode) +
      (description.hashCode) +
      (translations.hashCode) +
      (welcomesInternational.hashCode);

  @override
  String toString() =>
      'ClinicProfile[languages=$languages, description=$description, translations=$translations, welcomesInternational=$welcomesInternational]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'languages'] = this.languages.toList(growable: false);
    json[r'description'] = this.description;
    json[r'translations'] = this.translations;
    json[r'welcomesInternational'] = this.welcomesInternational;
    return json;
  }

  /// Returns a new [ClinicProfile] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ClinicProfile? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'languages'),
            'Required key "ClinicProfile[languages]" is missing from JSON.');
        assert(json[r'languages'] != null,
            'Required key "ClinicProfile[languages]" has a null value in JSON.');
        assert(json.containsKey(r'description'),
            'Required key "ClinicProfile[description]" is missing from JSON.');
        assert(json[r'description'] != null,
            'Required key "ClinicProfile[description]" has a null value in JSON.');
        assert(json.containsKey(r'translations'),
            'Required key "ClinicProfile[translations]" is missing from JSON.');
        assert(json[r'translations'] != null,
            'Required key "ClinicProfile[translations]" has a null value in JSON.');
        assert(json.containsKey(r'welcomesInternational'),
            'Required key "ClinicProfile[welcomesInternational]" is missing from JSON.');
        assert(json[r'welcomesInternational'] != null,
            'Required key "ClinicProfile[welcomesInternational]" has a null value in JSON.');
        return true;
      }());

      return ClinicProfile(
        languages: Language.listFromJson(json[r'languages']).toSet(),
        description: mapValueOfType<String>(json, r'description')!,
        translations:
            ClinicProfileTranslation.listFromJson(json[r'translations']),
        welcomesInternational:
            mapValueOfType<bool>(json, r'welcomesInternational')!,
      );
    }
    return null;
  }

  static List<ClinicProfile> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ClinicProfile>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ClinicProfile.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ClinicProfile> mapFromJson(dynamic json) {
    final map = <String, ClinicProfile>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ClinicProfile.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ClinicProfile-objects as value to a dart map
  static Map<String, List<ClinicProfile>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ClinicProfile>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ClinicProfile.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'languages',
    'description',
    'translations',
    'welcomesInternational',
  };
}
