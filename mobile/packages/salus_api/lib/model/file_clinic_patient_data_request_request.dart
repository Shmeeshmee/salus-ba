//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class FileClinicPatientDataRequestRequest {
  /// Returns a new [FileClinicPatientDataRequestRequest] instance.
  FileClinicPatientDataRequestRequest({
    required this.kind,
  });

  DataRequestKind kind;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is FileClinicPatientDataRequestRequest && other.kind == kind;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (kind.hashCode);

  @override
  String toString() => 'FileClinicPatientDataRequestRequest[kind=$kind]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'kind'] = this.kind;
    return json;
  }

  /// Returns a new [FileClinicPatientDataRequestRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static FileClinicPatientDataRequestRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'kind'),
            'Required key "FileClinicPatientDataRequestRequest[kind]" is missing from JSON.');
        assert(json[r'kind'] != null,
            'Required key "FileClinicPatientDataRequestRequest[kind]" has a null value in JSON.');
        return true;
      }());

      return FileClinicPatientDataRequestRequest(
        kind: DataRequestKind.fromJson(json[r'kind'])!,
      );
    }
    return null;
  }

  static List<FileClinicPatientDataRequestRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <FileClinicPatientDataRequestRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = FileClinicPatientDataRequestRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, FileClinicPatientDataRequestRequest> mapFromJson(
      dynamic json) {
    final map = <String, FileClinicPatientDataRequestRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = FileClinicPatientDataRequestRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of FileClinicPatientDataRequestRequest-objects as value to a dart map
  static Map<String, List<FileClinicPatientDataRequestRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<FileClinicPatientDataRequestRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = FileClinicPatientDataRequestRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'kind',
  };
}
