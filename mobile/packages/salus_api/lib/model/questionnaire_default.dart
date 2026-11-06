//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class QuestionnaireDefault {
  /// Returns a new [QuestionnaireDefault] instance.
  QuestionnaireDefault({
    required this.practitionerRoleId,
    this.url,
    required this.versionId,
  });

  String practitionerRoleId;

  /// The canonical url of the default questionnaire; absent while the role has none.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? url;

  /// 0 while the role has none.
  int versionId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is QuestionnaireDefault &&
          other.practitionerRoleId == practitionerRoleId &&
          other.url == url &&
          other.versionId == versionId;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (practitionerRoleId.hashCode) +
      (url == null ? 0 : url!.hashCode) +
      (versionId.hashCode);

  @override
  String toString() =>
      'QuestionnaireDefault[practitionerRoleId=$practitionerRoleId, url=$url, versionId=$versionId]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'practitionerRoleId'] = this.practitionerRoleId;
    if (this.url != null) {
      json[r'url'] = this.url;
    } else {
      json[r'url'] = null;
    }
    json[r'versionId'] = this.versionId;
    return json;
  }

  /// Returns a new [QuestionnaireDefault] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static QuestionnaireDefault? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'practitionerRoleId'),
            'Required key "QuestionnaireDefault[practitionerRoleId]" is missing from JSON.');
        assert(json[r'practitionerRoleId'] != null,
            'Required key "QuestionnaireDefault[practitionerRoleId]" has a null value in JSON.');
        assert(json.containsKey(r'versionId'),
            'Required key "QuestionnaireDefault[versionId]" is missing from JSON.');
        assert(json[r'versionId'] != null,
            'Required key "QuestionnaireDefault[versionId]" has a null value in JSON.');
        return true;
      }());

      return QuestionnaireDefault(
        practitionerRoleId:
            mapValueOfType<String>(json, r'practitionerRoleId')!,
        url: mapValueOfType<String>(json, r'url'),
        versionId: mapValueOfType<int>(json, r'versionId')!,
      );
    }
    return null;
  }

  static List<QuestionnaireDefault> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <QuestionnaireDefault>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = QuestionnaireDefault.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, QuestionnaireDefault> mapFromJson(dynamic json) {
    final map = <String, QuestionnaireDefault>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = QuestionnaireDefault.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of QuestionnaireDefault-objects as value to a dart map
  static Map<String, List<QuestionnaireDefault>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<QuestionnaireDefault>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = QuestionnaireDefault.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'practitionerRoleId',
    'versionId',
  };
}
