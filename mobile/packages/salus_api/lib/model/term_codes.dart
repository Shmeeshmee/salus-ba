//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class TermCodes {
  /// Returns a new [TermCodes] instance.
  TermCodes({
    this.wikidata,
    this.icd10,
    this.ta98,
    this.atc,
    this.pcs,
  });

  /// The Wikidata item, e.g. Q9368.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? wikidata;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? icd10;

  /// Terminologia Anatomica 98.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? ta98;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? atc;

  /// An ICD-10-PCS or ICD-9-CM procedure code.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? pcs;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is TermCodes &&
          other.wikidata == wikidata &&
          other.icd10 == icd10 &&
          other.ta98 == ta98 &&
          other.atc == atc &&
          other.pcs == pcs;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (wikidata == null ? 0 : wikidata!.hashCode) +
      (icd10 == null ? 0 : icd10!.hashCode) +
      (ta98 == null ? 0 : ta98!.hashCode) +
      (atc == null ? 0 : atc!.hashCode) +
      (pcs == null ? 0 : pcs!.hashCode);

  @override
  String toString() =>
      'TermCodes[wikidata=$wikidata, icd10=$icd10, ta98=$ta98, atc=$atc, pcs=$pcs]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    if (this.wikidata != null) {
      json[r'wikidata'] = this.wikidata;
    } else {
      json[r'wikidata'] = null;
    }
    if (this.icd10 != null) {
      json[r'icd10'] = this.icd10;
    } else {
      json[r'icd10'] = null;
    }
    if (this.ta98 != null) {
      json[r'ta98'] = this.ta98;
    } else {
      json[r'ta98'] = null;
    }
    if (this.atc != null) {
      json[r'atc'] = this.atc;
    } else {
      json[r'atc'] = null;
    }
    if (this.pcs != null) {
      json[r'pcs'] = this.pcs;
    } else {
      json[r'pcs'] = null;
    }
    return json;
  }

  /// Returns a new [TermCodes] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static TermCodes? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        return true;
      }());

      return TermCodes(
        wikidata: mapValueOfType<String>(json, r'wikidata'),
        icd10: mapValueOfType<String>(json, r'icd10'),
        ta98: mapValueOfType<String>(json, r'ta98'),
        atc: mapValueOfType<String>(json, r'atc'),
        pcs: mapValueOfType<String>(json, r'pcs'),
      );
    }
    return null;
  }

  static List<TermCodes> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <TermCodes>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = TermCodes.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, TermCodes> mapFromJson(dynamic json) {
    final map = <String, TermCodes>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = TermCodes.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of TermCodes-objects as value to a dart map
  static Map<String, List<TermCodes>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<TermCodes>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = TermCodes.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{};
}
