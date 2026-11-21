//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class OfferedService {
  /// Returns a new [OfferedService] instance.
  OfferedService({
    required this.id,
    required this.name,
    required this.priceMinor,
    required this.currency,
    this.durationMinutes,
  });

  String id;

  /// In the language asked for where translated.
  String name;

  /// Minimum value: 0
  int priceMinor;

  String currency;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  int? durationMinutes;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is OfferedService &&
          other.id == id &&
          other.name == name &&
          other.priceMinor == priceMinor &&
          other.currency == currency &&
          other.durationMinutes == durationMinutes;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (name.hashCode) +
      (priceMinor.hashCode) +
      (currency.hashCode) +
      (durationMinutes == null ? 0 : durationMinutes!.hashCode);

  @override
  String toString() =>
      'OfferedService[id=$id, name=$name, priceMinor=$priceMinor, currency=$currency, durationMinutes=$durationMinutes]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'name'] = this.name;
    json[r'priceMinor'] = this.priceMinor;
    json[r'currency'] = this.currency;
    if (this.durationMinutes != null) {
      json[r'durationMinutes'] = this.durationMinutes;
    } else {
      json[r'durationMinutes'] = null;
    }
    return json;
  }

  /// Returns a new [OfferedService] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static OfferedService? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "OfferedService[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "OfferedService[id]" has a null value in JSON.');
        assert(json.containsKey(r'name'),
            'Required key "OfferedService[name]" is missing from JSON.');
        assert(json[r'name'] != null,
            'Required key "OfferedService[name]" has a null value in JSON.');
        assert(json.containsKey(r'priceMinor'),
            'Required key "OfferedService[priceMinor]" is missing from JSON.');
        assert(json[r'priceMinor'] != null,
            'Required key "OfferedService[priceMinor]" has a null value in JSON.');
        assert(json.containsKey(r'currency'),
            'Required key "OfferedService[currency]" is missing from JSON.');
        assert(json[r'currency'] != null,
            'Required key "OfferedService[currency]" has a null value in JSON.');
        return true;
      }());

      return OfferedService(
        id: mapValueOfType<String>(json, r'id')!,
        name: mapValueOfType<String>(json, r'name')!,
        priceMinor: mapValueOfType<int>(json, r'priceMinor')!,
        currency: mapValueOfType<String>(json, r'currency')!,
        durationMinutes: mapValueOfType<int>(json, r'durationMinutes'),
      );
    }
    return null;
  }

  static List<OfferedService> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <OfferedService>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = OfferedService.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, OfferedService> mapFromJson(dynamic json) {
    final map = <String, OfferedService>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = OfferedService.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of OfferedService-objects as value to a dart map
  static Map<String, List<OfferedService>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<OfferedService>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = OfferedService.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'id',
    'name',
    'priceMinor',
    'currency',
  };
}
