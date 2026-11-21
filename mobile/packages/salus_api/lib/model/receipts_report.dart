//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ReceiptsReport {
  /// Returns a new [ReceiptsReport] instance.
  ReceiptsReport({
    required this.from,
    required this.to,
    required this.count,
    required this.totalMinor,
    required this.currency,
    this.byPaymentMethod = const [],
    this.byService = const [],
  });

  DateTime from;

  DateTime to;

  /// Receipts issued
  int count;

  int totalMinor;

  String currency;

  List<ReceiptsReportByPaymentMethodInner> byPaymentMethod;

  List<ReceiptsReportByServiceInner> byService;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ReceiptsReport &&
          other.from == from &&
          other.to == to &&
          other.count == count &&
          other.totalMinor == totalMinor &&
          other.currency == currency &&
          _deepEquality.equals(other.byPaymentMethod, byPaymentMethod) &&
          _deepEquality.equals(other.byService, byService);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (from.hashCode) +
      (to.hashCode) +
      (count.hashCode) +
      (totalMinor.hashCode) +
      (currency.hashCode) +
      (byPaymentMethod.hashCode) +
      (byService.hashCode);

  @override
  String toString() =>
      'ReceiptsReport[from=$from, to=$to, count=$count, totalMinor=$totalMinor, currency=$currency, byPaymentMethod=$byPaymentMethod, byService=$byService]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'from'] = this.from.toUtc().toIso8601String();
    json[r'to'] = this.to.toUtc().toIso8601String();
    json[r'count'] = this.count;
    json[r'totalMinor'] = this.totalMinor;
    json[r'currency'] = this.currency;
    json[r'byPaymentMethod'] = this.byPaymentMethod;
    json[r'byService'] = this.byService;
    return json;
  }

  /// Returns a new [ReceiptsReport] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ReceiptsReport? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'from'),
            'Required key "ReceiptsReport[from]" is missing from JSON.');
        assert(json[r'from'] != null,
            'Required key "ReceiptsReport[from]" has a null value in JSON.');
        assert(json.containsKey(r'to'),
            'Required key "ReceiptsReport[to]" is missing from JSON.');
        assert(json[r'to'] != null,
            'Required key "ReceiptsReport[to]" has a null value in JSON.');
        assert(json.containsKey(r'count'),
            'Required key "ReceiptsReport[count]" is missing from JSON.');
        assert(json[r'count'] != null,
            'Required key "ReceiptsReport[count]" has a null value in JSON.');
        assert(json.containsKey(r'totalMinor'),
            'Required key "ReceiptsReport[totalMinor]" is missing from JSON.');
        assert(json[r'totalMinor'] != null,
            'Required key "ReceiptsReport[totalMinor]" has a null value in JSON.');
        assert(json.containsKey(r'currency'),
            'Required key "ReceiptsReport[currency]" is missing from JSON.');
        assert(json[r'currency'] != null,
            'Required key "ReceiptsReport[currency]" has a null value in JSON.');
        assert(json.containsKey(r'byPaymentMethod'),
            'Required key "ReceiptsReport[byPaymentMethod]" is missing from JSON.');
        assert(json[r'byPaymentMethod'] != null,
            'Required key "ReceiptsReport[byPaymentMethod]" has a null value in JSON.');
        assert(json.containsKey(r'byService'),
            'Required key "ReceiptsReport[byService]" is missing from JSON.');
        assert(json[r'byService'] != null,
            'Required key "ReceiptsReport[byService]" has a null value in JSON.');
        return true;
      }());

      return ReceiptsReport(
        from: mapDateTime(json, r'from', r'')!,
        to: mapDateTime(json, r'to', r'')!,
        count: mapValueOfType<int>(json, r'count')!,
        totalMinor: mapValueOfType<int>(json, r'totalMinor')!,
        currency: mapValueOfType<String>(json, r'currency')!,
        byPaymentMethod: ReceiptsReportByPaymentMethodInner.listFromJson(
            json[r'byPaymentMethod']),
        byService:
            ReceiptsReportByServiceInner.listFromJson(json[r'byService']),
      );
    }
    return null;
  }

  static List<ReceiptsReport> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ReceiptsReport>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ReceiptsReport.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ReceiptsReport> mapFromJson(dynamic json) {
    final map = <String, ReceiptsReport>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ReceiptsReport.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ReceiptsReport-objects as value to a dart map
  static Map<String, List<ReceiptsReport>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ReceiptsReport>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ReceiptsReport.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'from',
    'to',
    'count',
    'totalMinor',
    'currency',
    'byPaymentMethod',
    'byService',
  };
}
