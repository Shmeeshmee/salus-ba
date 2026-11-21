//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SubscriptionInvoice {
  /// Returns a new [SubscriptionInvoice] instance.
  SubscriptionInvoice({
    required this.id,
    required this.number,
    required this.periodStart,
    required this.periodEnd,
    required this.seats,
    required this.amountMinor,
    required this.currency,
    required this.issuedAt,
    required this.dueAt,
    this.paidAt,
    required this.status,
  });

  String id;

  int number;

  DateTime periodStart;

  /// Exclusive.
  DateTime periodEnd;

  /// Active practitioner roles on the 1st.
  int seats;

  int amountMinor;

  String currency;

  DateTime issuedAt;

  DateTime dueAt;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? paidAt;

  SubscriptionInvoiceStatus status;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SubscriptionInvoice &&
          other.id == id &&
          other.number == number &&
          other.periodStart == periodStart &&
          other.periodEnd == periodEnd &&
          other.seats == seats &&
          other.amountMinor == amountMinor &&
          other.currency == currency &&
          other.issuedAt == issuedAt &&
          other.dueAt == dueAt &&
          other.paidAt == paidAt &&
          other.status == status;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (number.hashCode) +
      (periodStart.hashCode) +
      (periodEnd.hashCode) +
      (seats.hashCode) +
      (amountMinor.hashCode) +
      (currency.hashCode) +
      (issuedAt.hashCode) +
      (dueAt.hashCode) +
      (paidAt == null ? 0 : paidAt!.hashCode) +
      (status.hashCode);

  @override
  String toString() =>
      'SubscriptionInvoice[id=$id, number=$number, periodStart=$periodStart, periodEnd=$periodEnd, seats=$seats, amountMinor=$amountMinor, currency=$currency, issuedAt=$issuedAt, dueAt=$dueAt, paidAt=$paidAt, status=$status]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'number'] = this.number;
    json[r'periodStart'] = _dateFormatter.format(this.periodStart);
    json[r'periodEnd'] = _dateFormatter.format(this.periodEnd);
    json[r'seats'] = this.seats;
    json[r'amountMinor'] = this.amountMinor;
    json[r'currency'] = this.currency;
    json[r'issuedAt'] = this.issuedAt.toUtc().toIso8601String();
    json[r'dueAt'] = this.dueAt.toUtc().toIso8601String();
    if (this.paidAt != null) {
      json[r'paidAt'] = this.paidAt!.toUtc().toIso8601String();
    } else {
      json[r'paidAt'] = null;
    }
    json[r'status'] = this.status;
    return json;
  }

  /// Returns a new [SubscriptionInvoice] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SubscriptionInvoice? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "SubscriptionInvoice[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "SubscriptionInvoice[id]" has a null value in JSON.');
        assert(json.containsKey(r'number'),
            'Required key "SubscriptionInvoice[number]" is missing from JSON.');
        assert(json[r'number'] != null,
            'Required key "SubscriptionInvoice[number]" has a null value in JSON.');
        assert(json.containsKey(r'periodStart'),
            'Required key "SubscriptionInvoice[periodStart]" is missing from JSON.');
        assert(json[r'periodStart'] != null,
            'Required key "SubscriptionInvoice[periodStart]" has a null value in JSON.');
        assert(json.containsKey(r'periodEnd'),
            'Required key "SubscriptionInvoice[periodEnd]" is missing from JSON.');
        assert(json[r'periodEnd'] != null,
            'Required key "SubscriptionInvoice[periodEnd]" has a null value in JSON.');
        assert(json.containsKey(r'seats'),
            'Required key "SubscriptionInvoice[seats]" is missing from JSON.');
        assert(json[r'seats'] != null,
            'Required key "SubscriptionInvoice[seats]" has a null value in JSON.');
        assert(json.containsKey(r'amountMinor'),
            'Required key "SubscriptionInvoice[amountMinor]" is missing from JSON.');
        assert(json[r'amountMinor'] != null,
            'Required key "SubscriptionInvoice[amountMinor]" has a null value in JSON.');
        assert(json.containsKey(r'currency'),
            'Required key "SubscriptionInvoice[currency]" is missing from JSON.');
        assert(json[r'currency'] != null,
            'Required key "SubscriptionInvoice[currency]" has a null value in JSON.');
        assert(json.containsKey(r'issuedAt'),
            'Required key "SubscriptionInvoice[issuedAt]" is missing from JSON.');
        assert(json[r'issuedAt'] != null,
            'Required key "SubscriptionInvoice[issuedAt]" has a null value in JSON.');
        assert(json.containsKey(r'dueAt'),
            'Required key "SubscriptionInvoice[dueAt]" is missing from JSON.');
        assert(json[r'dueAt'] != null,
            'Required key "SubscriptionInvoice[dueAt]" has a null value in JSON.');
        assert(json.containsKey(r'status'),
            'Required key "SubscriptionInvoice[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "SubscriptionInvoice[status]" has a null value in JSON.');
        return true;
      }());

      return SubscriptionInvoice(
        id: mapValueOfType<String>(json, r'id')!,
        number: mapValueOfType<int>(json, r'number')!,
        periodStart: mapDateTime(json, r'periodStart', r'')!,
        periodEnd: mapDateTime(json, r'periodEnd', r'')!,
        seats: mapValueOfType<int>(json, r'seats')!,
        amountMinor: mapValueOfType<int>(json, r'amountMinor')!,
        currency: mapValueOfType<String>(json, r'currency')!,
        issuedAt: mapDateTime(json, r'issuedAt', r'')!,
        dueAt: mapDateTime(json, r'dueAt', r'')!,
        paidAt: mapDateTime(json, r'paidAt', r''),
        status: SubscriptionInvoiceStatus.fromJson(json[r'status'])!,
      );
    }
    return null;
  }

  static List<SubscriptionInvoice> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SubscriptionInvoice>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SubscriptionInvoice.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SubscriptionInvoice> mapFromJson(dynamic json) {
    final map = <String, SubscriptionInvoice>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SubscriptionInvoice.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SubscriptionInvoice-objects as value to a dart map
  static Map<String, List<SubscriptionInvoice>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SubscriptionInvoice>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SubscriptionInvoice.listFromJson(
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
    'number',
    'periodStart',
    'periodEnd',
    'seats',
    'amountMinor',
    'currency',
    'issuedAt',
    'dueAt',
    'status',
  };
}
