import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

import 'support.dart';

Future<Harness> signedIn(WidgetTester tester) async {
  final h = Harness(store: FakeTokenStore(token));
  h.api.answer('GET /auth/me', 200, account());
  await h.pump(tester);
  return h;
}

void main() {
  testWidgets('sign-out ends the session and clears the stored token', (tester) async {
    final h = await signedIn(tester);
    h.api.answer('POST /auth/logout', 204);
    await tapButton(tester, 'Sign out');

    expect(find.text('Sign in to Salus'), findsOneWidget);
    expect(h.store.token, isNull);
    expect(h.api.last('POST /auth/logout').headers['Authorization'], 'Bearer $token');
    expectNoCsrfHeader(h.api);
  });

  testWidgets('a session the API already ended also counts as signed out', (tester) async {
    final h = await signedIn(tester);
    h.api.answer('POST /auth/logout', 401, error('unauthenticated'));
    await tapButton(tester, 'Sign out');

    expect(find.text('Sign in to Salus'), findsOneWidget);
    expect(h.store.token, isNull);
  });

  testWidgets('after sign-out the next request carries no token', (tester) async {
    final h = await signedIn(tester);
    h.api.answer('POST /auth/logout', 204);
    await tapButton(tester, 'Sign out');

    h.api.answer('POST /auth/login', 401, error('wrong_credentials'));
    await enter(tester, 'E-mail', email);
    await enter(tester, 'Password', password);
    await tapButton(tester, 'Sign in');

    expect(h.api.last('POST /auth/login').headers['Authorization'], isNull);
  });

  testWidgets('a sign-out that does not reach the API keeps the session', (tester) async {
    final h = await signedIn(tester);
    h.api.on('POST /auth/logout', (_) async => throw http.ClientException('network'));
    await tapButton(tester, 'Sign out');

    expect(find.text('Could not sign out. Check your connection and try again.'), findsOneWidget);
    expect(find.text('Signed in as $email'), findsOneWidget);
    expect(h.store.token, token);
  });
}
