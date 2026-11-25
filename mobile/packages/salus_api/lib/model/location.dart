//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Location {
  /// Returns a new [Location] instance.
  Location({
    required this.id,
    required this.organizationId,
    required this.name,
    this.addressLine,
    this.postalCode,
    this.cityId,
    this.phone,
    required this.timeZone,
    required this.active,
    required this.versionId,
  });

  String id;

  String organizationId;

  String name;

  /// Absent for none.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? addressLine;

  /// Absent for none.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? postalCode;

  /// The city, a place's id (a Wikidata item); absent for none.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? cityId;

  /// E.164; absent for none.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? phone;

  /// IANA, the clinic's by default.
  String timeZone;

  bool active;

  int versionId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Location &&
          other.id == id &&
          other.organizationId == organizationId &&
          other.name == name &&
          other.addressLine == addressLine &&
          other.postalCode == postalCode &&
          other.cityId == cityId &&
          other.phone == phone &&
          other.timeZone == timeZone &&
          other.active == active &&
          other.versionId == versionId;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (organizationId.hashCode) +
      (name.hashCode) +
      (addressLine == null ? 0 : addressLine!.hashCode) +
      (postalCode == null ? 0 : postalCode!.hashCode) +
      (cityId == null ? 0 : cityId!.hashCode) +
      (phone == null ? 0 : phone!.hashCode) +
      (timeZone.hashCode) +
      (active.hashCode) +
      (versionId.hashCode);

  @override
  String toString() =>
      'Location[id=$id, organizationId=$organizationId, name=$name, addressLine=$addressLine, postalCode=$postalCode, cityId=$cityId, phone=$phone, timeZone=$timeZone, active=$active, versionId=$versionId]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'organizationId'] = this.organizationId;
    json[r'name'] = this.name;
    if (this.addressLine != null) {
      json[r'addressLine'] = this.addressLine;
    } else {
      json[r'addressLine'] = null;
    }
    if (this.postalCode != null) {
      json[r'postalCode'] = this.postalCode;
    } else {
      json[r'postalCode'] = null;
    }
    if (this.cityId != null) {
      json[r'cityId'] = this.cityId;
    } else {
      json[r'cityId'] = null;
    }
    if (this.phone != null) {
      json[r'phone'] = this.phone;
    } else {
      json[r'phone'] = null;
    }
    json[r'timeZone'] = this.timeZone;
    json[r'active'] = this.active;
    json[r'versionId'] = this.versionId;
    return json;
  }

  /// Returns a new [Location] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Location? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "Location[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "Location[id]" has a null value in JSON.');
        assert(json.containsKey(r'organizationId'),
            'Required key "Location[organizationId]" is missing from JSON.');
        assert(json[r'organizationId'] != null,
            'Required key "Location[organizationId]" has a null value in JSON.');
        assert(json.containsKey(r'name'),
            'Required key "Location[name]" is missing from JSON.');
        assert(json[r'name'] != null,
            'Required key "Location[name]" has a null value in JSON.');
        assert(json.containsKey(r'timeZone'),
            'Required key "Location[timeZone]" is missing from JSON.');
        assert(json[r'timeZone'] != null,
            'Required key "Location[timeZone]" has a null value in JSON.');
        assert(json.containsKey(r'active'),
            'Required key "Location[active]" is missing from JSON.');
        assert(json[r'active'] != null,
            'Required key "Location[active]" has a null value in JSON.');
        assert(json.containsKey(r'versionId'),
            'Required key "Location[versionId]" is missing from JSON.');
        assert(json[r'versionId'] != null,
            'Required key "Location[versionId]" has a null value in JSON.');
        return true;
      }());

      return Location(
        id: mapValueOfType<String>(json, r'id')!,
        organizationId: mapValueOfType<String>(json, r'organizationId')!,
        name: mapValueOfType<String>(json, r'name')!,
        addressLine: mapValueOfType<String>(json, r'addressLine'),
        postalCode: mapValueOfType<String>(json, r'postalCode'),
        cityId: mapValueOfType<String>(json, r'cityId'),
        phone: mapValueOfType<String>(json, r'phone'),
        timeZone: mapValueOfType<String>(json, r'timeZone')!,
        active: mapValueOfType<bool>(json, r'active')!,
        versionId: mapValueOfType<int>(json, r'versionId')!,
      );
    }
    return null;
  }

  static List<Location> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Location>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Location.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Location> mapFromJson(dynamic json) {
    final map = <String, Location>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Location.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Location-objects as value to a dart map
  static Map<String, List<Location>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Location>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Location.listFromJson(
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
    'organizationId',
    'name',
    'timeZone',
    'active',
    'versionId',
  };
}
