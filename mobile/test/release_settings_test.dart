import 'package:flutter_test/flutter_test.dart';
import 'package:salus/release_settings.dart';

// A release build refuses to start with a build-time setting still on its placeholder (main.dart): the support address
// in brackets, the privacy notice on salus.invalid, the API on localhost (docs/deploy/BEFORE-PRODUCTION.md).
void main() {
  const good = (
    api: 'https://api.clinic.example',
    privacy: 'https://clinic.example/privacy',
    support: 'support@clinic.example',
  );

  List<String> problems({String? api, String? privacy, String? support}) => releaseSettingProblems(
        apiBaseUrl: api ?? good.api,
        privacyNoticeUrl: privacy ?? good.privacy,
        supportEmail: support ?? good.support,
      );

  test('real values pass', () {
    expect(problems(), isEmpty);
  });

  test('the defaults of a build without --dart-define are each refused, by name', () {
    expect(
      releaseSettingProblems(
        apiBaseUrl: 'http://localhost:8080',
        privacyNoticeUrl: 'https://salus.invalid/privacy',
        supportEmail: '[SUPPORT EMAIL]',
      ),
      ['API_BASE_URL', 'PRIVACY_NOTICE_URL', 'SUPPORT_EMAIL'],
    );
  });

  test('a support address that is empty, bracketed or not an address is refused', () {
    for (final support in ['', '  ', '[SUPPORT EMAIL]', '[support]', 'support', 'help@salus.invalid']) {
      expect(problems(support: support), ['SUPPORT_EMAIL'], reason: support);
    }
  });

  test('a privacy notice URL that is not https or on a placeholder host is refused', () {
    for (final url in ['', 'https://salus.invalid/privacy', 'http://clinic.example/privacy', '[PRIVACY URL]', 'https://localhost/privacy', 'privacy']) {
      expect(problems(privacy: url), ['PRIVACY_NOTICE_URL'], reason: url);
    }
  });

  test('an API base that is not https or on localhost or the emulator host is refused', () {
    for (final url in ['', 'http://localhost:8080', 'https://localhost', 'https://127.0.0.1:8080', 'http://10.0.2.2:8080', 'http://api.clinic.example', 'https://api.salus.invalid']) {
      expect(problems(api: url), ['API_BASE_URL'], reason: url);
    }
  });
}
