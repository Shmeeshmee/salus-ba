//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class DataRequest {
  /// Returns a new [DataRequest] instance.
  DataRequest({
    required this.id,
    required this.kind,
    required this.target,
    this.organizationId,
    this.patientId,
    required this.status,
    required this.receivedAt,
    required this.dueAt,
    this.extendedUntil,
    this.extensionReason,
    this.answeredAt,
    this.answerCode,
    required this.filedByPatient,
    required this.version,
    this.history = const [],
  });

  String id;

  DataRequestKind kind;

  DataRequestTarget target;

  /// The clinic of a clinic target.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? organizationId;

  /// The patient's profile; on the clinic's reads.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? patientId;

  DataRequestStatus status;

  DateTime receivedAt;

  DateTime dueAt;

  /// The deadline once extended.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? extendedUntil;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DataRequestExtensionReason? extensionReason;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? answeredAt;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DataRequestAnswerCode? answerCode;

  /// False for a request the clinic filed at the desk for the patient.
  bool filedByPatient;

  /// Minimum value: 1
  int version;

  List<DataRequestChange> history;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DataRequest &&
          other.id == id &&
          other.kind == kind &&
          other.target == target &&
          other.organizationId == organizationId &&
          other.patientId == patientId &&
          other.status == status &&
          other.receivedAt == receivedAt &&
          other.dueAt == dueAt &&
          other.extendedUntil == extendedUntil &&
          other.extensionReason == extensionReason &&
          other.answeredAt == answeredAt &&
          other.answerCode == answerCode &&
          other.filedByPatient == filedByPatient &&
          other.version == version &&
          _deepEquality.equals(other.history, history);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (kind.hashCode) +
      (target.hashCode) +
      (organizationId == null ? 0 : organizationId!.hashCode) +
      (patientId == null ? 0 : patientId!.hashCode) +
      (status.hashCode) +
      (receivedAt.hashCode) +
      (dueAt.hashCode) +
      (extendedUntil == null ? 0 : extendedUntil!.hashCode) +
      (extensionReason == null ? 0 : extensionReason!.hashCode) +
      (answeredAt == null ? 0 : answeredAt!.hashCode) +
      (answerCode == null ? 0 : answerCode!.hashCode) +
      (filedByPatient.hashCode) +
      (version.hashCode) +
      (history.hashCode);

  @override
  String toString() =>
      'DataRequest[id=$id, kind=$kind, target=$target, organizationId=$organizationId, patientId=$patientId, status=$status, receivedAt=$receivedAt, dueAt=$dueAt, extendedUntil=$extendedUntil, extensionReason=$extensionReason, answeredAt=$answeredAt, answerCode=$answerCode, filedByPatient=$filedByPatient, version=$version, history=$history]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'kind'] = this.kind;
    json[r'target'] = this.target;
    if (this.organizationId != null) {
      json[r'organizationId'] = this.organizationId;
    } else {
      json[r'organizationId'] = null;
    }
    if (this.patientId != null) {
      json[r'patientId'] = this.patientId;
    } else {
      json[r'patientId'] = null;
    }
    json[r'status'] = this.status;
    json[r'receivedAt'] = this.receivedAt.toUtc().toIso8601String();
    json[r'dueAt'] = this.dueAt.toUtc().toIso8601String();
    if (this.extendedUntil != null) {
      json[r'extendedUntil'] = this.extendedUntil!.toUtc().toIso8601String();
    } else {
      json[r'extendedUntil'] = null;
    }
    if (this.extensionReason != null) {
      json[r'extensionReason'] = this.extensionReason;
    } else {
      json[r'extensionReason'] = null;
    }
    if (this.answeredAt != null) {
      json[r'answeredAt'] = this.answeredAt!.toUtc().toIso8601String();
    } else {
      json[r'answeredAt'] = null;
    }
    if (this.answerCode != null) {
      json[r'answerCode'] = this.answerCode;
    } else {
      json[r'answerCode'] = null;
    }
    json[r'filedByPatient'] = this.filedByPatient;
    json[r'version'] = this.version;
    json[r'history'] = this.history;
    return json;
  }

  /// Returns a new [DataRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static DataRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "DataRequest[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "DataRequest[id]" has a null value in JSON.');
        assert(json.containsKey(r'kind'),
            'Required key "DataRequest[kind]" is missing from JSON.');
        assert(json[r'kind'] != null,
            'Required key "DataRequest[kind]" has a null value in JSON.');
        assert(json.containsKey(r'target'),
            'Required key "DataRequest[target]" is missing from JSON.');
        assert(json[r'target'] != null,
            'Required key "DataRequest[target]" has a null value in JSON.');
        assert(json.containsKey(r'status'),
            'Required key "DataRequest[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "DataRequest[status]" has a null value in JSON.');
        assert(json.containsKey(r'receivedAt'),
            'Required key "DataRequest[receivedAt]" is missing from JSON.');
        assert(json[r'receivedAt'] != null,
            'Required key "DataRequest[receivedAt]" has a null value in JSON.');
        assert(json.containsKey(r'dueAt'),
            'Required key "DataRequest[dueAt]" is missing from JSON.');
        assert(json[r'dueAt'] != null,
            'Required key "DataRequest[dueAt]" has a null value in JSON.');
        assert(json.containsKey(r'filedByPatient'),
            'Required key "DataRequest[filedByPatient]" is missing from JSON.');
        assert(json[r'filedByPatient'] != null,
            'Required key "DataRequest[filedByPatient]" has a null value in JSON.');
        assert(json.containsKey(r'version'),
            'Required key "DataRequest[version]" is missing from JSON.');
        assert(json[r'version'] != null,
            'Required key "DataRequest[version]" has a null value in JSON.');
        return true;
      }());

      return DataRequest(
        id: mapValueOfType<String>(json, r'id')!,
        kind: DataRequestKind.fromJson(json[r'kind'])!,
        target: DataRequestTarget.fromJson(json[r'target'])!,
        organizationId: mapValueOfType<String>(json, r'organizationId'),
        patientId: mapValueOfType<String>(json, r'patientId'),
        status: DataRequestStatus.fromJson(json[r'status'])!,
        receivedAt: mapDateTime(json, r'receivedAt', r'')!,
        dueAt: mapDateTime(json, r'dueAt', r'')!,
        extendedUntil: mapDateTime(json, r'extendedUntil', r''),
        extensionReason:
            DataRequestExtensionReason.fromJson(json[r'extensionReason']),
        answeredAt: mapDateTime(json, r'answeredAt', r''),
        answerCode: DataRequestAnswerCode.fromJson(json[r'answerCode']),
        filedByPatient: mapValueOfType<bool>(json, r'filedByPatient')!,
        version: mapValueOfType<int>(json, r'version')!,
        history: DataRequestChange.listFromJson(json[r'history']),
      );
    }
    return null;
  }

  static List<DataRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DataRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DataRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, DataRequest> mapFromJson(dynamic json) {
    final map = <String, DataRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = DataRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of DataRequest-objects as value to a dart map
  static Map<String, List<DataRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<DataRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = DataRequest.listFromJson(
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
    'kind',
    'target',
    'status',
    'receivedAt',
    'dueAt',
    'filedByPatient',
    'version',
  };
}
