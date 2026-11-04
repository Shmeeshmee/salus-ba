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

String failureMessage(AuthFailure failure) {
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
