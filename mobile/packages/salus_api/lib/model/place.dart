//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Place {
  /// Returns a new [Place] instance.
  Place({
    required this.id,
    required this.kind,
    this.parent,
    required this.nameLatn,
    required this.nameCyrl,
  });

  /// An ISO 3166-2:BA code (entity, region) or a Wikidata id (city).
  String id;

  PlaceKindEnum kind;

  /// Absent for the entity and for a region that is its own entity.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? parent;

  String nameLatn;

  String nameCyrl;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Place &&
          other.id == id &&
          other.kind == kind &&
          other.parent == parent &&
          other.nameLatn == nameLatn &&
          other.nameCyrl == nameCyrl;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (kind.hashCode) +
      (parent == null ? 0 : parent!.hashCode) +
      (nameLatn.hashCode) +
      (nameCyrl.hashCode);

  @override
  String toString() =>
      'Place[id=$id, kind=$kind, parent=$parent, nameLatn=$nameLatn, nameCyrl=$nameCyrl]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'kind'] = this.kind;
    if (this.parent != null) {
      json[r'parent'] = this.parent;
    } else {
      json[r'parent'] = null;
    }
    json[r'nameLatn'] = this.nameLatn;
    json[r'nameCyrl'] = this.nameCyrl;
    return json;
  }

  /// Returns a new [Place] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Place? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "Place[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "Place[id]" has a null value in JSON.');
        assert(json.containsKey(r'kind'),
            'Required key "Place[kind]" is missing from JSON.');
        assert(json[r'kind'] != null,
            'Required key "Place[kind]" has a null value in JSON.');
        assert(json.containsKey(r'nameLatn'),
            'Required key "Place[nameLatn]" is missing from JSON.');
        assert(json[r'nameLatn'] != null,
            'Required key "Place[nameLatn]" has a null value in JSON.');
        assert(json.containsKey(r'nameCyrl'),
            'Required key "Place[nameCyrl]" is missing from JSON.');
        assert(json[r'nameCyrl'] != null,
            'Required key "Place[nameCyrl]" has a null value in JSON.');
        return true;
      }());

      return Place(
        id: mapValueOfType<String>(json, r'id')!,
        kind: PlaceKindEnum.fromJson(json[r'kind'])!,
        parent: mapValueOfType<String>(json, r'parent'),
        nameLatn: mapValueOfType<String>(json, r'nameLatn')!,
        nameCyrl: mapValueOfType<String>(json, r'nameCyrl')!,
      );
    }
    return null;
  }

  static List<Place> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Place>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Place.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Place> mapFromJson(dynamic json) {
    final map = <String, Place>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Place.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Place-objects as value to a dart map
  static Map<String, List<Place>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Place>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Place.listFromJson(
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
    'nameLatn',
    'nameCyrl',
  };
}

enum PlaceKindEnum {
  entity._(r'entity'),
  region._(r'region'),
  city._(r'city'),
  ;

  /// Instantiate a new enum with the provided value.
  const PlaceKindEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [PlaceKindEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static PlaceKindEnum? fromJson(dynamic value) =>
      PlaceKindEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [PlaceKindEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<PlaceKindEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <PlaceKindEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = PlaceKindEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [PlaceKindEnum] to String,
/// and [decode] dynamic data back to [PlaceKindEnum].
class PlaceKindEnumTypeTransformer {
  factory PlaceKindEnumTypeTransformer() =>
      _instance ??= const PlaceKindEnumTypeTransformer._();

  const PlaceKindEnumTypeTransformer._();

  String encode(PlaceKindEnum data) => data._value;

  /// Returns the instance of [PlaceKindEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  PlaceKindEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is PlaceKindEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'entity':
          return PlaceKindEnum.entity;
        case r'region':
          return PlaceKindEnum.region;
        case r'city':
          return PlaceKindEnum.city;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static PlaceKindEnumTypeTransformer? _instance;
}
