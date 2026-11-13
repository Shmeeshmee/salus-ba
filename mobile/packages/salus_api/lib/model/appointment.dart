//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Appointment {
  /// Returns a new [Appointment] instance.
  Appointment({
    required this.id,
    required this.organizationId,
    required this.practitionerRoleId,
    required this.slotId,
    required this.start,
    required this.end,
    required this.status,
    this.cancellationReason,
    this.cancelledAt,
    this.confirmedAt,
    this.confirmedVia,
    required this.bookedVia,
    required this.rescheduleCount,
    required this.versionId,
  });

  String id;

  String organizationId;

  String practitionerRoleId;

  String slotId;

  DateTime start;

  DateTime end;

  AppointmentStatusEnum status;

  /// http://terminology.hl7.org/CodeSystem/appointment-cancellation-reason: `pat` the patient (also by revoking consent), `prov` staff. Absent unless cancelled.
  AppointmentCancellationReasonEnum? cancellationReason;

  /// Absent unless cancelled.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? cancelledAt;

  /// When the patient confirmed it; absent while unconfirmed (docs/plans/appointments.md).
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  DateTime? confirmedAt;

  /// Where the patient confirmed it; absent while unconfirmed.
  AppointmentConfirmedViaEnum? confirmedVia;

  /// A free slot of the directory, or an offer of the waitlist.
  AppointmentBookedViaEnum bookedVia;

  /// How often the patient moved it.
  ///
  /// Minimum value: 0
  int rescheduleCount;

  int versionId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Appointment &&
          other.id == id &&
          other.organizationId == organizationId &&
          other.practitionerRoleId == practitionerRoleId &&
          other.slotId == slotId &&
          other.start == start &&
          other.end == end &&
          other.status == status &&
          other.cancellationReason == cancellationReason &&
          other.cancelledAt == cancelledAt &&
          other.confirmedAt == confirmedAt &&
          other.confirmedVia == confirmedVia &&
          other.bookedVia == bookedVia &&
          other.rescheduleCount == rescheduleCount &&
          other.versionId == versionId;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (id.hashCode) +
      (organizationId.hashCode) +
      (practitionerRoleId.hashCode) +
      (slotId.hashCode) +
      (start.hashCode) +
      (end.hashCode) +
      (status.hashCode) +
      (cancellationReason == null ? 0 : cancellationReason!.hashCode) +
      (cancelledAt == null ? 0 : cancelledAt!.hashCode) +
      (confirmedAt == null ? 0 : confirmedAt!.hashCode) +
      (confirmedVia == null ? 0 : confirmedVia!.hashCode) +
      (bookedVia.hashCode) +
      (rescheduleCount.hashCode) +
      (versionId.hashCode);

  @override
  String toString() =>
      'Appointment[id=$id, organizationId=$organizationId, practitionerRoleId=$practitionerRoleId, slotId=$slotId, start=$start, end=$end, status=$status, cancellationReason=$cancellationReason, cancelledAt=$cancelledAt, confirmedAt=$confirmedAt, confirmedVia=$confirmedVia, bookedVia=$bookedVia, rescheduleCount=$rescheduleCount, versionId=$versionId]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'id'] = this.id;
    json[r'organizationId'] = this.organizationId;
    json[r'practitionerRoleId'] = this.practitionerRoleId;
    json[r'slotId'] = this.slotId;
    json[r'start'] = this.start.toUtc().toIso8601String();
    json[r'end'] = this.end.toUtc().toIso8601String();
    json[r'status'] = this.status;
    if (this.cancellationReason != null) {
      json[r'cancellationReason'] = this.cancellationReason;
    } else {
      json[r'cancellationReason'] = null;
    }
    if (this.cancelledAt != null) {
      json[r'cancelledAt'] = this.cancelledAt!.toUtc().toIso8601String();
    } else {
      json[r'cancelledAt'] = null;
    }
    if (this.confirmedAt != null) {
      json[r'confirmedAt'] = this.confirmedAt!.toUtc().toIso8601String();
    } else {
      json[r'confirmedAt'] = null;
    }
    if (this.confirmedVia != null) {
      json[r'confirmedVia'] = this.confirmedVia;
    } else {
      json[r'confirmedVia'] = null;
    }
    json[r'bookedVia'] = this.bookedVia;
    json[r'rescheduleCount'] = this.rescheduleCount;
    json[r'versionId'] = this.versionId;
    return json;
  }

  /// Returns a new [Appointment] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Appointment? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'id'),
            'Required key "Appointment[id]" is missing from JSON.');
        assert(json[r'id'] != null,
            'Required key "Appointment[id]" has a null value in JSON.');
        assert(json.containsKey(r'organizationId'),
            'Required key "Appointment[organizationId]" is missing from JSON.');
        assert(json[r'organizationId'] != null,
            'Required key "Appointment[organizationId]" has a null value in JSON.');
        assert(json.containsKey(r'practitionerRoleId'),
            'Required key "Appointment[practitionerRoleId]" is missing from JSON.');
        assert(json[r'practitionerRoleId'] != null,
            'Required key "Appointment[practitionerRoleId]" has a null value in JSON.');
        assert(json.containsKey(r'slotId'),
            'Required key "Appointment[slotId]" is missing from JSON.');
        assert(json[r'slotId'] != null,
            'Required key "Appointment[slotId]" has a null value in JSON.');
        assert(json.containsKey(r'start'),
            'Required key "Appointment[start]" is missing from JSON.');
        assert(json[r'start'] != null,
            'Required key "Appointment[start]" has a null value in JSON.');
        assert(json.containsKey(r'end'),
            'Required key "Appointment[end]" is missing from JSON.');
        assert(json[r'end'] != null,
            'Required key "Appointment[end]" has a null value in JSON.');
        assert(json.containsKey(r'status'),
            'Required key "Appointment[status]" is missing from JSON.');
        assert(json[r'status'] != null,
            'Required key "Appointment[status]" has a null value in JSON.');
        assert(json.containsKey(r'bookedVia'),
            'Required key "Appointment[bookedVia]" is missing from JSON.');
        assert(json[r'bookedVia'] != null,
            'Required key "Appointment[bookedVia]" has a null value in JSON.');
        assert(json.containsKey(r'rescheduleCount'),
            'Required key "Appointment[rescheduleCount]" is missing from JSON.');
        assert(json[r'rescheduleCount'] != null,
            'Required key "Appointment[rescheduleCount]" has a null value in JSON.');
        assert(json.containsKey(r'versionId'),
            'Required key "Appointment[versionId]" is missing from JSON.');
        assert(json[r'versionId'] != null,
            'Required key "Appointment[versionId]" has a null value in JSON.');
        return true;
      }());

      return Appointment(
        id: mapValueOfType<String>(json, r'id')!,
        organizationId: mapValueOfType<String>(json, r'organizationId')!,
        practitionerRoleId:
            mapValueOfType<String>(json, r'practitionerRoleId')!,
        slotId: mapValueOfType<String>(json, r'slotId')!,
        start: mapDateTime(json, r'start', r'')!,
        end: mapDateTime(json, r'end', r'')!,
        status: AppointmentStatusEnum.fromJson(json[r'status'])!,
        cancellationReason: AppointmentCancellationReasonEnum.fromJson(
            json[r'cancellationReason']),
        cancelledAt: mapDateTime(json, r'cancelledAt', r''),
        confirmedAt: mapDateTime(json, r'confirmedAt', r''),
        confirmedVia:
            AppointmentConfirmedViaEnum.fromJson(json[r'confirmedVia']),
        bookedVia: AppointmentBookedViaEnum.fromJson(json[r'bookedVia'])!,
        rescheduleCount: mapValueOfType<int>(json, r'rescheduleCount')!,
        versionId: mapValueOfType<int>(json, r'versionId')!,
      );
    }
    return null;
  }

  static List<Appointment> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Appointment>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Appointment.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Appointment> mapFromJson(dynamic json) {
    final map = <String, Appointment>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Appointment.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Appointment-objects as value to a dart map
  static Map<String, List<Appointment>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Appointment>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Appointment.listFromJson(
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
    'organizationId',
    'practitionerRoleId',
    'slotId',
    'start',
    'end',
    'status',
    'bookedVia',
    'rescheduleCount',
    'versionId',
  };
}

