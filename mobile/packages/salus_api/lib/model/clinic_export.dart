//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ClinicExport {
  /// Returns a new [ClinicExport] instance.
  ClinicExport({
    required this.id,
    required this.status,
    required this.requestedAt,
    this.expiresAt,
  });

  String id;

  ClinicExportStatus status;

  DateTime requestedAt;

  /// When a ready archive stops being downloadable.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? expiresAt;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ClinicExport &&
          other.id == id &&
          other.status == status &&
          other.requestedAt == requestedAt &&
          other.expiresAt == expiresAt;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (status.hashCode) +
      (requestedAt.hashCode) +
      (expiresAt == null ? 0 : expiresAt!.hashCode);

  @override
  String toString() =>
      'ClinicExport[id=$id, status=$status, requestedAt=$requestedAt, expiresAt=$expiresAt]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'status'] = this.status;
    json[r'requestedAt'] = this.requestedAt.toUtc().toIso8601String();
    if (this.expiresAt != null) {
      json[r'expiresAt'] = this.expiresAt!.toUtc().toIso8601String();
    } else {
      json[r'expiresAt'] = null;
    }
    return json;
  }

  /// Returns a new [ClinicExport] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ClinicExport? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "ClinicExport[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "ClinicExport[id]" has a null value in JSON.');
        assert(json.containsKey(r'status'),
            'Required key "ClinicExport[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "ClinicExport[status]" has a null value in JSON.');
        assert(json.containsKey(r'requestedAt'),
            'Required key "ClinicExport[requestedAt]" is missing from JSON.');
        assert(json[r'requestedAt'] != null,
            'Required key "ClinicExport[requestedAt]" has a null value in JSON.');
        return true;
      }());

      return ClinicExport(
        id: mapValueOfType<String>(json, r'id')!,
        status: ClinicExportStatus.fromJson(json[r'status'])!,
        requestedAt: mapDateTime(json, r'requestedAt', r'')!,
        expiresAt: mapDateTime(json, r'expiresAt', r''),
      );
    }
    return null;
  }

  static List<ClinicExport> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ClinicExport>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ClinicExport.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ClinicExport> mapFromJson(dynamic json) {
    final map = <String, ClinicExport>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ClinicExport.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ClinicExport-objects as value to a dart map
  static Map<String, List<ClinicExport>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ClinicExport>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ClinicExport.listFromJson(
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
    'status',
    'requestedAt',
  };
}
