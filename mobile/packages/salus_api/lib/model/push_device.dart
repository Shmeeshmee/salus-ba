//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class PushDevice {
  /// Returns a new [PushDevice] instance.
  PushDevice({
    required this.platform,
    required this.token,
  });

  PushDevicePlatformEnum platform;

  /// The FCM registration token or the APNs device token.
  String token;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is PushDevice && other.platform == platform && other.token == token;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (platform.hashCode) + (token.hashCode);

  @override
  String toString() => 'PushDevice[platform=$platform, token=$token]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'platform'] = this.platform;
    json[r'token'] = this.token;
    return json;
  }

  /// Returns a new [PushDevice] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static PushDevice? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'platform'),
            'Required key "PushDevice[platform]" is missing from JSON.');
        assert(json[r'platform'] != null,
            'Required key "PushDevice[platform]" has a null value in JSON.');
        assert(json.containsKey(r'token'),
            'Required key "PushDevice[token]" is missing from JSON.');
        assert(json[r'token'] != null,
            'Required key "PushDevice[token]" has a null value in JSON.');
        return true;
      }());

      return PushDevice(
        platform: PushDevicePlatformEnum.fromJson(json[r'platform'])!,
        token: mapValueOfType<String>(json, r'token')!,
      );
    }
    return null;
  }

  static List<PushDevice> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <PushDevice>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = PushDevice.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, PushDevice> mapFromJson(dynamic json) {
    final map = <String, PushDevice>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = PushDevice.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of PushDevice-objects as value to a dart map
  static Map<String, List<PushDevice>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<PushDevice>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = PushDevice.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'platform',
    'token',
  };
}

enum PushDevicePlatformEnum {
  android._(r'android'),
  ios._(r'ios'),
  ;

  /// Instantiate a new enum with the provided value.
  const PushDevicePlatformEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [PushDevicePlatformEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static PushDevicePlatformEnum? fromJson(dynamic value) =>
      PushDevicePlatformEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [PushDevicePlatformEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<PushDevicePlatformEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <PushDevicePlatformEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = PushDevicePlatformEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [PushDevicePlatformEnum] to String,
/// and [decode] dynamic data back to [PushDevicePlatformEnum].
class PushDevicePlatformEnumTypeTransformer {
  factory PushDevicePlatformEnumTypeTransformer() =>
      _instance ??= const PushDevicePlatformEnumTypeTransformer._();

  const PushDevicePlatformEnumTypeTransformer._();

  String encode(PushDevicePlatformEnum data) => data._value;

  /// Returns the instance of [PushDevicePlatformEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  PushDevicePlatformEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is PushDevicePlatformEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'android':
          return PushDevicePlatformEnum.android;
        case r'ios':
          return PushDevicePlatformEnum.ios;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static PushDevicePlatformEnumTypeTransformer? _instance;
}
