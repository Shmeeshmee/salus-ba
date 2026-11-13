//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class WaitlistOffer {
  /// Returns a new [WaitlistOffer] instance.
  WaitlistOffer({
    required this.id,
    required this.slotStart,
    required this.slotEnd,
    required this.expiresAt,
  });

  String id;

  DateTime slotStart;

  DateTime slotEnd;

  DateTime expiresAt;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is WaitlistOffer &&
          other.id == id &&
          other.slotStart == slotStart &&
          other.slotEnd == slotEnd &&
          other.expiresAt == expiresAt;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (slotStart.hashCode) +
      (slotEnd.hashCode) +
      (expiresAt.hashCode);

  @override
  String toString() =>
      'WaitlistOffer[id=$id, slotStart=$slotStart, slotEnd=$slotEnd, expiresAt=$expiresAt]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'slot_start'] = this.slotStart.toUtc().toIso8601String();
    json[r'slot_end'] = this.slotEnd.toUtc().toIso8601String();
    json[r'expires_at'] = this.expiresAt.toUtc().toIso8601String();
    return json;
  }

  /// Returns a new [WaitlistOffer] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static WaitlistOffer? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "WaitlistOffer[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "WaitlistOffer[id]" has a null value in JSON.');
        assert(json.containsKey(r'slot_start'),
            'Required key "WaitlistOffer[slot_start]" is missing from JSON.');
        assert(json[r'slot_start'] != null,
            'Required key "WaitlistOffer[slot_start]" has a null value in JSON.');
        assert(json.containsKey(r'slot_end'),
            'Required key "WaitlistOffer[slot_end]" is missing from JSON.');
        assert(json[r'slot_end'] != null,
            'Required key "WaitlistOffer[slot_end]" has a null value in JSON.');
        assert(json.containsKey(r'expires_at'),
            'Required key "WaitlistOffer[expires_at]" is missing from JSON.');
        assert(json[r'expires_at'] != null,
            'Required key "WaitlistOffer[expires_at]" has a null value in JSON.');
        return true;
      }());

      return WaitlistOffer(
        id: mapValueOfType<String>(json, r'id')!,
        slotStart: mapDateTime(json, r'slot_start', r'')!,
        slotEnd: mapDateTime(json, r'slot_end', r'')!,
        expiresAt: mapDateTime(json, r'expires_at', r'')!,
      );
    }
    return null;
  }

  static List<WaitlistOffer> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <WaitlistOffer>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = WaitlistOffer.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, WaitlistOffer> mapFromJson(dynamic json) {
    final map = <String, WaitlistOffer>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = WaitlistOffer.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of WaitlistOffer-objects as value to a dart map
  static Map<String, List<WaitlistOffer>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<WaitlistOffer>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = WaitlistOffer.listFromJson(
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
    'slot_start',
    'slot_end',
    'expires_at',
  };
}