enum AppointmentStatusEnum {
  booked._(r'booked'),
  cancelled._(r'cancelled'),
  fulfilled._(r'fulfilled'),
  noshow._(r'noshow'),
  ;

  /// Instantiate a new enum with the provided value.
  const AppointmentStatusEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [AppointmentStatusEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static AppointmentStatusEnum? fromJson(dynamic value) =>
      AppointmentStatusEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [AppointmentStatusEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<AppointmentStatusEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <AppointmentStatusEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = AppointmentStatusEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [AppointmentStatusEnum] to String,
/// and [decode] dynamic data back to [AppointmentStatusEnum].
class AppointmentStatusEnumTypeTransformer {
  factory AppointmentStatusEnumTypeTransformer() =>
      _instance ??= const AppointmentStatusEnumTypeTransformer._();

  const AppointmentStatusEnumTypeTransformer._();

  String encode(AppointmentStatusEnum data) => data._value;

  /// Returns the instance of [AppointmentStatusEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  AppointmentStatusEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is AppointmentStatusEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'booked':
          return AppointmentStatusEnum.booked;
        case r'cancelled':
          return AppointmentStatusEnum.cancelled;
        case r'fulfilled':
          return AppointmentStatusEnum.fulfilled;
        case r'noshow':
          return AppointmentStatusEnum.noshow;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static AppointmentStatusEnumTypeTransformer? _instance;
}

/// http://terminology.hl7.org/CodeSystem/appointment-cancellation-reason: `pat` the patient (also by revoking consent), `prov` staff. Absent unless cancelled.
enum AppointmentCancellationReasonEnum {
  pat._(r'pat'),
  prov._(r'prov'),
  ;

  /// Instantiate a new enum with the provided value.
  const AppointmentCancellationReasonEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [AppointmentCancellationReasonEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static AppointmentCancellationReasonEnum? fromJson(dynamic value) =>
      AppointmentCancellationReasonEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [AppointmentCancellationReasonEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<AppointmentCancellationReasonEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <AppointmentCancellationReasonEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = AppointmentCancellationReasonEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [AppointmentCancellationReasonEnum] to String,
/// and [decode] dynamic data back to [AppointmentCancellationReasonEnum].
class AppointmentCancellationReasonEnumTypeTransformer {
  factory AppointmentCancellationReasonEnumTypeTransformer() =>
      _instance ??= const AppointmentCancellationReasonEnumTypeTransformer._();

  const AppointmentCancellationReasonEnumTypeTransformer._();

  String encode(AppointmentCancellationReasonEnum data) => data._value;

  /// Returns the instance of [AppointmentCancellationReasonEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  AppointmentCancellationReasonEnum? decode(dynamic data,
      {bool allowNull = true}) {
    if (data is AppointmentCancellationReasonEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'pat':
          return AppointmentCancellationReasonEnum.pat;
        case r'prov':
          return AppointmentCancellationReasonEnum.prov;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static AppointmentCancellationReasonEnumTypeTransformer? _instance;
}

/// Where the patient confirmed it; absent while unconfirmed.
enum AppointmentConfirmedViaEnum {
  app._(r'app'),
  link._(r'link'),
  ;

  /// Instantiate a new enum with the provided value.
  const AppointmentConfirmedViaEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [AppointmentConfirmedViaEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static AppointmentConfirmedViaEnum? fromJson(dynamic value) =>
      AppointmentConfirmedViaEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [AppointmentConfirmedViaEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<AppointmentConfirmedViaEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <AppointmentConfirmedViaEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = AppointmentConfirmedViaEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [AppointmentConfirmedViaEnum] to String,
/// and [decode] dynamic data back to [AppointmentConfirmedViaEnum].
class AppointmentConfirmedViaEnumTypeTransformer {
  factory AppointmentConfirmedViaEnumTypeTransformer() =>
      _instance ??= const AppointmentConfirmedViaEnumTypeTransformer._();

  const AppointmentConfirmedViaEnumTypeTransformer._();

  String encode(AppointmentConfirmedViaEnum data) => data._value;

  /// Returns the instance of [AppointmentConfirmedViaEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  AppointmentConfirmedViaEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is AppointmentConfirmedViaEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'app':
          return AppointmentConfirmedViaEnum.app;
        case r'link':
          return AppointmentConfirmedViaEnum.link;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static AppointmentConfirmedViaEnumTypeTransformer? _instance;
}

/// A free slot of the directory, or an offer of the waitlist.
enum AppointmentBookedViaEnum {
  direct._(r'direct'),
  waitlist._(r'waitlist'),
  ;

  /// Instantiate a new enum with the provided value.
  const AppointmentBookedViaEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [AppointmentBookedViaEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static AppointmentBookedViaEnum? fromJson(dynamic value) =>
      AppointmentBookedViaEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [AppointmentBookedViaEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<AppointmentBookedViaEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <AppointmentBookedViaEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = AppointmentBookedViaEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [AppointmentBookedViaEnum] to String,
/// and [decode] dynamic data back to [AppointmentBookedViaEnum].
class AppointmentBookedViaEnumTypeTransformer {
  factory AppointmentBookedViaEnumTypeTransformer() =>
      _instance ??= const AppointmentBookedViaEnumTypeTransformer._();

  const AppointmentBookedViaEnumTypeTransformer._();

  String encode(AppointmentBookedViaEnum data) => data._value;

  /// Returns the instance of [AppointmentBookedViaEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  AppointmentBookedViaEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is AppointmentBookedViaEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'direct':
          return AppointmentBookedViaEnum.direct;
        case r'waitlist':
          return AppointmentBookedViaEnum.waitlist;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static AppointmentBookedViaEnumTypeTransformer? _instance;
}
