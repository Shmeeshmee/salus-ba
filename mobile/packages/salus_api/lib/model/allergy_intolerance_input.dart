//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class AllergyIntoleranceInput {
  /// Returns a new [AllergyIntoleranceInput] instance.
  AllergyIntoleranceInput({
    this.termId,
    required this.text,
    this.category,
    this.reaction,
    this.severity,
    required this.clinicalStatus,
  });

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? termId;

  String text;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  AllergyCategory? category;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? reaction;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  AllergySeverity? severity;

  AllergyClinicalStatus clinicalStatus;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is AllergyIntoleranceInput &&
          other.termId == termId &&
          other.text == text &&
          other.category == category &&
          other.reaction == reaction &&
          other.severity == severity &&
          other.clinicalStatus == clinicalStatus;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (termId == null ? 0 : termId!.hashCode) +
      (text.hashCode) +
      (category == null ? 0 : category!.hashCode) +
      (reaction == null ? 0 : reaction!.hashCode) +
      (severity == null ? 0 : severity!.hashCode) +
      (clinicalStatus.hashCode);

  @override
  String toString() =>
      'AllergyIntoleranceInput[termId=$termId, text=$text, category=$category, reaction=$reaction, severity=$severity, clinicalStatus=$clinicalStatus]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    if (this.termId != null) {
      json[r'termId'] = this.termId;
    } else {
      json[r'termId'] = null;
    }
    json[r'text'] = this.text;
    if (this.category != null) {
      json[r'category'] = this.category;
    } else {
      json[r'category'] = null;
    }
    if (this.reaction != null) {
      json[r'reaction'] = this.reaction;
    } else {
      json[r'reaction'] = null;
    }
    if (this.severity != null) {
      json[r'severity'] = this.severity;
    } else {
      json[r'severity'] = null;
    }
    json[r'clinicalStatus'] = this.clinicalStatus;
    return json;
  }

  /// Returns a new [AllergyIntoleranceInput] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static AllergyIntoleranceInput? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'text'),
            'Required key "AllergyIntoleranceInput[text]" is missing from JSON.');
        assert(json[r'text'] != null,
            'Required key "AllergyIntoleranceInput[text]" has a null value in JSON.');
        assert(json.containsKey(r'clinicalStatus'),
            'Required key "AllergyIntoleranceInput[clinicalStatus]" is missing from JSON.');
        assert(json[r'clinicalStatus'] != null,
            'Required key "AllergyIntoleranceInput[clinicalStatus]" has a null value in JSON.');
        return true;
      }());

      return AllergyIntoleranceInput(
        termId: mapValueOfType<String>(json, r'termId'),
        text: mapValueOfType<String>(json, r'text')!,
        category: AllergyCategory.fromJson(json[r'category']),
        reaction: mapValueOfType<String>(json, r'reaction'),
        severity: AllergySeverity.fromJson(json[r'severity']),
        clinicalStatus:
            AllergyClinicalStatus.fromJson(json[r'clinicalStatus'])!,
      );
    }
    return null;
  }

  static List<AllergyIntoleranceInput> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <AllergyIntoleranceInput>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = AllergyIntoleranceInput.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, AllergyIntoleranceInput> mapFromJson(dynamic json) {
    final map = <String, AllergyIntoleranceInput>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = AllergyIntoleranceInput.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of AllergyIntoleranceInput-objects as value to a dart map
  static Map<String, List<AllergyIntoleranceInput>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<AllergyIntoleranceInput>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = AllergyIntoleranceInput.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'text',
    'clinicalStatus',
  };
}
