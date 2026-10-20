import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:salus/health_screen.dart';
import 'package:salus_api/api.dart';

// healthApiAnswering returns the generated client with its HTTP client replaced by [respond].
HealthApi healthApiAnswering(Future<http.Response> Function(http.Request) respond) {
  final client = ApiClient(basePath: 'http://api.test')..client = MockClient(respond);
  return HealthApi(client);
}

Future<void> pumpScreen(WidgetTester tester, HealthApi api) async {
  await tester.pumpWidget(MaterialApp(home: HealthScreen(api: api)));
}

void main() {
  testWidgets('calls /healthz and shows ok', (tester) async {
    String? path;
    final api = healthApiAnswering((request) async {
      path = request.url.path;
      return http.Response('{"status":"ok"}', 200, headers: {'content-type': 'application/json'});
    });

    await pumpScreen(tester, api);
    expect(find.text('API status: checking'), findsOneWidget);
    await tester.pumpAndSettle();

    expect(find.text('API status: ok'), findsOneWidget);
    expect(path, '/healthz');
  });

  testWidgets('shows unavailable when the API answers an error', (tester) async {
    await pumpScreen(tester, healthApiAnswering((_) async => http.Response('', 503)));
    await tester.pumpAndSettle();

    expect(find.text('API status: unavailable'), findsOneWidget);
  });

  testWidgets('shows unavailable when the API cannot be reached', (tester) async {
    await pumpScreen(tester, healthApiAnswering((_) async => throw http.ClientException('network')));
    await tester.pumpAndSettle();

    expect(find.text('API status: unavailable'), findsOneWidget);
  });
}
