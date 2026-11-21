//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class IssueReceiptRequest {
  /// Returns a new [IssueReceiptRequest] instance.
  IssueReceiptRequest({
    required this.kind,
    this.patientId,
    this.companyName,
    this.companyTaxId,
    this.appointmentId,
    required this.paymentMethod,
    this.lines = const [],
  });

  ReceiptKind kind;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? patientId;

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

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? appointmentId;

  PaymentMethod paymentMethod;

  List<ReceiptPick> lines;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is IssueReceiptRequest &&
          other.kind == kind &&
          other.patientId == patientId &&
          other.companyName == companyName &&
          other.companyTaxId == companyTaxId &&
          other.appointmentId == appointmentId &&
          other.paymentMethod == paymentMethod &&
          _deepEquality.equals(other.lines, lines);

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (kind.hashCode) +
      (patientId == null ? 0 : patientId!.hashCode) +
      (companyName == null ? 0 : companyName!.hashCode) +
      (companyTaxId == null ? 0 : companyTaxId!.hashCode) +
      (appointmentId == null ? 0 : appointmentId!.hashCode) +
      (paymentMethod.hashCode) +
      (lines.hashCode);

  @override
  String toString() =>
      'IssueReceiptRequest[kind=$kind, patientId=$patientId, companyName=$companyName, companyTaxId=$companyTaxId, appointmentId=$appointmentId, paymentMethod=$paymentMethod, lines=$lines]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'kind'] = this.kind;
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
    json[r'paymentMethod'] = this.paymentMethod;
    json[r'lines'] = this.lines;
    return json;
  }

  /// Returns a new [IssueReceiptRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static IssueReceiptRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'kind'),
            'Required key "IssueReceiptRequest[kind]" is missing from JSON.');
        assert(json[r'kind'] != null,
            'Required key "IssueReceiptRequest[kind]" has a null value in JSON.');
        assert(json.containsKey(r'paymentMethod'),
            'Required key "IssueReceiptRequest[paymentMethod]" is missing from JSON.');
        assert(json[r'paymentMethod'] != null,
            'Required key "IssueReceiptRequest[paymentMethod]" has a null value in JSON.');
        assert(json.containsKey(r'lines'),
            'Required key "IssueReceiptRequest[lines]" is missing from JSON.');
        assert(json[r'lines'] != null,
            'Required key "IssueReceiptRequest[lines]" has a null value in JSON.');
        return true;
      }());

      return IssueReceiptRequest(
        kind: ReceiptKind.fromJson(json[r'kind'])!,
        patientId: mapValueOfType<String>(json, r'patientId'),
        companyName: mapValueOfType<String>(json, r'companyName'),
        companyTaxId: mapValueOfType<String>(json, r'companyTaxId'),
        appointmentId: mapValueOfType<String>(json, r'appointmentId'),
        paymentMethod: PaymentMethod.fromJson(json[r'paymentMethod'])!,
        lines: ReceiptPick.listFromJson(json[r'lines']),
      );
    }
    return null;
  }

  static List<IssueReceiptRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <IssueReceiptRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = IssueReceiptRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, IssueReceiptRequest> mapFromJson(dynamic json) {
    final map = <String, IssueReceiptRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = IssueReceiptRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of IssueReceiptRequest-objects as value to a dart map
  static Map<String, List<IssueReceiptRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<IssueReceiptRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = IssueReceiptRequest.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'kind',
    'paymentMethod',
    'lines',
  };
}
