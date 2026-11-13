//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class HistorySummary {
  /// Returns a new [HistorySummary] instance.
  HistorySummary({
    this.lastVisit,
    this.allergies = const [],
    this.medications = const [],
    this.openItems = const [],
    this.allergiesUnavailable,
    this.medicationsUnavailable,
  });

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  HistoryItem? lastVisit;

  List<SummaryAllergy> allergies;

  List<SummaryMedication> medications;

  List<OpenItem> openItems;

  /// The allergies could not be read; the empty list means nothing is known, not none recorded.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  bool? allergiesUnavailable;

  /// The medications could not be read; the empty list means nothing is known, not none recorded.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  bool? medicationsUnavailable;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is HistorySummary &&
          other.lastVisit == lastVisit &&
          _deepEquality.equals(other.allergies, allergies) &&
          _deepEquality.equals(other.medications, medications) &&
          _deepEquality.equals(other.openItems, openItems) &&
          other.allergiesUnavailable == allergiesUnavailable &&
          other.medicationsUnavailable == medicationsUnavailable;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (lastVisit == null ? 0 : lastVisit!.hashCode) +
      (allergies.hashCode) +
      (medications.hashCode) +
      (openItems.hashCode) +
      (allergiesUnavailable == null ? 0 : allergiesUnavailable!.hashCode) +
      (medicationsUnavailable == null ? 0 : medicationsUnavailable!.hashCode);

  @override
  String toString() =>
      'HistorySummary[lastVisit=$lastVisit, allergies=$allergies, medications=$medications, openItems=$openItems, allergiesUnavailable=$allergiesUnavailable, medicationsUnavailable=$medicationsUnavailable]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    if (this.lastVisit != null) {
      json[r'lastVisit'] = this.lastVisit;
    } else {
      json[r'lastVisit'] = null;
    }
    json[r'allergies'] = this.allergies;
    json[r'medications'] = this.medications;
    json[r'openItems'] = this.openItems;
    if (this.allergiesUnavailable != null) {
      json[r'allergiesUnavailable'] = this.allergiesUnavailable;
    } else {
      json[r'allergiesUnavailable'] = null;
    }
    if (this.medicationsUnavailable != null) {
      json[r'medicationsUnavailable'] = this.medicationsUnavailable;
    } else {
      json[r'medicationsUnavailable'] = null;
    }
    return json;
  }

  /// Returns a new [HistorySummary] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static HistorySummary? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'allergies'),
            'Required key "HistorySummary[allergies]" is missing from JSON.');
        assert(json[r'allergies'] != null,
            'Required key "HistorySummary[allergies]" has a null value in JSON.');
        assert(json.containsKey(r'medications'),
            'Required key "HistorySummary[medications]" is missing from JSON.');
        assert(json[r'medications'] != null,
            'Required key "HistorySummary[medications]" has a null value in JSON.');
        assert(json.containsKey(r'openItems'),
            'Required key "HistorySummary[openItems]" is missing from JSON.');
        assert(json[r'openItems'] != null,
            'Required key "HistorySummary[openItems]" has a null value in JSON.');
        return true;
      }());

      return HistorySummary(
        lastVisit: HistoryItem.fromJson(json[r'lastVisit']),
        allergies: SummaryAllergy.listFromJson(json[r'allergies']),
        medications: SummaryMedication.listFromJson(json[r'medications']),
        openItems: OpenItem.listFromJson(json[r'openItems']),
        allergiesUnavailable:
            mapValueOfType<bool>(json, r'allergiesUnavailable'),
        medicationsUnavailable:
            mapValueOfType<bool>(json, r'medicationsUnavailable'),
      );
    }
    return null;
  }

  static List<HistorySummary> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <HistorySummary>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = HistorySummary.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, HistorySummary> mapFromJson(dynamic json) {
    final map = <String, HistorySummary>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = HistorySummary.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of HistorySummary-objects as value to a dart map
  static Map<String, List<HistorySummary>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<HistorySummary>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = HistorySummary.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'allergies',
    'medications',
    'openItems',
  };
}
