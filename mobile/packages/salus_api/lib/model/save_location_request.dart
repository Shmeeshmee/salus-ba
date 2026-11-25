//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SaveLocationRequest {
  /// Returns a new [SaveLocationRequest] instance.
  SaveLocationRequest({
    required this.name,
    this.addressLine,
    this.postalCode,
    this.cityId,
    this.phone,
    this.timeZone,
    required this.active,
  });

  String name;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? addressLine;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? postalCode;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? cityId;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? phone;

  /// IANA; the clinic's when omitted.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? timeZone;

  bool active;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SaveLocationRequest &&
          other.name == name &&
          other.addressLine == addressLine &&
          other.postalCode == postalCode &&
          other.cityId == cityId &&
          other.phone == phone &&
          other.timeZone == timeZone &&
          other.active == active;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (name.hashCode) +
      (addressLine == null ? 0 : addressLine!.hashCode) +
      (postalCode == null ? 0 : postalCode!.hashCode) +
      (cityId == null ? 0 : cityId!.hashCode) +
      (phone == null ? 0 : phone!.hashCode) +
      (timeZone == null ? 0 : timeZone!.hashCode) +
      (active.hashCode);

  @override
  String toString() =>
      'SaveLocationRequest[name=$name, addressLine=$addressLine, postalCode=$postalCode, cityId=$cityId, phone=$phone, timeZone=$timeZone, active=$active]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
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
    if (this.timeZone != null) {
      json[r'timeZone'] = this.timeZone;
    } else {
      json[r'timeZone'] = null;
    }
    json[r'active'] = this.active;
    return json;
  }

  /// Returns a new [SaveLocationRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SaveLocationRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'name'),
            'Required key "SaveLocationRequest[name]" is missing from JSON.');
        assert(json[r'name'] != null,
            'Required key "SaveLocationRequest[name]" has a null value in JSON.');
        assert(json.containsKey(r'active'),
            'Required key "SaveLocationRequest[active]" is missing from JSON.');
        assert(json[r'active'] != null,
            'Required key "SaveLocationRequest[active]" has a null value in JSON.');
        return true;
      }());

      return SaveLocationRequest(
        name: mapValueOfType<String>(json, r'name')!,
        addressLine: mapValueOfType<String>(json, r'addressLine'),
        postalCode: mapValueOfType<String>(json, r'postalCode'),
        cityId: mapValueOfType<String>(json, r'cityId'),
        phone: mapValueOfType<String>(json, r'phone'),
        timeZone: mapValueOfType<String>(json, r'timeZone'),
        active: mapValueOfType<bool>(json, r'active')!,
      );
    }
    return null;
  }

  static List<SaveLocationRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SaveLocationRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SaveLocationRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SaveLocationRequest> mapFromJson(dynamic json) {
    final map = <String, SaveLocationRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SaveLocationRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SaveLocationRequest-objects as value to a dart map
  static Map<String, List<SaveLocationRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SaveLocationRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SaveLocationRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'name',
    'active',
  };
}
