//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SuggestionStatus {
  /// Returns a new [SuggestionStatus] instance.
  SuggestionStatus({
    required this.id,
    required this.status,
  });

  String id;

  SuggestionStatusStatusEnum status;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SuggestionStatus && other.id == id && other.status == status;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) + (status.hashCode);

  @override
  String toString() => 'SuggestionStatus[id=$id, status=$status]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'status'] = this.status;
    return json;
  }

  /// Returns a new [SuggestionStatus] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SuggestionStatus? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "SuggestionStatus[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "SuggestionStatus[id]" has a null value in JSON.');
        assert(json.containsKey(r'status'),
            'Required key "SuggestionStatus[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "SuggestionStatus[status]" has a null value in JSON.');
        return true;
      }());

      return SuggestionStatus(
        id: mapValueOfType<String>(json, r'id')!,
        status: SuggestionStatusStatusEnum.fromJson(json[r'status'])!,
      );
    }
    return null;
  }

  static List<SuggestionStatus> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SuggestionStatus>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SuggestionStatus.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SuggestionStatus> mapFromJson(dynamic json) {
    final map = <String, SuggestionStatus>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SuggestionStatus.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SuggestionStatus-objects as value to a dart map
  static Map<String, List<SuggestionStatus>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SuggestionStatus>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SuggestionStatus.listFromJson(
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
    'status',
  };
}

enum SuggestionStatusStatusEnum {
  shown._(r'shown'),
  dismissed._(r'dismissed'),
  pinned._(r'pinned'),
  ;

  /// Instantiate a new enum with the provided value.
  const SuggestionStatusStatusEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [SuggestionStatusStatusEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static SuggestionStatusStatusEnum? fromJson(dynamic value) =>
      SuggestionStatusStatusEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [SuggestionStatusStatusEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<SuggestionStatusStatusEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SuggestionStatusStatusEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SuggestionStatusStatusEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [SuggestionStatusStatusEnum] to String,
/// and [decode] dynamic data back to [SuggestionStatusStatusEnum].
class SuggestionStatusStatusEnumTypeTransformer {
  factory SuggestionStatusStatusEnumTypeTransformer() =>
      _instance ??= const SuggestionStatusStatusEnumTypeTransformer._();

  const SuggestionStatusStatusEnumTypeTransformer._();

  String encode(SuggestionStatusStatusEnum data) => data._value;

  /// Returns the instance of [SuggestionStatusStatusEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  SuggestionStatusStatusEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is SuggestionStatusStatusEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'shown':
          return SuggestionStatusStatusEnum.shown;
        case r'dismissed':
          return SuggestionStatusStatusEnum.dismissed;
        case r'pinned':
          return SuggestionStatusStatusEnum.pinned;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static SuggestionStatusStatusEnumTypeTransformer? _instance;
}
