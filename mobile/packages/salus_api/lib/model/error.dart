//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class Error {
  /// Returns a new [Error] instance.
  Error({
    required this.error,
    this.linkIds = const [],
    this.language,
  });

  ErrorErrorEnum error;

  /// With `invalid_questionnaire`, `answers_invalid` and `translation_incomplete` only, the offending items' linkIds in document order.
  List<String> linkIds;

  /// With `translation_incomplete` only, the translation that is incomplete.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  Language? language;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Error &&
          other.error == error &&
          _deepEquality.equals(other.linkIds, linkIds) &&
          other.language == language;

  @override
  int get hashCode =>
      // ignore: unnecessary_parenthesis
      (error.hashCode) +
      (linkIds.hashCode) +
      (language == null ? 0 : language!.hashCode);

  @override
  String toString() =>
      'Error[error=$error, linkIds=$linkIds, language=$language]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
    json[r'error'] = this.error;
    json[r'linkIds'] = this.linkIds;
    if (this.language != null) {
      json[r'language'] = this.language;
    } else {
      json[r'language'] = null;
    }
    return json;
  }

  /// Returns a new [Error] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static Error? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        assert(json.containsKey(r'error'),
            'Required key "Error[error]" is missing from JSON.');
        assert(json[r'error'] != null,
            'Required key "Error[error]" has a null value in JSON.');
        return true;
      }());

      return Error(
        error: ErrorErrorEnum.fromJson(json[r'error'])!,
        linkIds: json[r'linkIds'] is Iterable
            ? (json[r'linkIds'] as Iterable)
                .cast<String>()
                .toList(growable: false)
            : const [],
        language: Language.fromJson(json[r'language']),
      );
    }
    return null;
  }

  static List<Error> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <Error>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = Error.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, Error> mapFromJson(dynamic json) {
    final map = <String, Error>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = Error.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of Error-objects as value to a dart map
  static Map<String, List<Error>> mapListFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final map = <String, List<Error>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = Error.listFromJson(
          entry.value,
          growable: growable,
        );
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'error',
  };
}

enum ErrorErrorEnum {
  invalidRequest._(r'invalid_request'),
  invalidEmail._(r'invalid_email'),
  invalidPassword._(r'invalid_password'),
  emailTaken._(r'email_taken'),
  unauthenticated._(r'unauthenticated'),
  wrongCredentials._(r'wrong_credentials'),
  rateLimited._(r'rate_limited'),
  invalidChallenge._(r'invalid_challenge'),
  wrongCode._(r'wrong_code'),
  alreadyEnrolled._(r'already_enrolled'),
  notEnrolled._(r'not_enrolled'),
  csrf._(r'csrf'),
  forbidden._(r'forbidden'),
  invalidRole._(r'invalid_role'),
  invalidName._(r'invalid_name'),
  invalidInvite._(r'invalid_invite'),
  invalidBirthDate._(r'invalid_birth_date'),
  invalidPhone._(r'invalid_phone'),
  notFound._(r'not_found'),
  versionConflict._(r'version_conflict'),
  consentActive._(r'consent_active'),
  consentInactive._(r'consent_inactive'),
  profileRequired._(r'profile_required'),
  invalidSpecialty._(r'invalid_specialty'),
  invalidSlotMinutes._(r'invalid_slot_minutes'),
  invalidAvailability._(r'invalid_availability'),
  consentRequired._(r'consent_required'),
  slotUnavailable._(r'slot_unavailable'),
  cancellationWindowClosed._(r'cancellation_window_closed'),
  appointmentClosed._(r'appointment_closed'),
  appointmentNotStarted._(r'appointment_not_started'),
  invalidStatus._(r'invalid_status'),
  joinWindowNotOpen._(r'join_window_not_open'),
  consultationEnded._(r'consultation_ended'),
  practitionerDidNotJoin._(r'practitioner_did_not_join'),
  consultationNotStarted._(r'consultation_not_started'),
  notDraft._(r'not_draft'),
  notActive._(r'not_active'),
  draftExists._(r'draft_exists'),
  invalidQuestionnaire._(r'invalid_questionnaire'),
  alreadyRequested._(r'already_requested'),
  appointmentStarted._(r'appointment_started'),
  submitted._(r'submitted'),
  withdrawn._(r'withdrawn'),
  pastDue._(r'past_due'),
  answersInvalid._(r'answers_invalid'),
  notAttachmentItem._(r'not_attachment_item'),
  lengthRequired._(r'length_required'),
  tooLarge._(r'too_large'),
  responseFull._(r'response_full'),
  tooManyFiles._(r'too_many_files'),
  unsupportedType._(r'unsupported_type'),
  scanning._(r'scanning'),
  rangeNotSatisfiable._(r'range_not_satisfiable'),
  attachmentsPending._(r'attachments_pending'),
  languageUnavailable._(r'language_unavailable'),
  languageLocked._(r'language_locked'),
  translationIncomplete._(r'translation_incomplete'),
  internalError._(r'internal error'),
  timeout._(r'timeout'),
  ;

  /// Instantiate a new enum with the provided value.
  const ErrorErrorEnum._(this._value);

  /// The underlying value of this enum member.
  final String _value;

  @override
  String toString() => _value;

  /// Encodes this enum as a value suitable for JSON.
  String toJson() => _value;

  /// Returns the instance of [ErrorErrorEnum] that was successfully decoded
  /// from the passed [value] on success, null otherwise.
  static ErrorErrorEnum? fromJson(dynamic value) =>
      ErrorErrorEnumTypeTransformer().decode(value);

