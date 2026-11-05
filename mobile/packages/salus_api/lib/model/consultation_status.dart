//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ConsultationStatus {
  /// Returns a new [ConsultationStatus] instance.
  ConsultationStatus({
    required this.state,
    this.patientJoinedAt,
    this.practitionerJoinedAt,
    this.endedAt,
    this.endReason,
    this.reason,
  });

  ConsultationStatusStateEnum state;

  /// When the patient first joined the room.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? patientJoinedAt;

  /// When the practitioner first joined the room.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? practitionerJoinedAt;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? endedAt;

  /// Why it ended: the practitioner ended it; the window closed with the room empty; the practitioner never joined; the 2-hour limit after the scheduled end; the appointment was cancelled.
  ConsultationStatusEndReasonEnum? endReason;

  /// Why the patient was removed or the consultation ended, as the clients show it: the patient was removed because their consent was withdrawn (also when that cancelled the appointment), or because the appointment was cancelled; while open, the patient who joined left on their own; the practitioner ended it or the window closed with the room empty; the window closed without the practitioner; the 2-hour limit. Absent while nobody has joined or left.
  ConsultationStatusReasonEnum? reason;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ConsultationStatus &&
          other.state == state &&
          other.patientJoinedAt == patientJoinedAt &&
          other.practitionerJoinedAt == practitionerJoinedAt &&
          other.endedAt == endedAt &&
          other.endReason == endReason &&
          other.reason == reason;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (state.hashCode) +
      (patientJoinedAt == null ? 0 : patientJoinedAt!.hashCode) +
      (practitionerJoinedAt == null ? 0 : practitionerJoinedAt!.hashCode) +
      (endedAt == null ? 0 : endedAt!.hashCode) +
      (endReason == null ? 0 : endReason!.hashCode) +
      (reason == null ? 0 : reason!.hashCode);

  @override
  String toString() =>
      'ConsultationStatus[state=$state, patientJoinedAt=$patientJoinedAt, practitionerJoinedAt=$practitionerJoinedAt, endedAt=$endedAt, endReason=$endReason, reason=$reason]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'state'] = this.state;
    if (this.patientJoinedAt != null) {
      json[r'patientJoinedAt'] =
          this.patientJoinedAt!.toUtc().toIso8601String();
    } else {
      json[r'patientJoinedAt'] = null;
    }
    if (this.practitionerJoinedAt != null) {
      json[r'practitionerJoinedAt'] =
          this.practitionerJoinedAt!.toUtc().toIso8601String();
    } else {
      json[r'practitionerJoinedAt'] = null;
    }
    if (this.endedAt != null) {
      json[r'endedAt'] = this.endedAt!.toUtc().toIso8601String();
    } else {
      json[r'endedAt'] = null;
    }
    if (this.endReason != null) {
      json[r'endReason'] = this.endReason;
    } else {
      json[r'endReason'] = null;
    }
    if (this.reason != null) {
      json[r'reason'] = this.reason;
    } else {
      json[r'reason'] = null;
    }
    return json;
  }

  /// Returns a new [ConsultationStatus] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ConsultationStatus? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'state'),
            'Required key "ConsultationStatus[state]" is missing from JSON.');
        assert(json[r'state'] != null,
            'Required key "ConsultationStatus[state]" has a null value in JSON.');
        return true;
      }());

      return ConsultationStatus(
        state: ConsultationStatusStateEnum.fromJson(json[r'state'])!,
        patientJoinedAt: mapDateTime(json, r'patientJoinedAt', r''),
        practitionerJoinedAt: mapDateTime(json, r'practitionerJoinedAt', r''),
        endedAt: mapDateTime(json, r'endedAt', r''),
        endReason: ConsultationStatusEndReasonEnum.fromJson(json[r'endReason']),
        reason: ConsultationStatusReasonEnum.fromJson(json[r'reason']),
      );
    }
    return null;
  }

  static List<ConsultationStatus> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ConsultationStatus>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ConsultationStatus.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ConsultationStatus> mapFromJson(dynamic json) {
    final map = <String, ConsultationStatus>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ConsultationStatus.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ConsultationStatus-objects as value to a dart map
  static Map<String, List<ConsultationStatus>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<ConsultationStatus>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ConsultationStatus.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'state',
  };
}

enum ConsultationStatusStateEnum {
  notStarted._(r'not_started'),
  open._(r'open'),
  ended._(r'ended'),
  ;

  /// Instantiate a new enum with the provided value.
  const ConsultationStatusStateEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ConsultationStatusStateEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ConsultationStatusStateEnum? fromJson(dynamic value) =>
      ConsultationStatusStateEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ConsultationStatusStateEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ConsultationStatusStateEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ConsultationStatusStateEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ConsultationStatusStateEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ConsultationStatusStateEnum] to String,
/// and [decode] dynamic data back to [ConsultationStatusStateEnum].
class ConsultationStatusStateEnumTypeTransformer {
  factory ConsultationStatusStateEnumTypeTransformer() =>
      _instance ??= const ConsultationStatusStateEnumTypeTransformer._();

