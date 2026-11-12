//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class HistoryItem {
  /// Returns a new [HistoryItem] instance.
  HistoryItem({
    required this.date,
    required this.clinic,
    required this.type,
    required this.reference,
    this.encounterId,
    this.specialty,
    this.title,
    this.diagnoses = const [],
    this.shared,
  });

  DateTime date;

  HistoryClinic clinic;

  HistoryItemType type;

  /// The FHIR resource to open, `Type/id`.
  String reference;

  /// The visit it belongs to.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? encounterId;

  /// The SNOMED CT code of the visit's practitioner role (`/specialties`).
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? specialty;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? title;

  /// A signed document's diagnoses: the ICD-10 codes (http://hl7.org/fhir/sid/icd-10) of its filled coded fields.
  List<String> diagnoses;

  /// For a practitioner, a visit of another clinic the patient shared, and everything of it.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  bool? shared;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is HistoryItem &&
          other.date == date &&
          other.clinic == clinic &&
          other.type == type &&
          other.reference == reference &&
          other.encounterId == encounterId &&
          other.specialty == specialty &&
          other.title == title &&
          _deepEquality.equals(other.diagnoses, diagnoses) &&
          other.shared == shared;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (date.hashCode) +
      (clinic.hashCode) +
      (type.hashCode) +
      (reference.hashCode) +
      (encounterId == null ? 0 : encounterId!.hashCode) +
      (specialty == null ? 0 : specialty!.hashCode) +
      (title == null ? 0 : title!.hashCode) +
      (diagnoses.hashCode) +
      (shared == null ? 0 : shared!.hashCode);

  @override
  String toString() =>
      'HistoryItem[date=$date, clinic=$clinic, type=$type, reference=$reference, encounterId=$encounterId, specialty=$specialty, title=$title, diagnoses=$diagnoses, shared=$shared]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'date'] = this.date.toUtc().toIso8601String();
    json[r'clinic'] = this.clinic;
    json[r'type'] = this.type;
    json[r'reference'] = this.reference;
    if (this.encounterId != null) {
      json[r'encounterId'] = this.encounterId;
    } else {
      json[r'encounterId'] = null;
    }
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
    json[r'diagnoses'] = this.diagnoses;
    if (this.shared != null) {
      json[r'shared'] = this.shared;
    } else {
      json[r'shared'] = null;
    }
    return json;
  }

  /// Returns a new [HistoryItem] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static HistoryItem? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'date'),
            'Required key "HistoryItem[date]" is missing from JSON.');
        assert(json[r'date'] != null,
            'Required key "HistoryItem[date]" has a null value in JSON.');
        assert(json.containsKey(r'clinic'),
            'Required key "HistoryItem[clinic]" is missing from JSON.');
        assert(json[r'clinic'] != null,
            'Required key "HistoryItem[clinic]" has a null value in JSON.');
        assert(json.containsKey(r'type'),
            'Required key "HistoryItem[type]" is missing from JSON.');
        assert(json[r'type'] != null,
            'Required key "HistoryItem[type]" has a null value in JSON.');
        assert(json.containsKey(r'reference'),
            'Required key "HistoryItem[reference]" is missing from JSON.');
        assert(json[r'reference'] != null,
            'Required key "HistoryItem[reference]" has a null value in JSON.');
        return true;
      }());

      return HistoryItem(
        date: mapDateTime(json, r'date', r'')!,
        clinic: HistoryClinic.fromJson(json[r'clinic'])!,
        type: HistoryItemType.fromJson(json[r'type'])!,
        reference: mapValueOfType<String>(json, r'reference')!,
        encounterId: mapValueOfType<String>(json, r'encounterId'),
        specialty: mapValueOfType<String>(json, r'specialty'),
        title: mapValueOfType<String>(json, r'title'),
        diagnoses: json[r'diagnoses'] is Iterable
            ? (json[r'diagnoses'] as Iterable)
                .cast<String>()
                .toList(growable: false)
            : const [],
        shared: mapValueOfType<bool>(json, r'shared'),
      );
    }
    return null;
  }

  static List<HistoryItem> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <HistoryItem>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = HistoryItem.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, HistoryItem> mapFromJson(dynamic json) {
    final map = <String, HistoryItem>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = HistoryItem.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of HistoryItem-objects as value to a dart map
  static Map<String, List<HistoryItem>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<HistoryItem>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = HistoryItem.listFromJson(
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
    'clinic',
    'type',
    'reference',
  };
}
