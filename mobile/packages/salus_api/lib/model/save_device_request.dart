//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SaveDeviceRequest {
  /// Returns a new [SaveDeviceRequest] instance.
  SaveDeviceRequest({
    required this.name,
    required this.active,
  });

  String name;

  bool active;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SaveDeviceRequest &&
          other.name == name &&
          other.active == active;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (name.hashCode) + (active.hashCode);

  @override
  String toString() => 'SaveDeviceRequest[name=$name, active=$active]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'name'] = this.name;
    json[r'active'] = this.active;
    return json;
  }

  /// Returns a new [SaveDeviceRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SaveDeviceRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'name'),
            'Required key "SaveDeviceRequest[name]" is missing from JSON.');
        assert(json[r'name'] != null,
            'Required key "SaveDeviceRequest[name]" has a null value in JSON.');
        assert(json.containsKey(r'active'),
            'Required key "SaveDeviceRequest[active]" is missing from JSON.');
        assert(json[r'active'] != null,
            'Required key "SaveDeviceRequest[active]" has a null value in JSON.');
        return true;
      }());

      return SaveDeviceRequest(
        name: mapValueOfType<String>(json, r'name')!,
        active: mapValueOfType<bool>(json, r'active')!,
      );
    }
    return null;
  }

  static List<SaveDeviceRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SaveDeviceRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SaveDeviceRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SaveDeviceRequest> mapFromJson(dynamic json) {
    final map = <String, SaveDeviceRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SaveDeviceRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SaveDeviceRequest-objects as value to a dart map
  static Map<String, List<SaveDeviceRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SaveDeviceRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SaveDeviceRequest.listFromJson(
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
