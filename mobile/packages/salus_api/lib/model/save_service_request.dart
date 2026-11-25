//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SaveServiceRequest {
  /// Returns a new [SaveServiceRequest] instance.
  SaveServiceRequest({
    required this.name,
    this.code,
    required this.priceMinor,
    this.currency = 'BAM',
    this.durationMinutes,
    this.vatRateBp,
    this.modes = const {},
    required this.active,
    this.translations = const [],
    this.deviceId,
  });

  String name;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? code;

  /// Minimum value: 0
  int priceMinor;

  String currency;

  /// Minimum value: 5
  /// Maximum value: 480
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  int? durationMinutes;

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

  /// A device of the clinic in use a visit for it needs (x-salus-package cms-rm1): a booking when it is busy is refused with `resource_busy`; an unknown device, another clinic's or one not in use is `invalid_service`. Omitted for none: a change without it clears the device.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? deviceId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SaveServiceRequest &&
          other.name == name &&
          other.code == code &&
          other.priceMinor == priceMinor &&
          other.currency == currency &&
          other.durationMinutes == durationMinutes &&
          other.vatRateBp == vatRateBp &&
          _deepEquality.equals(other.modes, modes) &&
          other.active == active &&
          _deepEquality.equals(other.translations, translations) &&
          other.deviceId == deviceId;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (name.hashCode) +
      (code == null ? 0 : code!.hashCode) +
      (priceMinor.hashCode) +
      (currency.hashCode) +
      (durationMinutes == null ? 0 : durationMinutes!.hashCode) +
      (vatRateBp == null ? 0 : vatRateBp!.hashCode) +
      (modes.hashCode) +
      (active.hashCode) +
      (translations.hashCode) +
      (deviceId == null ? 0 : deviceId!.hashCode);

  @override
  String toString() =>
      'SaveServiceRequest[name=$name, code=$code, priceMinor=$priceMinor, currency=$currency, durationMinutes=$durationMinutes, vatRateBp=$vatRateBp, modes=$modes, active=$active, translations=$translations, deviceId=$deviceId]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
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
    return json;
  }

  /// Returns a new [SaveServiceRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SaveServiceRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'name'),
            'Required key "SaveServiceRequest[name]" is missing from JSON.');
        assert(json[r'name'] != null,
            'Required key "SaveServiceRequest[name]" has a null value in JSON.');
        assert(json.containsKey(r'priceMinor'),
            'Required key "SaveServiceRequest[priceMinor]" is missing from JSON.');
        assert(json[r'priceMinor'] != null,
            'Required key "SaveServiceRequest[priceMinor]" has a null value in JSON.');
        assert(json.containsKey(r'modes'),
            'Required key "SaveServiceRequest[modes]" is missing from JSON.');
        assert(json[r'modes'] != null,
            'Required key "SaveServiceRequest[modes]" has a null value in JSON.');
        assert(json.containsKey(r'active'),
            'Required key "SaveServiceRequest[active]" is missing from JSON.');
        assert(json[r'active'] != null,
            'Required key "SaveServiceRequest[active]" has a null value in JSON.');
        assert(json.containsKey(r'translations'),
            'Required key "SaveServiceRequest[translations]" is missing from JSON.');
        assert(json[r'translations'] != null,
            'Required key "SaveServiceRequest[translations]" has a null value in JSON.');
        return true;
      }());

      return SaveServiceRequest(
        name: mapValueOfType<String>(json, r'name')!,
        code: mapValueOfType<String>(json, r'code'),
        priceMinor: mapValueOfType<int>(json, r'priceMinor')!,
        currency: mapValueOfType<String>(json, r'currency') ?? 'BAM',
        durationMinutes: mapValueOfType<int>(json, r'durationMinutes'),
        vatRateBp: mapValueOfType<int>(json, r'vatRateBp'),
        modes: VisitMode.listFromJson(json[r'modes']).toSet(),
        active: mapValueOfType<bool>(json, r'active')!,
        translations: ServiceTranslation.listFromJson(json[r'translations']),
        deviceId: mapValueOfType<String>(json, r'deviceId'),
      );
    }
    return null;
  }

  static List<SaveServiceRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SaveServiceRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SaveServiceRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SaveServiceRequest> mapFromJson(dynamic json) {
    final map = <String, SaveServiceRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SaveServiceRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SaveServiceRequest-objects as value to a dart map
  static Map<String, List<SaveServiceRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SaveServiceRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SaveServiceRequest.listFromJson(
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
    'priceMinor',
    'modes',
    'active',
    'translations',
  };
}
