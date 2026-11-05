import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'support.dart';

const revokeWarning =
    '$clinicName will no longer see your profile. It keeps your name and the details of appointments that have '
    'already taken place, as part of its care records. Your future appointments with $clinicName will be '
    'cancelled.';

Future<Harness> atConsents(WidgetTester tester) async {
  final h = Harness(store: FakeTokenStore(token));
  h.api.answer('GET /auth/me', 200, account());
  answerDirectory(h.api);
  h.api.answer('GET /me/consents', 200, {
    'consents': [consent('active-one'), consent('old-one', revokedAt: '2026-10-20T12:00:00Z')],
  });
  await h.pump(tester);
  await tapOutlined(tester, 'My consents');
  return h;
}

void main() {
  testWidgets('lists the consents with the clinic, when given and whether revoked', (tester) async {
    await atConsents(tester);

    expect(find.text(clinicName), findsNWidgets(2));
    expect(find.text('Given on Sunday 1 November 2026'), findsNWidgets(2));
    expect(find.text('Active'), findsOneWidget);
    expect(find.text('Revoked on Tuesday 20 October 2026'), findsOneWidget);
    expect(find.widgetWithText(OutlinedButton, 'Revoke consent'), findsOneWidget);
  });

  testWidgets('revoking says that future appointments there are cancelled, and can be called off', (tester) async {
    final h = await atConsents(tester);
    await tapOutlined(tester, 'Revoke consent');

    expect(find.text('Revoke your consent?'), findsOneWidget);
    expect(find.text(revokeWarning), findsOneWidget);

    await tapLink(tester, 'Keep consent');
    expect(h.api.calls.where((c) => c.startsWith('POST')), isEmpty);
    expect(find.text('Active'), findsOneWidget);
  });

  testWidgets('revokes after the confirmation and reads the list again', (tester) async {
    final h = await atConsents(tester);
    h.api.answer('POST /consents/active-one/revoke', 200, consent('active-one', revokedAt: '2026-11-04T09:00:00Z'));
    await tapOutlined(tester, 'Revoke consent');

    h.api.answer('GET /me/consents', 200, {
      'consents': [
        consent('active-one', revokedAt: '2026-11-04T09:00:00Z'),
        consent('old-one', revokedAt: '2026-10-20T12:00:00Z'),
      ],
    });
    await tapButton(tester, 'Revoke consent');

    expect(h.api.calls.where((c) => c.startsWith('POST')), ['POST /consents/active-one/revoke']);
    expect(
      find.text(
        'Your consent is revoked. Your future appointments with this clinic will be cancelled, and you will '
        'get an e-mail for each.',
      ),
      findsOneWidget,
    );
    expect(find.text('Active'), findsNothing);
    expect(find.widgetWithText(OutlinedButton, 'Revoke consent'), findsNothing);
    expectNoCsrfHeader(h.api);
  });

  testWidgets('a consent revoked elsewhere in the meantime says so', (tester) async {
    final h = await atConsents(tester);
    h.api.answer('POST /consents/active-one/revoke', 409, error('consent_inactive'));
    await tapOutlined(tester, 'Revoke consent');
    await tapButton(tester, 'Revoke consent');

    expect(find.text('This consent is already revoked.'), findsOneWidget);
  });

  testWidgets('no consents yet', (tester) async {
    final h = Harness(store: FakeTokenStore(token));
    h.api.answer('GET /auth/me', 200, account());
    answerDirectory(h.api);
    h.api.answer('GET /me/consents', 200, {'consents': []});
    await h.pump(tester);
    await tapOutlined(tester, 'My consents');

    expect(find.text('You have not given any consents yet.'), findsOneWidget);
  });
}
