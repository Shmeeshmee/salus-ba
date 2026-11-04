//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SaveAvailabilityRequest {
  /// Returns a new [SaveAvailabilityRequest] instance.
  SaveAvailabilityRequest({
    required this.slotMinutes,
    this.availability = const [],
  });

  /// Minimum value: 5
  /// Maximum value: 240
  int slotMinutes;

  List<AvailabilityRule> availability;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SaveAvailabilityRequest &&
          other.slotMinutes == slotMinutes &&
          _deepEquality.equals(other.availability, availability);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (slotMinutes.hashCode) + (availability.hashCode);

  @override
  String toString() =>
      'SaveAvailabilityRequest[slotMinutes=$slotMinutes, availability=$availability]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'slotMinutes'] = this.slotMinutes;
    json[r'availability'] = this.availability;
    return json;
  }

  /// Returns a new [SaveAvailabilityRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SaveAvailabilityRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'slotMinutes'),
            'Required key "SaveAvailabilityRequest[slotMinutes]" is missing from JSON.');
        assert(json[r'slotMinutes'] != null,
            'Required key "SaveAvailabilityRequest[slotMinutes]" has a null value in JSON.');
        assert(json.containsKey(r'availability'),
            'Required key "SaveAvailabilityRequest[availability]" is missing from JSON.');
        assert(json[r'availability'] != null,
            'Required key "SaveAvailabilityRequest[availability]" has a null value in JSON.');
        return true;
      }());

      return SaveAvailabilityRequest(
        slotMinutes: mapValueOfType<int>(json, r'slotMinutes')!,
        availability: AvailabilityRule.listFromJson(json[r'availability']),
      );
    }
    return null;
  }

  static List<SaveAvailabilityRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SaveAvailabilityRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SaveAvailabilityRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SaveAvailabilityRequest> mapFromJson(dynamic json) {
    final map = <String, SaveAvailabilityRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SaveAvailabilityRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SaveAvailabilityRequest-objects as value to a dart map
  static Map<String, List<SaveAvailabilityRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SaveAvailabilityRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SaveAvailabilityRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'slotMinutes',
    'availability',
  };
}
