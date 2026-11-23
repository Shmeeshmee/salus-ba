//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ReceiptLine {
  /// Returns a new [ReceiptLine] instance.
  ReceiptLine({
    this.serviceId,
    required this.name,
    required this.quantity,
    required this.unitPriceMinor,
    required this.discountMinor,
    this.vatRateBp,
    required this.totalMinor,
  });

  /// The price list's service; absent for an item off it.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? serviceId;

  /// As on the price list at issue. Lines are in the order they were entered.
  String name;

  /// Negative on a reversal.
  int quantity;

  int unitPriceMinor;

  int discountMinor;

  /// Absent for exempt.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  int? vatRateBp;

  int totalMinor;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ReceiptLine &&
          other.serviceId == serviceId &&
          other.name == name &&
          other.quantity == quantity &&
          other.unitPriceMinor == unitPriceMinor &&
          other.discountMinor == discountMinor &&
          other.vatRateBp == vatRateBp &&
          other.totalMinor == totalMinor;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (serviceId == null ? 0 : serviceId!.hashCode) +
      (name.hashCode) +
      (quantity.hashCode) +
      (unitPriceMinor.hashCode) +
      (discountMinor.hashCode) +
      (vatRateBp == null ? 0 : vatRateBp!.hashCode) +
      (totalMinor.hashCode);

  @override
  String toString() =>
      'ReceiptLine[serviceId=$serviceId, name=$name, quantity=$quantity, unitPriceMinor=$unitPriceMinor, discountMinor=$discountMinor, vatRateBp=$vatRateBp, totalMinor=$totalMinor]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    if (this.serviceId != null) {
      json[r'serviceId'] = this.serviceId;
    } else {
      json[r'serviceId'] = null;
    }
    json[r'name'] = this.name;
    json[r'quantity'] = this.quantity;
    json[r'unitPriceMinor'] = this.unitPriceMinor;
    json[r'discountMinor'] = this.discountMinor;
    if (this.vatRateBp != null) {
      json[r'vatRateBp'] = this.vatRateBp;
    } else {
      json[r'vatRateBp'] = null;
    }
    json[r'totalMinor'] = this.totalMinor;
    return json;
  }

  /// Returns a new [ReceiptLine] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ReceiptLine? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'name'),
            'Required key "ReceiptLine[name]" is missing from JSON.');
        assert(json[r'name'] != null,
            'Required key "ReceiptLine[name]" has a null value in JSON.');
        assert(json.containsKey(r'quantity'),
            'Required key "ReceiptLine[quantity]" is missing from JSON.');
        assert(json[r'quantity'] != null,
            'Required key "ReceiptLine[quantity]" has a null value in JSON.');
        assert(json.containsKey(r'unitPriceMinor'),
            'Required key "ReceiptLine[unitPriceMinor]" is missing from JSON.');
        assert(json[r'unitPriceMinor'] != null,
            'Required key "ReceiptLine[unitPriceMinor]" has a null value in JSON.');
        assert(json.containsKey(r'discountMinor'),
            'Required key "ReceiptLine[discountMinor]" is missing from JSON.');
        assert(json[r'discountMinor'] != null,
            'Required key "ReceiptLine[discountMinor]" has a null value in JSON.');
        assert(json.containsKey(r'totalMinor'),
            'Required key "ReceiptLine[totalMinor]" is missing from JSON.');
        assert(json[r'totalMinor'] != null,
            'Required key "ReceiptLine[totalMinor]" has a null value in JSON.');
        return true;
      }());

      return ReceiptLine(
        serviceId: mapValueOfType<String>(json, r'serviceId'),
        name: mapValueOfType<String>(json, r'name')!,
        quantity: mapValueOfType<int>(json, r'quantity')!,
        unitPriceMinor: mapValueOfType<int>(json, r'unitPriceMinor')!,
        discountMinor: mapValueOfType<int>(json, r'discountMinor')!,
        vatRateBp: mapValueOfType<int>(json, r'vatRateBp'),
        totalMinor: mapValueOfType<int>(json, r'totalMinor')!,
      );
    }
    return null;
  }

  static List<ReceiptLine> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ReceiptLine>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ReceiptLine.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ReceiptLine> mapFromJson(dynamic json) {
    final map = <String, ReceiptLine>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ReceiptLine.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ReceiptLine-objects as value to a dart map
  static Map<String, List<ReceiptLine>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ReceiptLine>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ReceiptLine.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'name',
    'quantity',
    'unitPriceMinor',
    'discountMinor',
    'totalMinor',
  };
}