  /// Returns a [List] containing instances of [ErrorErrorEnum]
  /// that were successfully decoded from the passed [JSON][json].
  static List<ErrorErrorEnum> listFromJson(
    dynamic json, {
    bool growable = false,
  }) {
    final result = <ErrorErrorEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ErrorErrorEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [ErrorErrorEnum] to String,
/// and [decode] dynamic data back to [ErrorErrorEnum].
class ErrorErrorEnumTypeTransformer {
  factory ErrorErrorEnumTypeTransformer() =>
      _instance ??= const ErrorErrorEnumTypeTransformer._();

  const ErrorErrorEnumTypeTransformer._();

  String encode(ErrorErrorEnum data) => data._value;

  /// Returns the instance of [ErrorErrorEnum] that was successfully decoded
  /// from the passed [data] value on success, null otherwise.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  ErrorErrorEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data is ErrorErrorEnum) {
      return data;
    }
    if (data != null) {
      switch (data) {
        case r'invalid_request':
          return ErrorErrorEnum.invalidRequest;
        case r'invalid_email':
          return ErrorErrorEnum.invalidEmail;
        case r'invalid_password':
          return ErrorErrorEnum.invalidPassword;
        case r'email_taken':
          return ErrorErrorEnum.emailTaken;
        case r'unauthenticated':
          return ErrorErrorEnum.unauthenticated;
        case r'wrong_credentials':
          return ErrorErrorEnum.wrongCredentials;
        case r'rate_limited':
          return ErrorErrorEnum.rateLimited;
        case r'invalid_challenge':
          return ErrorErrorEnum.invalidChallenge;
        case r'wrong_code':
          return ErrorErrorEnum.wrongCode;
        case r'already_enrolled':
          return ErrorErrorEnum.alreadyEnrolled;
        case r'not_enrolled':
          return ErrorErrorEnum.notEnrolled;
        case r'csrf':
          return ErrorErrorEnum.csrf;
        case r'forbidden':
          return ErrorErrorEnum.forbidden;
        case r'invalid_role':
          return ErrorErrorEnum.invalidRole;
        case r'invalid_name':
          return ErrorErrorEnum.invalidName;
        case r'invalid_invite':
          return ErrorErrorEnum.invalidInvite;
        case r'invalid_birth_date':
          return ErrorErrorEnum.invalidBirthDate;
        case r'invalid_phone':
          return ErrorErrorEnum.invalidPhone;
        case r'not_found':
          return ErrorErrorEnum.notFound;
        case r'version_conflict':
          return ErrorErrorEnum.versionConflict;
        case r'consent_active':
          return ErrorErrorEnum.consentActive;
        case r'consent_inactive':
          return ErrorErrorEnum.consentInactive;
        case r'profile_required':
          return ErrorErrorEnum.profileRequired;
        case r'invalid_specialty':
          return ErrorErrorEnum.invalidSpecialty;
        case r'invalid_slot_minutes':
          return ErrorErrorEnum.invalidSlotMinutes;
        case r'invalid_availability':
          return ErrorErrorEnum.invalidAvailability;
        case r'consent_required':
          return ErrorErrorEnum.consentRequired;
        case r'slot_unavailable':
          return ErrorErrorEnum.slotUnavailable;
        case r'cancellation_window_closed':
          return ErrorErrorEnum.cancellationWindowClosed;
        case r'appointment_closed':
          return ErrorErrorEnum.appointmentClosed;
        case r'appointment_not_started':
          return ErrorErrorEnum.appointmentNotStarted;
        case r'invalid_status':
          return ErrorErrorEnum.invalidStatus;
        case r'join_window_not_open':
          return ErrorErrorEnum.joinWindowNotOpen;
        case r'consultation_ended':
          return ErrorErrorEnum.consultationEnded;
        case r'practitioner_did_not_join':
          return ErrorErrorEnum.practitionerDidNotJoin;
        case r'consultation_not_started':
          return ErrorErrorEnum.consultationNotStarted;
        case r'not_draft':
          return ErrorErrorEnum.notDraft;
        case r'not_active':
          return ErrorErrorEnum.notActive;
        case r'draft_exists':
          return ErrorErrorEnum.draftExists;
        case r'invalid_questionnaire':
          return ErrorErrorEnum.invalidQuestionnaire;
        case r'already_requested':
          return ErrorErrorEnum.alreadyRequested;
        case r'appointment_started':
          return ErrorErrorEnum.appointmentStarted;
        case r'submitted':
          return ErrorErrorEnum.submitted;
        case r'withdrawn':
          return ErrorErrorEnum.withdrawn;
        case r'past_due':
          return ErrorErrorEnum.pastDue;
        case r'answers_invalid':
          return ErrorErrorEnum.answersInvalid;
        case r'not_attachment_item':
          return ErrorErrorEnum.notAttachmentItem;
        case r'length_required':
          return ErrorErrorEnum.lengthRequired;
        case r'too_large':
          return ErrorErrorEnum.tooLarge;
        case r'response_full':
          return ErrorErrorEnum.responseFull;
        case r'too_many_files':
          return ErrorErrorEnum.tooManyFiles;
        case r'unsupported_type':
          return ErrorErrorEnum.unsupportedType;
        case r'scanning':
          return ErrorErrorEnum.scanning;
        case r'range_not_satisfiable':
          return ErrorErrorEnum.rangeNotSatisfiable;
        case r'attachments_pending':
          return ErrorErrorEnum.attachmentsPending;
        case r'language_unavailable':
          return ErrorErrorEnum.languageUnavailable;
        case r'language_locked':
          return ErrorErrorEnum.languageLocked;
        case r'translation_incomplete':
          return ErrorErrorEnum.translationIncomplete;
        case r'internal error':
          return ErrorErrorEnum.internalError;
        case r'timeout':
          return ErrorErrorEnum.timeout;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// The singleton instance of this transformer.
  static ErrorErrorEnumTypeTransformer? _instance;
}
