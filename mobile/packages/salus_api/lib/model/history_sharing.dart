//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class HistorySharing {
  /// Returns a new [HistorySharing] instance.
  HistorySharing({
    required this.organizationId,
    this.consentId,
    this.versionId,
    this.shared = const [],
  });

  String organizationId;

  /// The history consent (FHIR Consent); absent when nothing is shared.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? consentId;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? versionId;

  /// The encounters shared.
  List<String> shared;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is HistorySharing &&
          other.organizationId == organizationId &&
          other.consentId == consentId &&
          other.versionId == versionId &&
          _deepEquality.equals(other.shared, shared);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (organizationId.hashCode) +
      (consentId == null ? 0 : consentId!.hashCode) +
      (versionId == null ? 0 : versionId!.hashCode) +
      (shared.hashCode);

  @override
  String toString() =>
      'HistorySharing[organizationId=$organizationId, consentId=$consentId, versionId=$versionId, shared=$shared]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'organizationId'] = this.organizationId;
    if (this.consentId != null) {
      json[r'consentId'] = this.consentId;
    } else {
      json[r'consentId'] = null;
    }
    if (this.versionId != null) {
      json[r'versionId'] = this.versionId;
    } else {
      json[r'versionId'] = null;
    }
    json[r'shared'] = this.shared;
    return json;
  }

  /// Returns a new [HistorySharing] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static HistorySharing? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'organizationId'),
            'Required key "HistorySharing[organizationId]" is missing from JSON.');
        assert(json[r'organizationId'] != null,
            'Required key "HistorySharing[organizationId]" has a null value in JSON.');
        assert(json.containsKey(r'shared'),
            'Required key "HistorySharing[shared]" is missing from JSON.');
        assert(json[r'shared'] != null,
            'Required key "HistorySharing[shared]" has a null value in JSON.');
        return true;
      }());

      return HistorySharing(
        organizationId: mapValueOfType<String>(json, r'organizationId')!,
        consentId: mapValueOfType<String>(json, r'consentId'),
        versionId: mapValueOfType<String>(json, r'versionId'),
        shared: json[r'shared'] is Iterable
            ? (json[r'shared'] as Iterable)
                .cast<String>()
                .toList(growable: false)
            : const [],
      );
    }
    return null;
  }

  static List<HistorySharing> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <HistorySharing>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = HistorySharing.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, HistorySharing> mapFromJson(dynamic json) {
    final map = <String, HistorySharing>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = HistorySharing.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of HistorySharing-objects as value to a dart map
  static Map<String, List<HistorySharing>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<HistorySharing>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = HistorySharing.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'organizationId',
    'shared',
  };
}
