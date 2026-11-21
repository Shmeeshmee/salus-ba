//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ReceiptsReportByServiceInner {
  /// Returns a new [ReceiptsReportByServiceInner] instance.
  ReceiptsReportByServiceInner({
    this.serviceId,
    required this.name,
    required this.quantity,
    required this.totalMinor,
  });

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? serviceId;

  String name;

  int quantity;

  int totalMinor;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ReceiptsReportByServiceInner &&
          other.serviceId == serviceId &&
          other.name == name &&
          other.quantity == quantity &&
          other.totalMinor == totalMinor;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (serviceId == null ? 0 : serviceId!.hashCode) +
      (name.hashCode) +
      (quantity.hashCode) +
      (totalMinor.hashCode);

  @override
  String toString() =>
      'ReceiptsReportByServiceInner[serviceId=$serviceId, name=$name, quantity=$quantity, totalMinor=$totalMinor]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    if (this.serviceId != null) {
      json[r'serviceId'] = this.serviceId;
    } else {
      json[r'serviceId'] = null;
    }
    json[r'name'] = this.name;
    json[r'quantity'] = this.quantity;
    json[r'totalMinor'] = this.totalMinor;
    return json;
  }

  /// Returns a new [ReceiptsReportByServiceInner] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ReceiptsReportByServiceInner? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'name'),
            'Required key "ReceiptsReportByServiceInner[name]" is missing from JSON.');
        assert(json[r'name'] != null,
            'Required key "ReceiptsReportByServiceInner[name]" has a null value in JSON.');
        assert(json.containsKey(r'quantity'),
            'Required key "ReceiptsReportByServiceInner[quantity]" is missing from JSON.');
        assert(json[r'quantity'] != null,
            'Required key "ReceiptsReportByServiceInner[quantity]" has a null value in JSON.');
        assert(json.containsKey(r'totalMinor'),
            'Required key "ReceiptsReportByServiceInner[totalMinor]" is missing from JSON.');
        assert(json[r'totalMinor'] != null,
            'Required key "ReceiptsReportByServiceInner[totalMinor]" has a null value in JSON.');
        return true;
      }());

      return ReceiptsReportByServiceInner(
        serviceId: mapValueOfType<String>(json, r'serviceId'),
        name: mapValueOfType<String>(json, r'name')!,
        quantity: mapValueOfType<int>(json, r'quantity')!,
        totalMinor: mapValueOfType<int>(json, r'totalMinor')!,
      );
    }
    return null;
  }

  static List<ReceiptsReportByServiceInner> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ReceiptsReportByServiceInner>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ReceiptsReportByServiceInner.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ReceiptsReportByServiceInner> mapFromJson(dynamic json) {
    final map = <String, ReceiptsReportByServiceInner>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ReceiptsReportByServiceInner.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ReceiptsReportByServiceInner-objects as value to a dart map
  static Map<String, List<ReceiptsReportByServiceInner>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ReceiptsReportByServiceInner>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ReceiptsReportByServiceInner.listFromJson(
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
    'totalMinor',
  };
}
