import 'package:flutter/widgets.dart';
import 'package:intl/intl.dart' show Bidi;

import 'fhir.dart';
import 'l10n/app_localizations.dart';
import 'session.dart';

// The app's text is in the catalogs (lib/l10n/app_<language>.arb, docs/i18n/SPEC.md §3 "Catalogs"); here API error
// codes map to their fixed sentences, and nothing from a response body is shown.

extension Words on BuildContext {
  /// The app's text in the language it shows.
  AppLocalizations get t => AppLocalizations.of(this);
}

String tooManyAttempts(AppLocalizations t, Duration? retryAfter) {
  if (retryAfter == null) return t.tooManyAttemptsLater;
  return t.tooManyAttempts((retryAfter.inSeconds / 60).ceil());
}

String failureMessage(AppLocalizations t, ApiFailure failure) {
  if (failure.unreachable) return t.unreachable;
  return switch (failure.code) {
    'wrong_credentials' => t.wrongCredentials,
    'rate_limited' => tooManyAttempts(t, failure.retryAfter),
    'wrong_code' => t.wrongCode,
    'invalid_challenge' => t.challengeEnded,
    'email_taken' => t.emailTaken,
    'invalid_email' => t.invalidEmail,
    'invalid_password' => t.invalidPassword,
    _ => t.somethingWrong,
  };
}

final _email = RegExp(r'^[^@\s]+@[^@\s]+$');
final _code = RegExp(r'^\d{6}$');

String? validateEmail(AppLocalizations t, String? value) {
  final email = (value ?? '').trim();
  return email.length <= 254 && _email.hasMatch(email) ? null : t.invalidEmail;
}

/// The API's rule: 12 to 128 characters (code points), no composition rules.
String? validateNewPassword(AppLocalizations t, String? value) {
  final length = (value ?? '').runes.length;
  return length >= 12 && length <= 128 ? null : t.invalidPassword;
}

String? validatePassword(AppLocalizations t, String? value) => (value ?? '').isEmpty ? t.missingPassword : null;

String? validateCode(AppLocalizations t, String? value) => _code.hasMatch((value ?? '').trim()) ? null : t.invalidCode;

/// The fixed sentence for a failed booking, appointment, consent or profile call.
String bookingMessage(AppLocalizations t, ApiFailure failure) {
  if (failure.unreachable) return t.unreachable;
  return switch (failure.code) {
    'slot_unavailable' => t.slotTaken,
    'consent_required' => t.consentNeeded,
    'cancellation_window_closed' => t.cancelWindowClosed,
    'appointment_closed' => t.appointmentClosed,
    'consent_inactive' => t.consentAlreadyRevoked,
    'invalid_name' => t.invalidName,
    'invalid_birth_date' => t.birthDateRange,
    'invalid_phone' => t.invalidPhone(phoneExample(t)),
    'version_conflict' => t.profileChanged,
    _ => t.somethingWrong,
  };
}

/// The fixed sentence for a refused write of an allergy or a medication (SAL-41), by its status: the API's codes
/// for these say no more.
String entryMessage(AppLocalizations t, ApiFailure failure) {
  if (failure.unreachable) return t.unreachable;
  return switch (failure.status) {
    400 => t.entryInvalid,
    403 => t.entryNotAllowed,
    404 => t.entryNotFound,
    412 => t.entryChanged,
    _ => t.somethingWrong,
  };
}

// Profile (docs/booking/SPEC.md "Profile"): the API's rules, checked here first.
String? validateGivenName(AppLocalizations t, String? value) => _nameOk(value) ? null : t.invalidGivenName;

String? validateFamilyName(AppLocalizations t, String? value) => _nameOk(value) ? null : t.invalidFamilyName;

bool _nameOk(String? value) {
  final length = (value ?? '').trim().runes.length;
  return length >= 1 && length <= 100;
}

final _date = RegExp(r'^(\d{4})-(\d{2})-(\d{2})$');

/// The date of `YYYY-MM-DD` as UTC midnight, or null when it is not a real day.
DateTime? parseDate(String? value) {
  final m = _date.firstMatch((value ?? '').trim());
  if (m == null) return null;
  final (y, mo, d) = (int.parse(m[1]!), int.parse(m[2]!), int.parse(m[3]!));
  final date = DateTime.utc(y, mo, d);
  return date.year == y && date.month == mo && date.day == d ? date : null;
}

