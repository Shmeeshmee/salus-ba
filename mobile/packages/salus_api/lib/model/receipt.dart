//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Receipt {
  /// Returns a new [Receipt] instance.
  Receipt({
    required this.id,
    required this.kind,
    required this.number,
    required this.issuedAt,
    this.patientId,
    this.companyName,
    this.companyTaxId,
    this.appointmentId,
    this.reversesId,
    this.reversedById,
    required this.paymentMethod,
    required this.totalMinor,
    required this.currency,
    required this.fiscalStatus,
    this.fiscalId,
    this.fiscalCode,
    this.fiscalizedAt,
    this.lines = const [],
  });

  String id;

  ReceiptKind kind;

  /// In the clinic's sequence.
  int number;

  DateTime issuedAt;

  /// Absent unless to a patient.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? patientId;

  /// Absent unless to a company.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? companyName;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? companyTaxId;

  /// The visit; absent for a receipt on its own.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? appointmentId;

  /// The receipt it reverses; absent unless a reversal.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? reversesId;

  /// The receipt that reverses it; absent unless reversed.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? reversedById;

  PaymentMethod paymentMethod;

  int totalMinor;

  String currency;

  FiscalStatus fiscalStatus;

  /// The tax authority's identifier; absent until fiscalized.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? fiscalId;

  /// The verification code printed on it; absent until fiscalized.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? fiscalCode;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? fiscalizedAt;

  List<ReceiptLine> lines;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Receipt &&
          other.id == id &&
          other.kind == kind &&
          other.number == number &&
          other.issuedAt == issuedAt &&
          other.patientId == patientId &&
          other.companyName == companyName &&
          other.companyTaxId == companyTaxId &&
          other.appointmentId == appointmentId &&
          other.reversesId == reversesId &&
          other.reversedById == reversedById &&
          other.paymentMethod == paymentMethod &&
          other.totalMinor == totalMinor &&
          other.currency == currency &&
          other.fiscalStatus == fiscalStatus &&
          other.fiscalId == fiscalId &&
          other.fiscalCode == fiscalCode &&
          other.fiscalizedAt == fiscalizedAt &&
          _deepEquality.equals(other.lines, lines);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (kind.hashCode) +
      (number.hashCode) +
      (issuedAt.hashCode) +
      (patientId == null ? 0 : patientId!.hashCode) +
      (companyName == null ? 0 : companyName!.hashCode) +
      (companyTaxId == null ? 0 : companyTaxId!.hashCode) +
      (appointmentId == null ? 0 : appointmentId!.hashCode) +
      (reversesId == null ? 0 : reversesId!.hashCode) +
      (reversedById == null ? 0 : reversedById!.hashCode) +
      (paymentMethod.hashCode) +
      (totalMinor.hashCode) +
      (currency.hashCode) +
      (fiscalStatus.hashCode) +
      (fiscalId == null ? 0 : fiscalId!.hashCode) +
      (fiscalCode == null ? 0 : fiscalCode!.hashCode) +
      (fiscalizedAt == null ? 0 : fiscalizedAt!.hashCode) +
      (lines.hashCode);

  @override
  String toString() =>
      'Receipt[id=$id, kind=$kind, number=$number, issuedAt=$issuedAt, patientId=$patientId, companyName=$companyName, companyTaxId=$companyTaxId, appointmentId=$appointmentId, reversesId=$reversesId, reversedById=$reversedById, paymentMethod=$paymentMethod, totalMinor=$totalMinor, currency=$currency, fiscalStatus=$fiscalStatus, fiscalId=$fiscalId, fiscalCode=$fiscalCode, fiscalizedAt=$fiscalizedAt, lines=$lines]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'kind'] = this.kind;
    json[r'number'] = this.number;
    json[r'issuedAt'] = this.issuedAt.toUtc().toIso8601String();
    if (this.patientId != null) {
      json[r'patientId'] = this.patientId;
    } else {
      json[r'patientId'] = null;
    }
    if (this.companyName != null) {
      json[r'companyName'] = this.companyName;
    } else {
      json[r'companyName'] = null;
    }
    if (this.companyTaxId != null) {
      json[r'companyTaxId'] = this.companyTaxId;
    } else {
      json[r'companyTaxId'] = null;
    }
    if (this.appointmentId != null) {
      json[r'appointmentId'] = this.appointmentId;
    } else {
      json[r'appointmentId'] = null;
    }
    if (this.reversesId != null) {
      json[r'reversesId'] = this.reversesId;
    } else {
      json[r'reversesId'] = null;
    }
    if (this.reversedById != null) {
      json[r'reversedById'] = this.reversedById;
    } else {
      json[r'reversedById'] = null;
    }
    json[r'paymentMethod'] = this.paymentMethod;
    json[r'totalMinor'] = this.totalMinor;
    json[r'currency'] = this.currency;
    json[r'fiscalStatus'] = this.fiscalStatus;
    if (this.fiscalId != null) {
      json[r'fiscalId'] = this.fiscalId;
    } else {
      json[r'fiscalId'] = null;
    }
    if (this.fiscalCode != null) {
      json[r'fiscalCode'] = this.fiscalCode;
    } else {
      json[r'fiscalCode'] = null;
    }
    if (this.fiscalizedAt != null) {
      json[r'fiscalizedAt'] = this.fiscalizedAt!.toUtc().toIso8601String();
    } else {
      json[r'fiscalizedAt'] = null;
    }
    json[r'lines'] = this.lines;
    return json;
  }

  /// Returns a new [Receipt] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Receipt? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "Receipt[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "Receipt[id]" has a null value in JSON.');
        assert(json.containsKey(r'kind'),
            'Required key "Receipt[kind]" is missing from JSON.');
        assert(json[r'kind'] != null,
            'Required key "Receipt[kind]" has a null value in JSON.');
        assert(json.containsKey(r'number'),
            'Required key "Receipt[number]" is missing from JSON.');
        assert(json[r'number'] != null,
            'Required key "Receipt[number]" has a null value in JSON.');
        assert(json.containsKey(r'issuedAt'),
            'Required key "Receipt[issuedAt]" is missing from JSON.');
        assert(json[r'issuedAt'] != null,
            'Required key "Receipt[issuedAt]" has a null value in JSON.');
        assert(json.containsKey(r'paymentMethod'),
            'Required key "Receipt[paymentMethod]" is missing from JSON.');
        assert(json[r'paymentMethod'] != null,
            'Required key "Receipt[paymentMethod]" has a null value in JSON.');
        assert(json.containsKey(r'totalMinor'),
            'Required key "Receipt[totalMinor]" is missing from JSON.');
        assert(json[r'totalMinor'] != null,
            'Required key "Receipt[totalMinor]" has a null value in JSON.');
        assert(json.containsKey(r'currency'),
            'Required key "Receipt[currency]" is missing from JSON.');
        assert(json[r'currency'] != null,
            'Required key "Receipt[currency]" has a null value in JSON.');
        assert(json.containsKey(r'fiscalStatus'),
            'Required key "Receipt[fiscalStatus]" is missing from JSON.');
        assert(json[r'fiscalStatus'] != null,
            'Required key "Receipt[fiscalStatus]" has a null value in JSON.');
        assert(json.containsKey(r'lines'),
            'Required key "Receipt[lines]" is missing from JSON.');
        assert(json[r'lines'] != null,
            'Required key "Receipt[lines]" has a null value in JSON.');
        return true;
      }());

      return Receipt(
        id: mapValueOfType<String>(json, r'id')!,
        kind: ReceiptKind.fromJson(json[r'kind'])!,
        number: mapValueOfType<int>(json, r'number')!,
        issuedAt: mapDateTime(json, r'issuedAt', r'')!,
        patientId: mapValueOfType<String>(json, r'patientId'),
        companyName: mapValueOfType<String>(json, r'companyName'),
        companyTaxId: mapValueOfType<String>(json, r'companyTaxId'),
        appointmentId: mapValueOfType<String>(json, r'appointmentId'),
        reversesId: mapValueOfType<String>(json, r'reversesId'),
        reversedById: mapValueOfType<String>(json, r'reversedById'),
        paymentMethod: PaymentMethod.fromJson(json[r'paymentMethod'])!,
        totalMinor: mapValueOfType<int>(json, r'totalMinor')!,
        currency: mapValueOfType<String>(json, r'currency')!,
        fiscalStatus: FiscalStatus.fromJson(json[r'fiscalStatus'])!,
        fiscalId: mapValueOfType<String>(json, r'fiscalId'),
        fiscalCode: mapValueOfType<String>(json, r'fiscalCode'),
        fiscalizedAt: mapDateTime(json, r'fiscalizedAt', r''),
        lines: ReceiptLine.listFromJson(json[r'lines']),
      );
    }
    return null;
  }

  static List<Receipt> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Receipt>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Receipt.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Receipt> mapFromJson(dynamic json) {
    final map = <String, Receipt>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Receipt.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Receipt-objects as value to a dart map
  static Map<String, List<Receipt>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Receipt>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Receipt.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'id',
    'kind',
    'number',
    'issuedAt',
    'paymentMethod',
    'totalMinor',
    'currency',
    'fiscalStatus',
    'lines',
  };
}
