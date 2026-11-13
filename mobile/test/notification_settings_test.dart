import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:salus/l10n/app_localizations.dart';
import 'package:salus/l10n/language.dart';
import 'package:salus/notification_settings_screen.dart';
import 'package:salus/zones.dart';

import 'support.dart';

const route = '/me/notification-settings';

/// The API's defaults: push and e-mail on, SMS off, quiet hours 22:00 to 08:00, no zone.
Map<String, Object?> defaults() => {
  'channels': {'push': true, 'email': true, 'sms': false},
  'quiet_hours': {'start': '22:00', 'end': '08:00'},
};

/// The screen on its own, with the device's zone given, in [language].
Future<Harness> atSettings(
  WidgetTester tester, {
  Map<String, Object?>? stored,
  String? deviceZone,
  String language = 'en',
}) async {
  final h = Harness(store: FakeTokenStore(token));
  h.api.answer('GET $route', 200, stored ?? defaults());
  h.api.on('PUT $route', (r) async => jsonResponse(200, jsonDecode(r.body)));
  await tester.pumpWidget(
    MaterialApp(
      locale: localeOf(language),
      supportedLocales: AppLocalizations.supportedLocales,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      home: NotificationSettingsScreen(booking: h.booking, deviceZone: () => deviceZone),
    ),
  );
  await tester.pumpAndSettle();
  return h;
}

bool switchOf(WidgetTester tester, String title) =>
    tester.widget<SwitchListTile>(find.widgetWithText(SwitchListTile, title)).value;

Future<void> toggle(WidgetTester tester, String title) => tapFound(tester, find.widgetWithText(SwitchListTile, title));

Map<String, Object?> lastPut(Harness h) => sent(h.api.last('PUT $route').body);

void main() {
  testWidgets('is reached from the home screen', (tester) async {
    final h = Harness(store: FakeTokenStore(token));
    h.api.answer('GET /auth/me', 200, account());
    h.api.answer('GET $route', 200, defaults());
    await h.pump(tester);

    await tapOutlined(tester, 'Reminders');

    expect(find.text('Choose how your clinics remind you of your appointments.'), findsOneWidget);
    expect(h.api.calls, contains('GET $route'));
  });

  testWidgets('shows the channels and the quiet hours as stored', (tester) async {
    await atSettings(tester);

    expect(switchOf(tester, 'Push notifications'), isTrue);
    expect(switchOf(tester, 'E-mail'), isTrue);
    expect(switchOf(tester, 'SMS'), isFalse);
    expect(
      find.text(
        'Reminders and waitlist offers by SMS to the phone number in your profile, only at clinics that offer SMS. '
        'Never used for advertising.',
      ),
      findsOneWidget,
    );
    expect(switchOf(tester, 'Quiet hours'), isTrue);
    expect(find.widgetWithText(ListTile, '22:00'), findsOneWidget);
    expect(find.widgetWithText(ListTile, '08:00'), findsOneWidget);
  });

  testWidgets('saves the channels and the quiet hours with the device zone', (tester) async {
    final h = await atSettings(tester, deviceZone: 'Asia/Dubai');

    await toggle(tester, 'SMS');
    await toggle(tester, 'Push notifications');
    await tapButton(tester, 'Save');

    expect(lastPut(h), {
      'channels': {'push': false, 'email': true, 'sms': true},
      'quiet_hours': {'start': '22:00', 'end': '08:00'},
      'time_zone': 'Asia/Dubai',
    });
    expect(find.text('Your choices are saved.'), findsOneWidget);
    expectNoCsrfHeader(h.api);
  });

  testWidgets('keeps the stored zone when the device gives none', (tester) async {
    final h = await atSettings(tester, stored: {...defaults(), 'time_zone': 'Europe/Sarajevo'});

    await tapButton(tester, 'Save');

    expect(lastPut(h)['time_zone'], 'Europe/Sarajevo');
  });

  testWidgets('turns the quiet hours off', (tester) async {
    final h = await atSettings(
      tester,
      stored: {
        ...defaults(),
        'quiet_hours': {'start': '23:30', 'end': '06:15'},
      },
    );

    expect(find.widgetWithText(ListTile, '23:30'), findsOneWidget);
    expect(find.widgetWithText(ListTile, '06:15'), findsOneWidget);
    await toggle(tester, 'Quiet hours');
    expect(find.text('From'), findsNothing);
    await tapButton(tester, 'Save');

    expect(lastPut(h).containsKey('quiet_hours'), isFalse);
  });

  testWidgets('turns the quiet hours on at 22:00 to 08:00', (tester) async {
    final h = await atSettings(
      tester,
      stored: {
        'channels': {'push': true, 'email': true, 'sms': false},
      },
    );

    expect(switchOf(tester, 'Quiet hours'), isFalse);
    await toggle(tester, 'Quiet hours');
    await tapButton(tester, 'Save');

    expect(lastPut(h)['quiet_hours'], {'start': '22:00', 'end': '08:00'});
  });

  testWidgets('picks a start time in 24 hours', (tester) async {
    final h = await atSettings(tester);

    await tapFound(tester, find.widgetWithText(ListTile, 'From'));
    final fields = find.descendant(of: find.byType(Dialog), matching: find.byType(TextField));
    await tester.enterText(fields.at(0), '23');
    await tester.enterText(fields.at(1), '45');
    await tapLink(tester, 'OK');
    await tapButton(tester, 'Save');

    expect(lastPut(h)['quiet_hours'], {'start': '23:45', 'end': '08:00'});
  });

  testWidgets('says SMS needs a phone number in the profile', (tester) async {
    final h = await atSettings(tester);
    h.api.answer('PUT $route', 409, error('phone_required'));

    await toggle(tester, 'SMS');
    await tapButton(tester, 'Save');

    expect(
      find.text('SMS needs a phone number in your profile. Add one in My profile, then try again.'),
      findsOneWidget,
    );
  });

  testWidgets('says when the settings are refused', (tester) async {
    final h = await atSettings(tester);
    h.api.answer('PUT $route', 400, error('invalid_request'));

    await tapButton(tester, 'Save');

    expect(find.text('These settings could not be saved. Check them and try again.'), findsOneWidget);
  });

  testWidgets('offers to try again when the settings cannot be loaded', (tester) async {
    final h = Harness(store: FakeTokenStore(token));
    h.api.answer('GET $route', 500, error('internal error'));
    await tester.pumpWidget(
      MaterialApp(
        locale: localeOf('en'),
        supportedLocales: AppLocalizations.supportedLocales,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        home: NotificationSettingsScreen(booking: h.booking, deviceZone: () => null),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.widgetWithText(FilledButton, 'Try again'), findsOneWidget);
  });

  testWidgets('is in the language of the app, the SMS opt-in included', (tester) async {
    await atSettings(tester, language: 'bs');

    expect(find.text('Podsjetnici'), findsOneWidget);
    expect(find.textContaining('Nikada za oglašavanje.'), findsOneWidget);
  });

  test('takes only an IANA zone from the device', () {
    expect(Zone.isIana('Asia/Dubai'), isTrue);
    expect(Zone.isIana('Europe/Sarajevo'), isTrue);
    expect(Zone.isIana('CET'), isFalse);
    expect(Zone.isIana('Mars/Olympus'), isFalse);
  });
}
