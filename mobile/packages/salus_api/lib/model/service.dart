//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Service {
  /// Returns a new [Service] instance.
  Service({
    required this.id,
    required this.name,
    this.code,
    required this.priceMinor,
    required this.currency,
    this.durationMinutes,
    this.vatRateBp,
    this.modes = const {},
    required this.active,
    this.translations = const [],
    this.deviceId,
    required this.versionId,
  });

  String id;

  /// In the clinic's language.
  String name;

  /// The clinic's own code; absent for none.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? code;

  /// In the currency's minor unit.
  ///
  /// Minimum value: 0
  int priceMinor;

  String currency;

  /// Absent for the slot's length.
  ///
  /// Minimum value: 5
  /// Maximum value: 480
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  int? durationMinutes;

  /// Basis points (1700 is 17 %); absent for exempt.
  ///
  /// Minimum value: 0
  /// Maximum value: 10000
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  int? vatRateBp;

  Set<VisitMode> modes;

  bool active;

  List<ServiceTranslation> translations;

  /// The device a visit for it needs (U4 of docs/clinic-cms/WAVE6.md, x-salus-package cms-rm1); absent for none.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? deviceId;

  int versionId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Service &&
          other.id == id &&
          other.name == name &&
          other.code == code &&
          other.priceMinor == priceMinor &&
          other.currency == currency &&
          other.durationMinutes == durationMinutes &&
          other.vatRateBp == vatRateBp &&
          _deepEquality.equals(other.modes, modes) &&
          other.active == active &&
          _deepEquality.equals(other.translations, translations) &&
          other.deviceId == deviceId &&
          other.versionId == versionId;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (name.hashCode) +
      (code == null ? 0 : code!.hashCode) +
      (priceMinor.hashCode) +
      (currency.hashCode) +
      (durationMinutes == null ? 0 : durationMinutes!.hashCode) +
      (vatRateBp == null ? 0 : vatRateBp!.hashCode) +
      (modes.hashCode) +
      (active.hashCode) +
      (translations.hashCode) +
      (deviceId == null ? 0 : deviceId!.hashCode) +
      (versionId.hashCode);

  @override
  String toString() =>
      'Service[id=$id, name=$name, code=$code, priceMinor=$priceMinor, currency=$currency, durationMinutes=$durationMinutes, vatRateBp=$vatRateBp, modes=$modes, active=$active, translations=$translations, deviceId=$deviceId, versionId=$versionId]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'name'] = this.name;
    if (this.code != null) {
      json[r'code'] = this.code;
    } else {
      json[r'code'] = null;
    }
    json[r'priceMinor'] = this.priceMinor;
    json[r'currency'] = this.currency;
    if (this.durationMinutes != null) {
      json[r'durationMinutes'] = this.durationMinutes;
    } else {
      json[r'durationMinutes'] = null;
    }
    if (this.vatRateBp != null) {
      json[r'vatRateBp'] = this.vatRateBp;
    } else {
      json[r'vatRateBp'] = null;
    }
    json[r'modes'] = this.modes.toList(growable: false);
    json[r'active'] = this.active;
    json[r'translations'] = this.translations;
    if (this.deviceId != null) {
      json[r'deviceId'] = this.deviceId;
    } else {
      json[r'deviceId'] = null;
    }
    json[r'versionId'] = this.versionId;
    return json;
  }

  /// Returns a new [Service] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Service? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "Service[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "Service[id]" has a null value in JSON.');
        assert(json.containsKey(r'name'),
            'Required key "Service[name]" is missing from JSON.');
        assert(json[r'name'] != null,
            'Required key "Service[name]" has a null value in JSON.');
        assert(json.containsKey(r'priceMinor'),
            'Required key "Service[priceMinor]" is missing from JSON.');
        assert(json[r'priceMinor'] != null,
            'Required key "Service[priceMinor]" has a null value in JSON.');
        assert(json.containsKey(r'currency'),
            'Required key "Service[currency]" is missing from JSON.');
        assert(json[r'currency'] != null,
            'Required key "Service[currency]" has a null value in JSON.');
        assert(json.containsKey(r'modes'),
            'Required key "Service[modes]" is missing from JSON.');
        assert(json[r'modes'] != null,
            'Required key "Service[modes]" has a null value in JSON.');
        assert(json.containsKey(r'active'),
            'Required key "Service[active]" is missing from JSON.');
        assert(json[r'active'] != null,
            'Required key "Service[active]" has a null value in JSON.');
        assert(json.containsKey(r'translations'),
            'Required key "Service[translations]" is missing from JSON.');
        assert(json[r'translations'] != null,
            'Required key "Service[translations]" has a null value in JSON.');
        assert(json.containsKey(r'versionId'),
            'Required key "Service[versionId]" is missing from JSON.');
        assert(json[r'versionId'] != null,
            'Required key "Service[versionId]" has a null value in JSON.');
        return true;
      }());

      return Service(
        id: mapValueOfType<String>(json, r'id')!,
        name: mapValueOfType<String>(json, r'name')!,
        code: mapValueOfType<String>(json, r'code'),
        priceMinor: mapValueOfType<int>(json, r'priceMinor')!,
        currency: mapValueOfType<String>(json, r'currency')!,
        durationMinutes: mapValueOfType<int>(json, r'durationMinutes'),
        vatRateBp: mapValueOfType<int>(json, r'vatRateBp'),
        modes: VisitMode.listFromJson(json[r'modes']).toSet(),
        active: mapValueOfType<bool>(json, r'active')!,
        translations: ServiceTranslation.listFromJson(json[r'translations']),
        deviceId: mapValueOfType<String>(json, r'deviceId'),
        versionId: mapValueOfType<int>(json, r'versionId')!,
      );
    }
    return null;
  }

  static List<Service> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Service>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Service.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Service> mapFromJson(dynamic json) {
    final map = <String, Service>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Service.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Service-objects as value to a dart map
  static Map<String, List<Service>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Service>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Service.listFromJson(
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
    'name',
    'priceMinor',
    'currency',
    'modes',
    'active',
    'translations',
    'versionId',
  };
}
