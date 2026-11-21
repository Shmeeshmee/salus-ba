//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ServiceTranslation {
  /// Returns a new [ServiceTranslation] instance.
  ServiceTranslation({
    required this.language,
    required this.name,
  });

  /// A BCP 47 tag of the language enum.
  String language;

  String name;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ServiceTranslation &&
          other.language == language &&
          other.name == name;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (language.hashCode) + (name.hashCode);

  @override
  String toString() => 'ServiceTranslation[language=$language, name=$name]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'language'] = this.language;
    json[r'name'] = this.name;
    return json;
  }

  /// Returns a new [ServiceTranslation] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ServiceTranslation? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'language'),
            'Required key "ServiceTranslation[language]" is missing from JSON.');
        assert(json[r'language'] != null,
            'Required key "ServiceTranslation[language]" has a null value in JSON.');
        assert(json.containsKey(r'name'),
            'Required key "ServiceTranslation[name]" is missing from JSON.');
        assert(json[r'name'] != null,
            'Required key "ServiceTranslation[name]" has a null value in JSON.');
        return true;
      }());

      return ServiceTranslation(
        language: mapValueOfType<String>(json, r'language')!,
        name: mapValueOfType<String>(json, r'name')!,
      );
    }
    return null;
  }

  static List<ServiceTranslation> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ServiceTranslation>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ServiceTranslation.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ServiceTranslation> mapFromJson(dynamic json) {
    final map = <String, ServiceTranslation>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ServiceTranslation.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ServiceTranslation-objects as value to a dart map
  static Map<String, List<ServiceTranslation>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ServiceTranslation>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ServiceTranslation.listFromJson(
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
    'name',
  };
}
