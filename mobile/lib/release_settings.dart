/// The build-time settings a release build must have been given (`--dart-define`, scripts/mobile-release.sh): their
/// defaults are placeholders that must never reach a patient (docs/deploy/BEFORE-PRODUCTION.md). [releaseSettingProblems]
/// names each one still on a placeholder; main() refuses to start a release build while any is.
library;

/// The hosts a released app may not talk to or link to: the developer's machine, the Android emulator's host, and the
/// placeholder domain.
bool _placeholderHost(String host) =>
    host == 'localhost' || host == '127.0.0.1' || host == '10.0.2.2' || host == 'invalid' || host.endsWith('.invalid');

bool _bracketed(String value) => value.contains('[') || value.contains(']');

bool _badUrl(String value) {
  final uri = Uri.tryParse(value.trim());
  return uri == null || uri.scheme != 'https' || uri.host.isEmpty || _placeholderHost(uri.host) || _bracketed(value);
}

bool _badEmail(String value) {
  final v = value.trim();
  final at = v.lastIndexOf('@');
  return at <= 0 || at == v.length - 1 || _bracketed(v) || _placeholderHost(v.substring(at + 1));
}

/// The names of the settings that are missing or still a placeholder: an API base or privacy notice URL that is not
/// https or is on localhost, the emulator's host or an .invalid domain; a support address that is not one, is in
/// brackets or is on an .invalid domain.
List<String> releaseSettingProblems({
  required String apiBaseUrl,
  required String privacyNoticeUrl,
  required String supportEmail,
}) => [
  if (_badUrl(apiBaseUrl)) 'API_BASE_URL',
  if (_badUrl(privacyNoticeUrl)) 'PRIVACY_NOTICE_URL',
  if (_badEmail(supportEmail)) 'SUPPORT_EMAIL',
];
