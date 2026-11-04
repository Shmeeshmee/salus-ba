import 'session.dart';

// Fixed sentences. API error codes map to these; nothing from a response body is shown.
const somethingWrong = 'Something went wrong. Try again.';
const unreachable = 'Salus could not be reached. Check your connection and try again.';
const wrongCredentials = 'The e-mail or password is wrong.';
const wrongCode = 'The code is wrong. Check the time on your device and try again.';
const challengeEnded = 'The sign-in has ended. Enter your e-mail and password again.';
const emailTaken = 'This e-mail already has an account. Sign in instead.';
const invalidEmail = 'Enter a valid e-mail address.';
const invalidPassword = 'Use 12 to 128 characters for the password.';
const missingPassword = 'Enter your password.';
const invalidCode = 'The code has 6 digits.';
const staffOnly = 'This app is for patients. Clinic staff use the Salus web app.';
const accountReady = 'Your account is ready. Sign in to continue.';
const signOutFailed = 'Could not sign out. Check your connection and try again.';

String tooManyAttempts(Duration? retryAfter) {
  if (retryAfter == null) return 'Too many attempts. Try again later.';
  final minutes = (retryAfter.inSeconds / 60).ceil();
  return 'Too many attempts. Try again in $minutes ${minutes == 1 ? 'minute' : 'minutes'}.';
}

String failureMessage(ApiFailure failure) {
  if (failure.unreachable) return unreachable;
  return switch (failure.code) {
    'wrong_credentials' => wrongCredentials,
    'rate_limited' => tooManyAttempts(failure.retryAfter),
    'wrong_code' => wrongCode,
    'invalid_challenge' => challengeEnded,
    'email_taken' => emailTaken,
    'invalid_email' => invalidEmail,
    'invalid_password' => invalidPassword,
    _ => somethingWrong,
  };
}

final _email = RegExp(r'^[^@\s]+@[^@\s]+$');
final _code = RegExp(r'^\d{6}$');

String? validateEmail(String? value) {
  final email = (value ?? '').trim();
  return email.length <= 254 && _email.hasMatch(email) ? null : invalidEmail;
}

/// The API's rule: 12 to 128 characters (code points), no composition rules.
String? validateNewPassword(String? value) {
  final length = (value ?? '').runes.length;
  return length >= 12 && length <= 128 ? null : invalidPassword;
}

String? validatePassword(String? value) => (value ?? '').isEmpty ? missingPassword : null;

String? validateCode(String? value) => _code.hasMatch((value ?? '').trim()) ? null : invalidCode;

// Profile (docs/booking/SPEC.md "Profile"): the API's rules, checked here first.
const invalidGivenName = 'Enter your given name, up to 100 characters.';
const invalidFamilyName = 'Enter your family name, up to 100 characters.';
const invalidName = 'Enter your given and family name, up to 100 characters each.';
const birthDateFormat = 'Enter your date of birth as YYYY-MM-DD.';
const birthDateRange = 'Enter a date of birth in the past, from 1900 on.';
const invalidPhone = 'Enter the phone number with its country code, for example +387 60 000 0000.';
const profileChanged = 'Your profile was changed in the meantime. Check it and save again.';
const completeProfileFirst = 'Complete your profile before you book.';

// Booking, appointments and consents.
const slotTaken = 'This time is no longer free. Choose another time.';
const consentNeeded = 'Booking with this clinic needs your consent.';
const appointmentBooked = 'Your appointment is booked.';
const appointmentCancelled = 'Your appointment is cancelled.';
const cancelWindowHint = 'Can be cancelled in the app until 24 hours before the start.';
const cancelWindowClosed = 'This appointment starts in less than 24 hours and can no longer be cancelled in the app.';
const appointmentClosed = 'This appointment is no longer booked.';
const consentRevoked =
    'Your consent is revoked. Your future appointments with this clinic will be cancelled, and you will get an '
    'e-mail for each.';
const consentAlreadyRevoked = 'This consent is already revoked.';
const unlistedClinic = 'A clinic no longer listed';

/// The fixed sentence for a failed booking, appointment, consent or profile call.
String bookingMessage(ApiFailure failure) {
  if (failure.unreachable) return unreachable;
  return switch (failure.code) {
    'slot_unavailable' => slotTaken,
    'consent_required' => consentNeeded,
    'cancellation_window_closed' => cancelWindowClosed,
    'appointment_closed' => appointmentClosed,
    'consent_inactive' => consentAlreadyRevoked,
    'invalid_name' => invalidName,
    'invalid_birth_date' => birthDateRange,
    'invalid_phone' => invalidPhone,
    'version_conflict' => profileChanged,
    _ => somethingWrong,
  };
}

String? validateGivenName(String? value) => _nameOk(value) ? null : invalidGivenName;

String? validateFamilyName(String? value) => _nameOk(value) ? null : invalidFamilyName;

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
String? validateBirthDate(String? value, DateTime now) {
  final date = parseDate(value);
  if (date == null) return birthDateFormat;
  final today = DateTime.utc(now.toUtc().year, now.toUtc().month, now.toUtc().day);
  return date.isBefore(DateTime.utc(1900)) || !date.isBefore(today) ? birthDateRange : null;
}

final _phone = RegExp(r'^\+\d{8,15}$');

/// The phone in E.164 with spaces, dashes, dots and brackets taken out; '' for none.
String normalisePhone(String? value) => (value ?? '').replaceAll(RegExp(r'[\s\-.()]'), '');

String? validatePhone(String? value) {
  final phone = normalisePhone(value);
  return phone.isEmpty || _phone.hasMatch(phone) ? null : invalidPhone;
}
