//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class DeskVisit {
  /// Returns a new [DeskVisit] instance.
  DeskVisit({
    required this.appointment,
    required this.patient,
    this.serviceName,
    this.reminder,
    this.receipts = const [],
  });

  Appointment appointment;

  ClinicPatient patient;

  /// The booked service's name; absent for none.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? serviceName;

  /// Whether a reminder reaches the patient (docs/clinic-cms R3): always for a patient with an account (by the channels they chose); for a patient without one only while the clinic offers SMS and the profile has a phone.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  bool? reminder;

  /// The visit's receipts, whoever issued them, by number; reversals are not listed, a reversed receipt says so. Each is audited as a read.
  List<DeskReceipt> receipts;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DeskVisit &&
          other.appointment == appointment &&
          other.patient == patient &&
          other.serviceName == serviceName &&
          other.reminder == reminder &&
          _deepEquality.equals(other.receipts, receipts);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (appointment.hashCode) +
      (patient.hashCode) +
      (serviceName == null ? 0 : serviceName!.hashCode) +
      (reminder == null ? 0 : reminder!.hashCode) +
      (receipts.hashCode);

  @override
  String toString() =>
      'DeskVisit[appointment=$appointment, patient=$patient, serviceName=$serviceName, reminder=$reminder, receipts=$receipts]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'appointment'] = this.appointment;
    json[r'patient'] = this.patient;
    if (this.serviceName != null) {
      json[r'serviceName'] = this.serviceName;
    } else {
      json[r'serviceName'] = null;
    }
    if (this.reminder != null) {
      json[r'reminder'] = this.reminder;
    } else {
      json[r'reminder'] = null;
    }
    json[r'receipts'] = this.receipts;
    return json;
  }

  /// Returns a new [DeskVisit] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static DeskVisit? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'appointment'),
            'Required key "DeskVisit[appointment]" is missing from JSON.');
        assert(json[r'appointment'] != null,
            'Required key "DeskVisit[appointment]" has a null value in JSON.');
        assert(json.containsKey(r'patient'),
            'Required key "DeskVisit[patient]" is missing from JSON.');
        assert(json[r'patient'] != null,
            'Required key "DeskVisit[patient]" has a null value in JSON.');
        assert(json.containsKey(r'receipts'),
            'Required key "DeskVisit[receipts]" is missing from JSON.');
        assert(json[r'receipts'] != null,
            'Required key "DeskVisit[receipts]" has a null value in JSON.');
        return true;
      }());

      return DeskVisit(
        appointment: Appointment.fromJson(json[r'appointment'])!,
        patient: ClinicPatient.fromJson(json[r'patient'])!,
        serviceName: mapValueOfType<String>(json, r'serviceName'),
        reminder: mapValueOfType<bool>(json, r'reminder'),
        receipts: DeskReceipt.listFromJson(json[r'receipts']),
      );
    }
    return null;
  }

  static List<DeskVisit> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DeskVisit>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DeskVisit.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, DeskVisit> mapFromJson(dynamic json) {
    final map = <String, DeskVisit>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = DeskVisit.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of DeskVisit-objects as value to a dart map
  static Map<String, List<DeskVisit>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<DeskVisit>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = DeskVisit.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'appointment',
    'patient',
    'receipts',
  };
}
