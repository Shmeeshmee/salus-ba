//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ReceiptsReportByPaymentMethodInner {
  /// Returns a new [ReceiptsReportByPaymentMethodInner] instance.
  ReceiptsReportByPaymentMethodInner({
    required this.paymentMethod,
    required this.count,
    required this.totalMinor,
  });

  PaymentMethod paymentMethod;

  int count;

  int totalMinor;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ReceiptsReportByPaymentMethodInner &&
          other.paymentMethod == paymentMethod &&
          other.count == count &&
          other.totalMinor == totalMinor;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (paymentMethod.hashCode) + (count.hashCode) + (totalMinor.hashCode);

  @override
  String toString() =>
      'ReceiptsReportByPaymentMethodInner[paymentMethod=$paymentMethod, count=$count, totalMinor=$totalMinor]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'paymentMethod'] = this.paymentMethod;
    json[r'count'] = this.count;
    json[r'totalMinor'] = this.totalMinor;
    return json;
  }

  /// Returns a new [ReceiptsReportByPaymentMethodInner] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ReceiptsReportByPaymentMethodInner? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'paymentMethod'),
            'Required key "ReceiptsReportByPaymentMethodInner[paymentMethod]" is missing from JSON.');
        assert(json[r'paymentMethod'] != null,
            'Required key "ReceiptsReportByPaymentMethodInner[paymentMethod]" has a null value in JSON.');
        assert(json.containsKey(r'count'),
            'Required key "ReceiptsReportByPaymentMethodInner[count]" is missing from JSON.');
        assert(json[r'count'] != null,
            'Required key "ReceiptsReportByPaymentMethodInner[count]" has a null value in JSON.');
        assert(json.containsKey(r'totalMinor'),
            'Required key "ReceiptsReportByPaymentMethodInner[totalMinor]" is missing from JSON.');
        assert(json[r'totalMinor'] != null,
            'Required key "ReceiptsReportByPaymentMethodInner[totalMinor]" has a null value in JSON.');
        return true;
      }());

      return ReceiptsReportByPaymentMethodInner(
        paymentMethod: PaymentMethod.fromJson(json[r'paymentMethod'])!,
        count: mapValueOfType<int>(json, r'count')!,
        totalMinor: mapValueOfType<int>(json, r'totalMinor')!,
      );
    }
    return null;
  }

  static List<ReceiptsReportByPaymentMethodInner> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ReceiptsReportByPaymentMethodInner>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ReceiptsReportByPaymentMethodInner.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ReceiptsReportByPaymentMethodInner> mapFromJson(
      dynamic json) {
    final map = <String, ReceiptsReportByPaymentMethodInner>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ReceiptsReportByPaymentMethodInner.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ReceiptsReportByPaymentMethodInner-objects as value to a dart map
  static Map<String, List<ReceiptsReportByPaymentMethodInner>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ReceiptsReportByPaymentMethodInner>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ReceiptsReportByPaymentMethodInner.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'paymentMethod',
    'count',
    'totalMinor',
  };
}
