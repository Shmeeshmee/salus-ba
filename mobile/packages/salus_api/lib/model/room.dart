//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Room {
  /// Returns a new [Room] instance.
  Room({
    required this.id,
    required this.locationId,
    required this.name,
    required this.active,
    required this.versionId,
  });

  String id;

  String locationId;

  String name;

  bool active;

  int versionId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Room &&
          other.id == id &&
          other.locationId == locationId &&
          other.name == name &&
          other.active == active &&
          other.versionId == versionId;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (locationId.hashCode) +
      (name.hashCode) +
      (active.hashCode) +
      (versionId.hashCode);

  @override
  String toString() =>
      'Room[id=$id, locationId=$locationId, name=$name, active=$active, versionId=$versionId]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'locationId'] = this.locationId;
    json[r'name'] = this.name;
    json[r'active'] = this.active;
    json[r'versionId'] = this.versionId;
    return json;
  }

  /// Returns a new [Room] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Room? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "Room[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "Room[id]" has a null value in JSON.');
        assert(json.containsKey(r'locationId'),
            'Required key "Room[locationId]" is missing from JSON.');
        assert(json[r'locationId'] != null,
            'Required key "Room[locationId]" has a null value in JSON.');
        assert(json.containsKey(r'name'),
            'Required key "Room[name]" is missing from JSON.');
        assert(json[r'name'] != null,
            'Required key "Room[name]" has a null value in JSON.');
        assert(json.containsKey(r'active'),
            'Required key "Room[active]" is missing from JSON.');
        assert(json[r'active'] != null,
            'Required key "Room[active]" has a null value in JSON.');
        assert(json.containsKey(r'versionId'),
            'Required key "Room[versionId]" is missing from JSON.');
        assert(json[r'versionId'] != null,
            'Required key "Room[versionId]" has a null value in JSON.');
        return true;
      }());

      return Room(
        id: mapValueOfType<String>(json, r'id')!,
        locationId: mapValueOfType<String>(json, r'locationId')!,
        name: mapValueOfType<String>(json, r'name')!,
        active: mapValueOfType<bool>(json, r'active')!,
        versionId: mapValueOfType<int>(json, r'versionId')!,
      );
    }
    return null;
  }

  static List<Room> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Room>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Room.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Room> mapFromJson(dynamic json) {
    final map = <String, Room>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Room.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Room-objects as value to a dart map
  static Map<String, List<Room>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Room>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Room.listFromJson(
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
    'locationId',
    'name',
    'active',
    'versionId',
  };
}
