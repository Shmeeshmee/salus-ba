//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class AppointmentChange {
  /// Returns a new [AppointmentChange] instance.
  AppointmentChange({
    required this.at,
    required this.kind,
    required this.via,
    this.fromStart,
    this.fromEnd,
    this.toStart,
    this.toEnd,
    this.fromStatus,
    this.toStatus,
  });

  DateTime at;

  AppointmentChangeKind kind;

  AppointmentChangeVia via;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? fromStart;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? fromEnd;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? toStart;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? toEnd;

  AppointmentChangeFromStatusEnum? fromStatus;

  AppointmentChangeToStatusEnum? toStatus;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is AppointmentChange &&
          other.at == at &&
          other.kind == kind &&
          other.via == via &&
          other.fromStart == fromStart &&
          other.fromEnd == fromEnd &&
          other.toStart == toStart &&
          other.toEnd == toEnd &&
          other.fromStatus == fromStatus &&
          other.toStatus == toStatus;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (at.hashCode) +
      (kind.hashCode) +
      (via.hashCode) +
      (fromStart == null ? 0 : fromStart!.hashCode) +
      (fromEnd == null ? 0 : fromEnd!.hashCode) +
      (toStart == null ? 0 : toStart!.hashCode) +
      (toEnd == null ? 0 : toEnd!.hashCode) +
      (fromStatus == null ? 0 : fromStatus!.hashCode) +
      (toStatus == null ? 0 : toStatus!.hashCode);

  @override
  String toString() =>
      'AppointmentChange[at=$at, kind=$kind, via=$via, fromStart=$fromStart, fromEnd=$fromEnd, toStart=$toStart, toEnd=$toEnd, fromStatus=$fromStatus, toStatus=$toStatus]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'at'] = this.at.toUtc().toIso8601String();
    json[r'kind'] = this.kind;
    json[r'via'] = this.via;
    if (this.fromStart != null) {
      json[r'from_start'] = this.fromStart!.toUtc().toIso8601String();
    } else {
      json[r'from_start'] = null;
    }
    if (this.fromEnd != null) {
      json[r'from_end'] = this.fromEnd!.toUtc().toIso8601String();
    } else {
      json[r'from_end'] = null;
    }
    if (this.toStart != null) {
      json[r'to_start'] = this.toStart!.toUtc().toIso8601String();
    } else {
      json[r'to_start'] = null;
    }
    if (this.toEnd != null) {
      json[r'to_end'] = this.toEnd!.toUtc().toIso8601String();
    } else {
      json[r'to_end'] = null;
    }
    if (this.fromStatus != null) {
      json[r'from_status'] = this.fromStatus;
    } else {
      json[r'from_status'] = null;
    }
    if (this.toStatus != null) {
      json[r'to_status'] = this.toStatus;
    } else {
      json[r'to_status'] = null;
    }
    return json;
  }

  /// Returns a new [AppointmentChange] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static AppointmentChange? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'at'),
            'Required key "AppointmentChange[at]" is missing from JSON.');
        assert(json[r'at'] != null,
            'Required key "AppointmentChange[at]" has a null value in JSON.');
        assert(json.containsKey(r'kind'),
            'Required key "AppointmentChange[kind]" is missing from JSON.');
        assert(json[r'kind'] != null,
            'Required key "AppointmentChange[kind]" has a null value in JSON.');
        assert(json.containsKey(r'via'),
            'Required key "AppointmentChange[via]" is missing from JSON.');
        assert(json[r'via'] != null,
            'Required key "AppointmentChange[via]" has a null value in JSON.');
        return true;
      }());

      return AppointmentChange(
        at: mapDateTime(json, r'at', r'')!,
        kind: AppointmentChangeKind.fromJson(json[r'kind'])!,
        via: AppointmentChangeVia.fromJson(json[r'via'])!,
        fromStart: mapDateTime(json, r'from_start', r''),
        fromEnd: mapDateTime(json, r'from_end', r''),
        toStart: mapDateTime(json, r'to_start', r''),
        toEnd: mapDateTime(json, r'to_end', r''),
        fromStatus:
            AppointmentChangeFromStatusEnum.fromJson(json[r'from_status']),
        toStatus: AppointmentChangeToStatusEnum.fromJson(json[r'to_status']),
      );
    }
    return null;
  }

  static List<AppointmentChange> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <AppointmentChange>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = AppointmentChange.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, AppointmentChange> mapFromJson(dynamic json) {
    final map = <String, AppointmentChange>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = AppointmentChange.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of AppointmentChange-objects as value to a dart map
  static Map<String, List<AppointmentChange>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<AppointmentChange>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = AppointmentChange.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'at',
    'kind',
    'via',
  };
}

