//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ReverseReceiptRequest {
  /// Returns a new [ReverseReceiptRequest] instance.
  ReverseReceiptRequest({
    required this.paymentMethod,
  });

  PaymentMethod paymentMethod;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ReverseReceiptRequest && other.paymentMethod == paymentMethod;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (paymentMethod.hashCode);

  @override
  String toString() => 'ReverseReceiptRequest[paymentMethod=$paymentMethod]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'paymentMethod'] = this.paymentMethod;
    return json;
  }

  /// Returns a new [ReverseReceiptRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ReverseReceiptRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'paymentMethod'),
            'Required key "ReverseReceiptRequest[paymentMethod]" is missing from JSON.');
        assert(json[r'paymentMethod'] != null,
            'Required key "ReverseReceiptRequest[paymentMethod]" has a null value in JSON.');
        return true;
      }());

      return ReverseReceiptRequest(
        paymentMethod: PaymentMethod.fromJson(json[r'paymentMethod'])!,
      );
    }
    return null;
  }

  static List<ReverseReceiptRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ReverseReceiptRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ReverseReceiptRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ReverseReceiptRequest> mapFromJson(dynamic json) {
    final map = <String, ReverseReceiptRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ReverseReceiptRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ReverseReceiptRequest-objects as value to a dart map
  static Map<String, List<ReverseReceiptRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ReverseReceiptRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ReverseReceiptRequest.listFromJson(
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
  };
}
