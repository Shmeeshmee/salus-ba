//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class WaitlistEntryList {
  /// Returns a new [WaitlistEntryList] instance.
  WaitlistEntryList({
    this.entries = const [],
  });

  List<WaitlistEntry> entries;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is WaitlistEntryList &&
          _deepEquality.equals(other.entries, entries);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (entries.hashCode);

  @override
  String toString() => 'WaitlistEntryList[entries=$entries]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'entries'] = this.entries;
    return json;
  }

  /// Returns a new [WaitlistEntryList] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static WaitlistEntryList? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'entries'),
            'Required key "WaitlistEntryList[entries]" is missing from JSON.');
        assert(json[r'entries'] != null,
            'Required key "WaitlistEntryList[entries]" has a null value in JSON.');
        return true;
      }());

      return WaitlistEntryList(
        entries: WaitlistEntry.listFromJson(json[r'entries']),
      );
    }
    return null;
  }

  static List<WaitlistEntryList> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <WaitlistEntryList>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = WaitlistEntryList.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, WaitlistEntryList> mapFromJson(dynamic json) {
    final map = <String, WaitlistEntryList>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = WaitlistEntryList.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of WaitlistEntryList-objects as value to a dart map
  static Map<String, List<WaitlistEntryList>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<WaitlistEntryList>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = WaitlistEntryList.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'entries',
  };
}
