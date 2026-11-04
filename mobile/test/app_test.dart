import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

import 'support.dart';

void main() {
  testWidgets('without a stored token it shows sign-in and calls nothing', (tester) async {
    final h = Harness();
    await h.pump(tester);

    expect(find.text('Sign in to Salus'), findsOneWidget);
    expect(h.api.requests, isEmpty);
  });

  testWidgets('with a stored token it reads the account with the bearer token', (tester) async {
    final h = Harness(store: FakeTokenStore(token));
    h.api.answer('GET /auth/me', 200, account());
    await h.pump(tester);

    expect(find.text('Signed in as $email'), findsOneWidget);
    expect(h.api.last('GET /auth/me').headers['Authorization'], 'Bearer $token');
    expectNoCsrfHeader(h.api);
  });

  testWidgets('a stored token the API refuses is deleted', (tester) async {
    final h = Harness(store: FakeTokenStore(token));
    h.api.answer('GET /auth/me', 401, error('unauthenticated'));
    await h.pump(tester);

    expect(find.text('Sign in to Salus'), findsOneWidget);
    expect(h.store.token, isNull);
  });

  testWidgets('an unreachable API keeps the token and offers to try again', (tester) async {
    final h = Harness(store: FakeTokenStore(token));
    h.api.on('GET /auth/me', (_) async => throw http.ClientException('network'));
    await h.pump(tester);

    expect(find.text('Salus could not be reached. Check your connection and try again.'), findsOneWidget);
    expect(h.store.token, token);

    h.api.answer('GET /auth/me', 200, account());
    await tapButton(tester, 'Try again');

    expect(find.text('Signed in as $email'), findsOneWidget);
  });

  testWidgets('a server error on start is a fixed message, not the response', (tester) async {
    final h = Harness(store: FakeTokenStore(token));
    h.api.answer('GET /auth/me', 500, error('internal error'));
    await h.pump(tester);

    expect(find.text('Something went wrong. Try again.'), findsOneWidget);
    expect(find.textContaining('internal error'), findsNothing);
    expect(h.store.token, token);
  });
}
