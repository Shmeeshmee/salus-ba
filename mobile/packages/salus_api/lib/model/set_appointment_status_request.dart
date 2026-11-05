//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SetAppointmentStatusRequest {
  /// Returns a new [SetAppointmentStatusRequest] instance.
  SetAppointmentStatusRequest({
    required this.status,
  });

  SetAppointmentStatusRequestStatusEnum status;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SetAppointmentStatusRequest && other.status == status;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (status.hashCode);

  @override
  String toString() => 'SetAppointmentStatusRequest[status=$status]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'status'] = this.status;
    return json;
  }

  /// Returns a new [SetAppointmentStatusRequest] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static SetAppointmentStatusRequest? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'status'),
            'Required key "SetAppointmentStatusRequest[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "SetAppointmentStatusRequest[status]" has a null value in JSON.');
        return true;
      }());

      return SetAppointmentStatusRequest(
        status:
            SetAppointmentStatusRequestStatusEnum.fromJson(json[r'status'])!,
      );
    }
    return null;
  }

  static List<SetAppointmentStatusRequest> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SetAppointmentStatusRequest>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SetAppointmentStatusRequest.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, SetAppointmentStatusRequest> mapFromJson(dynamic json) {
    final map = <String, SetAppointmentStatusRequest>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = SetAppointmentStatusRequest.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of SetAppointmentStatusRequest-objects as value to a dart map
  static Map<String, List<SetAppointmentStatusRequest>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<SetAppointmentStatusRequest>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = SetAppointmentStatusRequest.listFromJson(
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

enum SetAppointmentStatusRequestStatusEnum {
  fulfilled._(r'fulfilled'),
  noshow._(r'noshow'),
  ;

  /// Instantiate a new enum with the provided value.
  const SetAppointmentStatusRequestStatusEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [SetAppointmentStatusRequestStatusEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static SetAppointmentStatusRequestStatusEnum? fromJson(dynamic value) =>
      SetAppointmentStatusRequestStatusEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [SetAppointmentStatusRequestStatusEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<SetAppointmentStatusRequestStatusEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <SetAppointmentStatusRequestStatusEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = SetAppointmentStatusRequestStatusEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [SetAppointmentStatusRequestStatusEnum] to String,
/// and [decode] dynamic data back to [SetAppointmentStatusRequestStatusEnum].
class SetAppointmentStatusRequestStatusEnumTypeTransformer {
  factory SetAppointmentStatusRequestStatusEnumTypeTransformer() =>
      _instance ??=
          const SetAppointmentStatusRequestStatusEnumTypeTransformer._();

  const SetAppointmentStatusRequestStatusEnumTypeTransformer._();

  String encode(SetAppointmentStatusRequestStatusEnum data) => data._value;

  /// Returns the instance of [SetAppointmentStatusRequestStatusEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  SetAppointmentStatusRequestStatusEnum? decode(dynamic data,
      {bool allowNull = true}) {
    if (data is SetAppointmentStatusRequestStatusEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'fulfilled':
          return SetAppointmentStatusRequestStatusEnum.fulfilled;
        case r'noshow':
          return SetAppointmentStatusRequestStatusEnum.noshow;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static SetAppointmentStatusRequestStatusEnumTypeTransformer? _instance;
}
