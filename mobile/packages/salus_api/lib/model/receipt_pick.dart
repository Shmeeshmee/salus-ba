//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ReceiptPick {
  /// Returns a new [ReceiptPick] instance.
  ReceiptPick({
    required this.serviceId,
    required this.quantity,
    this.discountMinor = 0,
  });

  String serviceId;

  /// Minimum value: 1
  /// Maximum value: 10000
  int quantity;

  /// Minimum value: 0
  int discountMinor;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ReceiptPick &&
          other.serviceId == serviceId &&
          other.quantity == quantity &&
          other.discountMinor == discountMinor;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (serviceId.hashCode) + (quantity.hashCode) + (discountMinor.hashCode);

  @override
  String toString() =>
      'ReceiptPick[serviceId=$serviceId, quantity=$quantity, discountMinor=$discountMinor]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'serviceId'] = this.serviceId;
    json[r'quantity'] = this.quantity;
    json[r'discountMinor'] = this.discountMinor;
    return json;
  }

  /// Returns a new [ReceiptPick] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ReceiptPick? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'serviceId'),
            'Required key "ReceiptPick[serviceId]" is missing from JSON.');
        assert(json[r'serviceId'] != null,
            'Required key "ReceiptPick[serviceId]" has a null value in JSON.');
        assert(json.containsKey(r'quantity'),
            'Required key "ReceiptPick[quantity]" is missing from JSON.');
        assert(json[r'quantity'] != null,
            'Required key "ReceiptPick[quantity]" has a null value in JSON.');
        return true;
      }());

      return ReceiptPick(
        serviceId: mapValueOfType<String>(json, r'serviceId')!,
        quantity: mapValueOfType<int>(json, r'quantity')!,
        discountMinor: mapValueOfType<int>(json, r'discountMinor') ?? 0,
      );
    }
    return null;
  }

  static List<ReceiptPick> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ReceiptPick>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ReceiptPick.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ReceiptPick> mapFromJson(dynamic json) {
    final map = <String, ReceiptPick>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ReceiptPick.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ReceiptPick-objects as value to a dart map
  static Map<String, List<ReceiptPick>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ReceiptPick>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ReceiptPick.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'serviceId',
    'quantity',
  };
}