  const ConsultationStatusStateEnumTypeTransformer._();

  String encode(ConsultationStatusStateEnum data) => data._value;

  /// Returns the instance of [ConsultationStatusStateEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ConsultationStatusStateEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ConsultationStatusStateEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'not_started':
          return ConsultationStatusStateEnum.notStarted;
        case r'open':
          return ConsultationStatusStateEnum.open;
        case r'ended':
          return ConsultationStatusStateEnum.ended;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ConsultationStatusStateEnumTypeTransformer? _instance;
}

/// Why it ended: the practitioner ended it; the window closed with the room empty; the practitioner never joined; the 2-hour limit after the scheduled end; the appointment was cancelled.
enum ConsultationStatusEndReasonEnum {
  practitioner._(r'practitioner'),
  windowClosed._(r'window_closed'),
  practitionerAbsent._(r'practitioner_absent'),
  timeLimit._(r'time_limit'),
  cancelled._(r'cancelled'),
  ;

  /// Instantiate a new enum with the provided value.
  const ConsultationStatusEndReasonEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ConsultationStatusEndReasonEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ConsultationStatusEndReasonEnum? fromJson(dynamic value) =>
      ConsultationStatusEndReasonEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ConsultationStatusEndReasonEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ConsultationStatusEndReasonEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ConsultationStatusEndReasonEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ConsultationStatusEndReasonEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ConsultationStatusEndReasonEnum] to String,
/// and [decode] dynamic data back to [ConsultationStatusEndReasonEnum].
class ConsultationStatusEndReasonEnumTypeTransformer {
  factory ConsultationStatusEndReasonEnumTypeTransformer() =>
      _instance ??= const ConsultationStatusEndReasonEnumTypeTransformer._();

  const ConsultationStatusEndReasonEnumTypeTransformer._();

  String encode(ConsultationStatusEndReasonEnum data) => data._value;

  /// Returns the instance of [ConsultationStatusEndReasonEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ConsultationStatusEndReasonEnum? decode(dynamic data,
      {bool allowNull = true}) {
    if (data is ConsultationStatusEndReasonEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'practitioner':
          return ConsultationStatusEndReasonEnum.practitioner;
        case r'window_closed':
          return ConsultationStatusEndReasonEnum.windowClosed;
        case r'practitioner_absent':
          return ConsultationStatusEndReasonEnum.practitionerAbsent;
        case r'time_limit':
          return ConsultationStatusEndReasonEnum.timeLimit;
        case r'cancelled':
          return ConsultationStatusEndReasonEnum.cancelled;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ConsultationStatusEndReasonEnumTypeTransformer? _instance;
}

/// Why the patient was removed or the consultation ended, as the clients show it: the patient was removed because their consent was withdrawn (also when that cancelled the appointment), or because the appointment was cancelled; while open, the patient who joined left on their own; the practitioner ended it or the window closed with the room empty; the window closed without the practitioner; the 2-hour limit. Absent while nobody has joined or left.
enum ConsultationStatusReasonEnum {
  consentWithdrawn._(r'consent_withdrawn'),
  cancelled._(r'cancelled'),
  left._(r'left'),
  ended._(r'ended'),
  practitionerDidNotJoin._(r'practitioner_did_not_join'),
  timeLimit._(r'time_limit'),
  ;

  /// Instantiate a new enum with the provided value.
  const ConsultationStatusReasonEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ConsultationStatusReasonEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ConsultationStatusReasonEnum? fromJson(dynamic value) =>
      ConsultationStatusReasonEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ConsultationStatusReasonEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ConsultationStatusReasonEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ConsultationStatusReasonEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ConsultationStatusReasonEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ConsultationStatusReasonEnum] to String,
/// and [decode] dynamic data back to [ConsultationStatusReasonEnum].
class ConsultationStatusReasonEnumTypeTransformer {
  factory ConsultationStatusReasonEnumTypeTransformer() =>
      _instance ??= const ConsultationStatusReasonEnumTypeTransformer._();

  const ConsultationStatusReasonEnumTypeTransformer._();

  String encode(ConsultationStatusReasonEnum data) => data._value;

  /// Returns the instance of [ConsultationStatusReasonEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ConsultationStatusReasonEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ConsultationStatusReasonEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'consent_withdrawn':
          return ConsultationStatusReasonEnum.consentWithdrawn;
        case r'cancelled':
          return ConsultationStatusReasonEnum.cancelled;
        case r'left':
          return ConsultationStatusReasonEnum.left;
        case r'ended':
          return ConsultationStatusReasonEnum.ended;
        case r'practitioner_did_not_join':
          return ConsultationStatusReasonEnum.practitionerDidNotJoin;
        case r'time_limit':
          return ConsultationStatusReasonEnum.timeLimit;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ConsultationStatusReasonEnumTypeTransformer? _instance;
}
