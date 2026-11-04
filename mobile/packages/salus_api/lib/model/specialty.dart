//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Specialty {
  /// Returns a new [Specialty] instance.
  Specialty({
    required this.system,
    required this.code,
    required this.display,
    required this.text,
  });

  SpecialtySystemEnum system;

  String code;

  /// The concept's display in the value set.
  String display;

  /// The name the apps show.
  String text;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Specialty &&
          other.system == system &&
          other.code == code &&
          other.display == display &&
          other.text == text;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (system.hashCode) +
      (code.hashCode) +
      (display.hashCode) +
      (text.hashCode);

  @override
  String toString() =>
      'Specialty[system=$system, code=$code, display=$display, text=$text]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'system'] = this.system;
    json[r'code'] = this.code;
    json[r'display'] = this.display;
    json[r'text'] = this.text;
    return json;
  }

  /// Returns a new [Specialty] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Specialty? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'system'),
            'Required key "Specialty[system]" is missing from JSON.');
        assert(json[r'system'] != null,
            'Required key "Specialty[system]" has a null value in JSON.');
        assert(json.containsKey(r'code'),
            'Required key "Specialty[code]" is missing from JSON.');
        assert(json[r'code'] != null,
            'Required key "Specialty[code]" has a null value in JSON.');
        assert(json.containsKey(r'display'),
            'Required key "Specialty[display]" is missing from JSON.');
        assert(json[r'display'] != null,
            'Required key "Specialty[display]" has a null value in JSON.');
        assert(json.containsKey(r'text'),
            'Required key "Specialty[text]" is missing from JSON.');
        assert(json[r'text'] != null,
            'Required key "Specialty[text]" has a null value in JSON.');
        return true;
      }());

      return Specialty(
        system: SpecialtySystemEnum.fromJson(json[r'system'])!,
        code: mapValueOfType<String>(json, r'code')!,
        display: mapValueOfType<String>(json, r'display')!,
        text: mapValueOfType<String>(json, r'text')!,
      );
    }
    return null;
  }

  static List<Specialty> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Specialty>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Specialty.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Specialty> mapFromJson(dynamic json) {
    final map = <String, Specialty>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Specialty.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Specialty-objects as value to a dart map
  static Map<String, List<Specialty>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Specialty>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Specialty.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'system',
    'code',
    'display',
    'text',
  };
}

enum SpecialtySystemEnum {
  httpColonSlashSlashSnomedPeriodInfoSlashSct._(r'http://snomed.info/sct'),
  ;

  /// Instantiate a new enum with the provided value.
  const SpecialtySystemEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [SpecialtySystemEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static SpecialtySystemEnum? fromJson(dynamic value) =>
      SpecialtySystemEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [SpecialtySystemEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<SpecialtySystemEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SpecialtySystemEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SpecialtySystemEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [SpecialtySystemEnum] to String,
/// and [decode] dynamic data back to [SpecialtySystemEnum].
class SpecialtySystemEnumTypeTransformer {
  factory SpecialtySystemEnumTypeTransformer() =>
      _instance ??= const SpecialtySystemEnumTypeTransformer._();

  const SpecialtySystemEnumTypeTransformer._();

  String encode(SpecialtySystemEnum data) => data._value;

  /// Returns the instance of [SpecialtySystemEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  SpecialtySystemEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is SpecialtySystemEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'http://snomed.info/sct':
          return SpecialtySystemEnum
              .httpColonSlashSlashSnomedPeriodInfoSlashSct;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static SpecialtySystemEnumTypeTransformer? _instance;
}