/// A birth date before [now]'s date in UTC and not before 1900-01-01, as the API checks it.
String? validateBirthDate(AppLocalizations t, String? value, DateTime now) {
  final date = parseDate(value);
  if (date == null) return t.birthDateFormat;
  final today = DateTime.utc(now.toUtc().year, now.toUtc().month, now.toUtc().day);
  return date.isBefore(DateTime.utc(1900)) || !date.isBefore(today) ? t.birthDateRange : null;
}

/// The example phone number of the profile's hint and error (synthetic, from the fake +387 60 000 0xxx range). Arabic
/// is written right to left, which would put its groups of digits in reverse order: there it is isolated as a left
/// to right run (docs/i18n/SPEC.md §3 "Right to left").
String phoneExample(AppLocalizations t) => Bidi.isRtlLanguage(t.localeName) ? '\u2066$_example\u2069' : _example;

const _example = '+387 60 000 0000';

final _phone = RegExp(r'^\+\d{8,15}$');

/// The phone in E.164 with spaces, dashes, dots and brackets taken out; '' for none.
String normalisePhone(String? value) => (value ?? '').replaceAll(RegExp(r'[\s\-.()]'), '');

String? validatePhone(AppLocalizations t, String? value) {
  final phone = normalisePhone(value);
  return phone.isEmpty || _phone.hasMatch(phone) ? null : t.invalidPhone(phoneExample(t));
}

/// The approved sentence for a refused join (consultation COPY M29, M30, M32, M33); `practitioner_did_not_join` is
/// its own screen (M31: M25 and M26).
String joinMessage(AppLocalizations t, ApiFailure failure) {
  if (failure.unreachable) return t.unreachable;
  return switch (failure.code) {
    'join_window_not_open' => t.windowNotOpen,
    'consultation_ended' => t.consultationEnded,
    'consent_required' => t.joinNeedsConsent,
    'not_found' => t.consultationNotAvailable,
    _ => t.somethingWrong,
  };
}

/// The approved sentence for a refused save or submit (questionnaire COPY M16); a 422 is shown on its questions
/// instead.
String answersMessage(AppLocalizations t, ApiFailure failure) {
  if (failure.unreachable) return t.unreachable;
  return switch (failure.code) {
    'consent_required' => t.fillingNeedsConsent,
    'past_due' => t.pastDue,
    'withdrawn' => t.withdrawnByClinic,
    'submitted' => t.alreadySubmitted,
    'version_conflict' => t.changedElsewhere,
    'not_found' => t.questionnaireNotAvailable,
    // The appointment was cancelled (added on review of PR #24, owner).
    'appointment_closed' => t.questionnaireNotAvailable,
    'attachments_pending' => t.filesBeingChecked, // docs/attachments/COPY.md M10
    'language_unavailable' => t.languageUnavailable, // docs/i18n/COPY.md M4
    _ => t.somethingWrong,
  };
}

/// A role's specialty in the app's language, from the catalog by its SNOMED CT code (docs/i18n/SPEC.md §3 "Server
/// text"); the API's own text only for a code the catalog does not have yet.
String? specialtyName(AppLocalizations t, PractitionerRole role) => specialtyOf(t, role.specialtyCode) ?? role.specialty;

/// The specialty of the SNOMED CT [code] in the app's language, or null for a code the catalog does not have.
String? specialtyOf(AppLocalizations t, String? code) => switch (code) {
  '394814009' => t.specialty394814009,
  '419192003' => t.specialty419192003,
  '394579002' => t.specialty394579002,
  '394582007' => t.specialty394582007,
  '394586005' => t.specialty394586005,
  '394537008' => t.specialty394537008,
  '394587001' => t.specialty394587001,
  '394591006' => t.specialty394591006,
  '418960008' => t.specialty418960008,
  '394594003' => t.specialty394594003,
  '394801008' => t.specialty394801008,
  '394583002' => t.specialty394583002,
  _ => null,
};

/// A practitioner's specialty and clinic as one line (docs/i18n/SPEC.md §3: no string built by joining pieces); the
/// clinic alone when the role has no specialty.
String specialtyAtClinic(AppLocalizations t, String? specialty, String clinic) =>
    specialty == null ? clinic : t.specialtyAtClinic(specialty, clinic);
