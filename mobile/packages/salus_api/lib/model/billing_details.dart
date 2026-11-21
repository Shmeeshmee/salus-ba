//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class BillingDetails {
  /// Returns a new [BillingDetails] instance.
  BillingDetails({
    this.legalName,
    this.taxId,
    required this.vatRegistered,
    this.billingEmail,
  });

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? legalName;

  /// The JIB (13 digits) or IDB (12).
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? taxId;

  bool vatRegistered;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? billingEmail;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is BillingDetails &&
          other.legalName == legalName &&
          other.taxId == taxId &&
          other.vatRegistered == vatRegistered &&
          other.billingEmail == billingEmail;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (legalName == null ? 0 : legalName!.hashCode) +
      (taxId == null ? 0 : taxId!.hashCode) +
      (vatRegistered.hashCode) +
      (billingEmail == null ? 0 : billingEmail!.hashCode);

  @override
  String toString() =>
      'BillingDetails[legalName=$legalName, taxId=$taxId, vatRegistered=$vatRegistered, billingEmail=$billingEmail]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    if (this.legalName != null) {
      json[r'legalName'] = this.legalName;
    } else {
      json[r'legalName'] = null;
    }
    if (this.taxId != null) {
      json[r'taxId'] = this.taxId;
    } else {
      json[r'taxId'] = null;
    }
    json[r'vatRegistered'] = this.vatRegistered;
    if (this.billingEmail != null) {
      json[r'billingEmail'] = this.billingEmail;
    } else {
      json[r'billingEmail'] = null;
    }
    return json;
  }

  /// Returns a new [BillingDetails] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static BillingDetails? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'vatRegistered'),
            'Required key "BillingDetails[vatRegistered]" is missing from JSON.');
        assert(json[r'vatRegistered'] != null,
            'Required key "BillingDetails[vatRegistered]" has a null value in JSON.');
        return true;
      }());

      return BillingDetails(
        legalName: mapValueOfType<String>(json, r'legalName'),
        taxId: mapValueOfType<String>(json, r'taxId'),
        vatRegistered: mapValueOfType<bool>(json, r'vatRegistered')!,
        billingEmail: mapValueOfType<String>(json, r'billingEmail'),
      );
    }
    return null;
  }

  static List<BillingDetails> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <BillingDetails>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = BillingDetails.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, BillingDetails> mapFromJson(dynamic json) {
    final map = <String, BillingDetails>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = BillingDetails.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of BillingDetails-objects as value to a dart map
  static Map<String, List<BillingDetails>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<BillingDetails>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = BillingDetails.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'vatRegistered',
  };
}
