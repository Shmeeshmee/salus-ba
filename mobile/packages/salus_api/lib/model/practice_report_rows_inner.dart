//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class PracticeReportRowsInner {
  /// Returns a new [PracticeReportRowsInner] instance.
  PracticeReportRowsInner({
    required this.practitionerRole,
    required this.visitMode,
    required this.minutesOffered,
    required this.visits,
    required this.receiptedMinor,
  });

  String practitionerRole;

  VisitMode visitMode;

  int minutesOffered;

  int visits;

  int receiptedMinor;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is PracticeReportRowsInner &&
          other.practitionerRole == practitionerRole &&
          other.visitMode == visitMode &&
          other.minutesOffered == minutesOffered &&
          other.visits == visits &&
          other.receiptedMinor == receiptedMinor;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (practitionerRole.hashCode) +
      (visitMode.hashCode) +
      (minutesOffered.hashCode) +
      (visits.hashCode) +
      (receiptedMinor.hashCode);

  @override
  String toString() =>
      'PracticeReportRowsInner[practitionerRole=$practitionerRole, visitMode=$visitMode, minutesOffered=$minutesOffered, visits=$visits, receiptedMinor=$receiptedMinor]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'practitionerRole'] = this.practitionerRole;
    json[r'visitMode'] = this.visitMode;
    json[r'minutesOffered'] = this.minutesOffered;
    json[r'visits'] = this.visits;
    json[r'receiptedMinor'] = this.receiptedMinor;
    return json;
  }

  /// Returns a new [PracticeReportRowsInner] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static PracticeReportRowsInner? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'practitionerRole'),
            'Required key "PracticeReportRowsInner[practitionerRole]" is missing from JSON.');
        assert(json[r'practitionerRole'] != null,
            'Required key "PracticeReportRowsInner[practitionerRole]" has a null value in JSON.');
        assert(json.containsKey(r'visitMode'),
            'Required key "PracticeReportRowsInner[visitMode]" is missing from JSON.');
        assert(json[r'visitMode'] != null,
            'Required key "PracticeReportRowsInner[visitMode]" has a null value in JSON.');
        assert(json.containsKey(r'minutesOffered'),
            'Required key "PracticeReportRowsInner[minutesOffered]" is missing from JSON.');
        assert(json[r'minutesOffered'] != null,
            'Required key "PracticeReportRowsInner[minutesOffered]" has a null value in JSON.');
        assert(json.containsKey(r'visits'),
            'Required key "PracticeReportRowsInner[visits]" is missing from JSON.');
        assert(json[r'visits'] != null,
            'Required key "PracticeReportRowsInner[visits]" has a null value in JSON.');
        assert(json.containsKey(r'receiptedMinor'),
            'Required key "PracticeReportRowsInner[receiptedMinor]" is missing from JSON.');
        assert(json[r'receiptedMinor'] != null,
            'Required key "PracticeReportRowsInner[receiptedMinor]" has a null value in JSON.');
        return true;
      }());

      return PracticeReportRowsInner(
        practitionerRole: mapValueOfType<String>(json, r'practitionerRole')!,
        visitMode: VisitMode.fromJson(json[r'visitMode'])!,
        minutesOffered: mapValueOfType<int>(json, r'minutesOffered')!,
        visits: mapValueOfType<int>(json, r'visits')!,
        receiptedMinor: mapValueOfType<int>(json, r'receiptedMinor')!,
      );
    }
    return null;
  }

  static List<PracticeReportRowsInner> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <PracticeReportRowsInner>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = PracticeReportRowsInner.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, PracticeReportRowsInner> mapFromJson(dynamic json) {
    final map = <String, PracticeReportRowsInner>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = PracticeReportRowsInner.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of PracticeReportRowsInner-objects as value to a dart map
  static Map<String, List<PracticeReportRowsInner>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<PracticeReportRowsInner>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = PracticeReportRowsInner.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'practitionerRole',
    'visitMode',
    'minutesOffered',
    'visits',
    'receiptedMinor',
  };
}
