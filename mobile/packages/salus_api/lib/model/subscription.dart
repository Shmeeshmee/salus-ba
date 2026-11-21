//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Subscription {
  /// Returns a new [Subscription] instance.
  Subscription({
    required this.planCode,
    required this.planName,
    required this.pricePerSeatMinor,
    required this.currency,
    required this.status,
    this.trialEndsAt,
    this.currentPeriodStart,
    this.currentPeriodEnd,
    this.pastDueSince,
  });

  String planCode;

  String planName;

  /// Per active practitioner per month.
  int pricePerSeatMinor;

  String currency;

  SubscriptionStatus status;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? trialEndsAt;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? currentPeriodStart;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? currentPeriodEnd;

  /// The due date of the oldest open invoice past due; absent for none.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? pastDueSince;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Subscription &&
          other.planCode == planCode &&
          other.planName == planName &&
          other.pricePerSeatMinor == pricePerSeatMinor &&
          other.currency == currency &&
          other.status == status &&
          other.trialEndsAt == trialEndsAt &&
          other.currentPeriodStart == currentPeriodStart &&
          other.currentPeriodEnd == currentPeriodEnd &&
          other.pastDueSince == pastDueSince;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (planCode.hashCode) +
      (planName.hashCode) +
      (pricePerSeatMinor.hashCode) +
      (currency.hashCode) +
      (status.hashCode) +
      (trialEndsAt == null ? 0 : trialEndsAt!.hashCode) +
      (currentPeriodStart == null ? 0 : currentPeriodStart!.hashCode) +
      (currentPeriodEnd == null ? 0 : currentPeriodEnd!.hashCode) +
      (pastDueSince == null ? 0 : pastDueSince!.hashCode);

  @override
  String toString() =>
      'Subscription[planCode=$planCode, planName=$planName, pricePerSeatMinor=$pricePerSeatMinor, currency=$currency, status=$status, trialEndsAt=$trialEndsAt, currentPeriodStart=$currentPeriodStart, currentPeriodEnd=$currentPeriodEnd, pastDueSince=$pastDueSince]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'planCode'] = this.planCode;
    json[r'planName'] = this.planName;
    json[r'pricePerSeatMinor'] = this.pricePerSeatMinor;
    json[r'currency'] = this.currency;
    json[r'status'] = this.status;
    if (this.trialEndsAt != null) {
      json[r'trialEndsAt'] = this.trialEndsAt!.toUtc().toIso8601String();
    } else {
      json[r'trialEndsAt'] = null;
    }
    if (this.currentPeriodStart != null) {
      json[r'currentPeriodStart'] =
          this.currentPeriodStart!.toUtc().toIso8601String();
    } else {
      json[r'currentPeriodStart'] = null;
    }
    if (this.currentPeriodEnd != null) {
      json[r'currentPeriodEnd'] =
          this.currentPeriodEnd!.toUtc().toIso8601String();
    } else {
      json[r'currentPeriodEnd'] = null;
    }
    if (this.pastDueSince != null) {
      json[r'pastDueSince'] = this.pastDueSince!.toUtc().toIso8601String();
    } else {
      json[r'pastDueSince'] = null;
    }
    return json;
  }

  /// Returns a new [Subscription] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Subscription? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'planCode'),
            'Required key "Subscription[planCode]" is missing from JSON.');
        assert(json[r'planCode'] != null,
            'Required key "Subscription[planCode]" has a null value in JSON.');
        assert(json.containsKey(r'planName'),
            'Required key "Subscription[planName]" is missing from JSON.');
        assert(json[r'planName'] != null,
            'Required key "Subscription[planName]" has a null value in JSON.');
        assert(json.containsKey(r'pricePerSeatMinor'),
            'Required key "Subscription[pricePerSeatMinor]" is missing from JSON.');
        assert(json[r'pricePerSeatMinor'] != null,
            'Required key "Subscription[pricePerSeatMinor]" has a null value in JSON.');
        assert(json.containsKey(r'currency'),
            'Required key "Subscription[currency]" is missing from JSON.');
        assert(json[r'currency'] != null,
            'Required key "Subscription[currency]" has a null value in JSON.');
        assert(json.containsKey(r'status'),
            'Required key "Subscription[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "Subscription[status]" has a null value in JSON.');
        return true;
      }());

      return Subscription(
        planCode: mapValueOfType<String>(json, r'planCode')!,
        planName: mapValueOfType<String>(json, r'planName')!,
        pricePerSeatMinor: mapValueOfType<int>(json, r'pricePerSeatMinor')!,
        currency: mapValueOfType<String>(json, r'currency')!,
        status: SubscriptionStatus.fromJson(json[r'status'])!,
        trialEndsAt: mapDateTime(json, r'trialEndsAt', r''),
        currentPeriodStart: mapDateTime(json, r'currentPeriodStart', r''),
        currentPeriodEnd: mapDateTime(json, r'currentPeriodEnd', r''),
        pastDueSince: mapDateTime(json, r'pastDueSince', r''),
      );
    }
    return null;
  }

  static List<Subscription> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Subscription>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Subscription.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Subscription> mapFromJson(dynamic json) {
    final map = <String, Subscription>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Subscription.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Subscription-objects as value to a dart map
  static Map<String, List<Subscription>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Subscription>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Subscription.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'planCode',
    'planName',
    'pricePerSeatMinor',
    'currency',
    'status',
  };
}