enum AppointmentChangeFromStatusEnum {
  booked._(r'booked'),
  cancelled._(r'cancelled'),
  fulfilled._(r'fulfilled'),
  noshow._(r'noshow'),
  ;

  /// Instantiate a new enum with the provided value.
  const AppointmentChangeFromStatusEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [AppointmentChangeFromStatusEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static AppointmentChangeFromStatusEnum? fromJson(dynamic value) =>
      AppointmentChangeFromStatusEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [AppointmentChangeFromStatusEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<AppointmentChangeFromStatusEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <AppointmentChangeFromStatusEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = AppointmentChangeFromStatusEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [AppointmentChangeFromStatusEnum] to String,
/// and [decode] dynamic data back to [AppointmentChangeFromStatusEnum].
class AppointmentChangeFromStatusEnumTypeTransformer {
  factory AppointmentChangeFromStatusEnumTypeTransformer() =>
      _instance ??= const AppointmentChangeFromStatusEnumTypeTransformer._();

  const AppointmentChangeFromStatusEnumTypeTransformer._();

  String encode(AppointmentChangeFromStatusEnum data) => data._value;

  /// Returns the instance of [AppointmentChangeFromStatusEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  AppointmentChangeFromStatusEnum? decode(dynamic data,
      {bool allowNull = true}) {
    if (data is AppointmentChangeFromStatusEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'booked':
          return AppointmentChangeFromStatusEnum.booked;
        case r'cancelled':
          return AppointmentChangeFromStatusEnum.cancelled;
        case r'fulfilled':
          return AppointmentChangeFromStatusEnum.fulfilled;
        case r'noshow':
          return AppointmentChangeFromStatusEnum.noshow;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static AppointmentChangeFromStatusEnumTypeTransformer? _instance;
}

enum AppointmentChangeToStatusEnum {
  booked._(r'booked'),
  cancelled._(r'cancelled'),
  fulfilled._(r'fulfilled'),
  noshow._(r'noshow'),
  ;

  /// Instantiate a new enum with the provided value.
  const AppointmentChangeToStatusEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [AppointmentChangeToStatusEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static AppointmentChangeToStatusEnum? fromJson(dynamic value) =>
      AppointmentChangeToStatusEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [AppointmentChangeToStatusEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<AppointmentChangeToStatusEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <AppointmentChangeToStatusEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = AppointmentChangeToStatusEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [AppointmentChangeToStatusEnum] to String,
/// and [decode] dynamic data back to [AppointmentChangeToStatusEnum].
class AppointmentChangeToStatusEnumTypeTransformer {
  factory AppointmentChangeToStatusEnumTypeTransformer() =>
      _instance ??= const AppointmentChangeToStatusEnumTypeTransformer._();

  const AppointmentChangeToStatusEnumTypeTransformer._();

  String encode(AppointmentChangeToStatusEnum data) => data._value;

  /// Returns the instance of [AppointmentChangeToStatusEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  AppointmentChangeToStatusEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is AppointmentChangeToStatusEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'booked':
          return AppointmentChangeToStatusEnum.booked;
        case r'cancelled':
          return AppointmentChangeToStatusEnum.cancelled;
        case r'fulfilled':
          return AppointmentChangeToStatusEnum.fulfilled;
        case r'noshow':
          return AppointmentChangeToStatusEnum.noshow;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static AppointmentChangeToStatusEnumTypeTransformer? _instance;
}
