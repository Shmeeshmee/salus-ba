//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class DataRequestChange {
  /// Returns a new [DataRequestChange] instance.
  DataRequestChange({
    required this.status,
    required this.changedAt,
  });

  DataRequestStatus status;

  DateTime changedAt;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DataRequestChange &&
          other.status == status &&
          other.changedAt == changedAt;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (status.hashCode) + (changedAt.hashCode);

  @override
  String toString() =>
      'DataRequestChange[status=$status, changedAt=$changedAt]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'status'] = this.status;
    json[r'changedAt'] = this.changedAt.toUtc().toIso8601String();
    return json;
  }

  /// Returns a new [DataRequestChange] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static DataRequestChange? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'status'),
            'Required key "DataRequestChange[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "DataRequestChange[status]" has a null value in JSON.');
        assert(json.containsKey(r'changedAt'),
            'Required key "DataRequestChange[changedAt]" is missing from JSON.');
        assert(json[r'changedAt'] != null,
            'Required key "DataRequestChange[changedAt]" has a null value in JSON.');
        return true;
      }());

      return DataRequestChange(
        status: DataRequestStatus.fromJson(json[r'status'])!,
        changedAt: mapDateTime(json, r'changedAt', r'')!,
      );
    }
    return null;
  }

  static List<DataRequestChange> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DataRequestChange>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DataRequestChange.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, DataRequestChange> mapFromJson(dynamic json) {
    final map = <String, DataRequestChange>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = DataRequestChange.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of DataRequestChange-objects as value to a dart map
  static Map<String, List<DataRequestChange>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<DataRequestChange>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = DataRequestChange.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'status',
    'changedAt',
  };
}
