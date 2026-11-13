//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SharingCandidate {
  /// Returns a new [SharingCandidate] instance.
  SharingCandidate({
    required this.date,
    required this.encounterId,
    required this.organizationId,
    required this.organizationName,
    required this.timeZone,
    this.specialty,
    this.title,
    required this.selected,
  });

  DateTime date;

  String encounterId;

  String organizationId;

  String organizationName;

  /// The visit's clinic's time zone (IANA), to show its date in.
  String timeZone;

  /// SNOMED CT code (`/specialties`).
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? specialty;

  /// The title of the visit's first signed document.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? title;

  /// Shared with the clinic now.
  bool selected;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SharingCandidate &&
          other.date == date &&
          other.encounterId == encounterId &&
          other.organizationId == organizationId &&
          other.organizationName == organizationName &&
          other.timeZone == timeZone &&
          other.specialty == specialty &&
          other.title == title &&
          other.selected == selected;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (date.hashCode) +
      (encounterId.hashCode) +
      (organizationId.hashCode) +
      (organizationName.hashCode) +
      (timeZone.hashCode) +
      (specialty == null ? 0 : specialty!.hashCode) +
      (title == null ? 0 : title!.hashCode) +
      (selected.hashCode);

  @override
  String toString() =>
      'SharingCandidate[date=$date, encounterId=$encounterId, organizationId=$organizationId, organizationName=$organizationName, timeZone=$timeZone, specialty=$specialty, title=$title, selected=$selected]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'date'] = this.date.toUtc().toIso8601String();
    json[r'encounterId'] = this.encounterId;
    json[r'organizationId'] = this.organizationId;
    json[r'organizationName'] = this.organizationName;
    json[r'timeZone'] = this.timeZone;
    if (this.specialty != null) {
      json[r'specialty'] = this.specialty;
    } else {
      json[r'specialty'] = null;
    }
    if (this.title != null) {
      json[r'title'] = this.title;
    } else {
      json[r'title'] = null;
    }
    json[r'selected'] = this.selected;
    return json;
  }

  /// Returns a new [SharingCandidate] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SharingCandidate? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'date'),
            'Required key "SharingCandidate[date]" is missing from JSON.');
        assert(json[r'date'] != null,
            'Required key "SharingCandidate[date]" has a null value in JSON.');
        assert(json.containsKey(r'encounterId'),
            'Required key "SharingCandidate[encounterId]" is missing from JSON.');
        assert(json[r'encounterId'] != null,
            'Required key "SharingCandidate[encounterId]" has a null value in JSON.');
        assert(json.containsKey(r'organizationId'),
            'Required key "SharingCandidate[organizationId]" is missing from JSON.');
        assert(json[r'organizationId'] != null,
            'Required key "SharingCandidate[organizationId]" has a null value in JSON.');
        assert(json.containsKey(r'organizationName'),
            'Required key "SharingCandidate[organizationName]" is missing from JSON.');
        assert(json[r'organizationName'] != null,
            'Required key "SharingCandidate[organizationName]" has a null value in JSON.');
        assert(json.containsKey(r'timeZone'),
            'Required key "SharingCandidate[timeZone]" is missing from JSON.');
        assert(json[r'timeZone'] != null,
            'Required key "SharingCandidate[timeZone]" has a null value in JSON.');
        assert(json.containsKey(r'selected'),
            'Required key "SharingCandidate[selected]" is missing from JSON.');
        assert(json[r'selected'] != null,
            'Required key "SharingCandidate[selected]" has a null value in JSON.');
        return true;
      }());

      return SharingCandidate(
        date: mapDateTime(json, r'date', r'')!,
        encounterId: mapValueOfType<String>(json, r'encounterId')!,
        organizationId: mapValueOfType<String>(json, r'organizationId')!,
        organizationName: mapValueOfType<String>(json, r'organizationName')!,
        timeZone: mapValueOfType<String>(json, r'timeZone')!,
        specialty: mapValueOfType<String>(json, r'specialty'),
        title: mapValueOfType<String>(json, r'title'),
        selected: mapValueOfType<bool>(json, r'selected')!,
      );
    }
    return null;
  }

  static List<SharingCandidate> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SharingCandidate>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SharingCandidate.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SharingCandidate> mapFromJson(dynamic json) {
    final map = <String, SharingCandidate>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SharingCandidate.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SharingCandidate-objects as value to a dart map
  static Map<String, List<SharingCandidate>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SharingCandidate>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SharingCandidate.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'date',
    'encounterId',
    'organizationId',
    'organizationName',
    'timeZone',
    'selected',
  };
}
