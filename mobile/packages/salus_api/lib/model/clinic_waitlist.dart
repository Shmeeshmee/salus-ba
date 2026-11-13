//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ClinicWaitlist {
  /// Returns a new [ClinicWaitlist] instance.
  ClinicWaitlist({
    this.nextOffset,
    this.entries = const [],
  });

  /// The next page's offset; absent after the last page.
  ///
  /// Minimum value: 0
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  int? nextOffset;

  List<ClinicWaitlistEntry> entries;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ClinicWaitlist &&
          other.nextOffset == nextOffset &&
          _deepEquality.equals(other.entries, entries);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (nextOffset == null ? 0 : nextOffset!.hashCode) + (entries.hashCode);

  @override
  String toString() =>
      'ClinicWaitlist[nextOffset=$nextOffset, entries=$entries]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    if (this.nextOffset != null) {
      json[r'next_offset'] = this.nextOffset;
    } else {
      json[r'next_offset'] = null;
    }
    json[r'entries'] = this.entries;
    return json;
  }

  /// Returns a new [ClinicWaitlist] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ClinicWaitlist? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'entries'),
            'Required key "ClinicWaitlist[entries]" is missing from JSON.');
        assert(json[r'entries'] != null,
            'Required key "ClinicWaitlist[entries]" has a null value in JSON.');
        return true;
      }());

      return ClinicWaitlist(
        nextOffset: mapValueOfType<int>(json, r'next_offset'),
        entries: ClinicWaitlistEntry.listFromJson(json[r'entries']),
      );
    }
    return null;
  }

  static List<ClinicWaitlist> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ClinicWaitlist>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ClinicWaitlist.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ClinicWaitlist> mapFromJson(dynamic json) {
    final map = <String, ClinicWaitlist>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ClinicWaitlist.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ClinicWaitlist-objects as value to a dart map
  static Map<String, List<ClinicWaitlist>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ClinicWaitlist>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ClinicWaitlist.listFromJson(
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
