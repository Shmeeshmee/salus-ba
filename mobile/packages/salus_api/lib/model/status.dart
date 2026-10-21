//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Status {
  /// Returns a new [Status] instance.
  Status({
    required this.status,
  });

  StatusStatusEnum status;

  @override
  bool operator ==(Object other) =>
      identical(this, other) || other is Status && other.status == status;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (status.hashCode);

  @override
  String toString() => 'Status[status=$status]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'status'] = this.status;
    return json;
  }

  /// Returns a new [Status] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Status? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'status'),
            'Required key "Status[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "Status[status]" has a null value in JSON.');
        return true;
      }());

      return Status(
        status: StatusStatusEnum.fromJson(json[r'status'])!,
      );
    }
    return null;
  }

  static List<Status> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Status>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Status.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Status> mapFromJson(dynamic json) {
    final map = <String, Status>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Status.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Status-objects as value to a dart map
  static Map<String, List<Status>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Status>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Status.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'status',
  };
}

enum StatusStatusEnum {
  ok._(r'ok'),
  ;

  /// Instantiate a new enum with the provided value.
  const StatusStatusEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [StatusStatusEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static StatusStatusEnum? fromJson(dynamic value) =>
      StatusStatusEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [StatusStatusEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<StatusStatusEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <StatusStatusEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = StatusStatusEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [StatusStatusEnum] to String,
/// and [decode] dynamic data back to [StatusStatusEnum].
class StatusStatusEnumTypeTransformer {
  factory StatusStatusEnumTypeTransformer() =>
      _instance ??= const StatusStatusEnumTypeTransformer._();

  const StatusStatusEnumTypeTransformer._();

  String encode(StatusStatusEnum data) => data._value;

  /// Returns the instance of [StatusStatusEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  StatusStatusEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is StatusStatusEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'ok':
          return StatusStatusEnum.ok;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static StatusStatusEnumTypeTransformer? _instance;
}
