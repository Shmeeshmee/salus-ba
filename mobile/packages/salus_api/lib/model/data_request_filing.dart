//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class DataRequestFiling {
  /// Returns a new [DataRequestFiling] instance.
  DataRequestFiling({
    required this.kind,
    required this.target,
    this.organizationId,
  });

  DataRequestKind kind;

  DataRequestTarget target;

  /// The clinic; required with the clinic target.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? organizationId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is DataRequestFiling &&
          other.kind == kind &&
          other.target == target &&
          other.organizationId == organizationId;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (kind.hashCode) +
      (target.hashCode) +
      (organizationId == null ? 0 : organizationId!.hashCode);

  @override
  String toString() =>
      'DataRequestFiling[kind=$kind, target=$target, organizationId=$organizationId]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'kind'] = this.kind;
    json[r'target'] = this.target;
    if (this.organizationId != null) {
      json[r'organizationId'] = this.organizationId;
    } else {
      json[r'organizationId'] = null;
    }
    return json;
  }

  /// Returns a new [DataRequestFiling] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static DataRequestFiling? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'kind'),
            'Required key "DataRequestFiling[kind]" is missing from JSON.');
        assert(json[r'kind'] != null,
            'Required key "DataRequestFiling[kind]" has a null value in JSON.');
        assert(json.containsKey(r'target'),
            'Required key "DataRequestFiling[target]" is missing from JSON.');
        assert(json[r'target'] != null,
            'Required key "DataRequestFiling[target]" has a null value in JSON.');
        return true;
      }());

      return DataRequestFiling(
        kind: DataRequestKind.fromJson(json[r'kind'])!,
        target: DataRequestTarget.fromJson(json[r'target'])!,
        organizationId: mapValueOfType<String>(json, r'organizationId'),
      );
    }
    return null;
  }

  static List<DataRequestFiling> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <DataRequestFiling>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = DataRequestFiling.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, DataRequestFiling> mapFromJson(dynamic json) {
    final map = <String, DataRequestFiling>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = DataRequestFiling.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of DataRequestFiling-objects as value to a dart map
  static Map<String, List<DataRequestFiling>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<DataRequestFiling>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = DataRequestFiling.listFromJson(
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
    'target',
  };
}
