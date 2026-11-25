import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

import 'support.dart';

const report = 'd5000000-0000-4000-8000-000000000001';
const second = 'd5000000-0000-4000-8000-000000000002';

// A synthetic released lab result: its names, values and ranges are placeholders, not clinical guidance.
Map<String, Object?> labResult(String id, {bool hasPdf = true, List<Map<String, Object?>>? results}) => {
  'id': id,
  'organizationId': clinicId,
  'patientId': 'b1000000-0000-4000-8000-000000000001',
  'appointmentId': 'e1000000-0000-4000-8000-000000000001',
  'status': 'final',
  'title': 'Placeholder panel',
  'effectiveDate': '2026-11-02',
  'results':
      results ??
      [
        {'test': 'Placeholder test A', 'value': 5.4, 'unit': 'mmol/L', 'referenceRange': '3.9-5.6', 'interpretation': 'N'},
        {'test': 'Placeholder test B', 'value': 12, 'unit': 'g/dL', 'interpretation': 'L'},
        {'test': 'Placeholder test C', 'valueText': 'Placeholder text value'},
      ],
  'hasPdf': hasPdf,
  'createdAt': '2026-11-02T08:00:00Z',
  'releasedAt': '2026-11-03T10:00:00Z',
  'versionId': 3,
};

Future<Harness> atLabResults(WidgetTester tester, List<Map<String, Object?>> reports) async {
  final h = Harness(store: FakeTokenStore(token));
  h.api.answer('GET /auth/me', 200, account());
  answerDirectory(h.api);
  h.api.answer('GET /me/diagnostic-reports', 200, reports);
  await h.pump(tester);
  await tapOutlined(tester, 'My lab results');
  return h;
}

void main() {
  testWidgets('lists the released results with their dates, values, ranges and flags', (tester) async {
    await atLabResults(tester, [labResult(report)]);

    expect(find.text('Placeholder panel'), findsOneWidget);
    expect(find.text('Sample taken on Monday 2 November 2026'), findsOneWidget);
    expect(find.text('Released on Tuesday 3 November 2026'), findsOneWidget);
    expect(find.text('Placeholder test A: 5.4 mmol/L'), findsOneWidget);
    expect(find.text('Reference range 3.9-5.6'), findsOneWidget);
    expect(find.text('Normal'), findsOneWidget);
    expect(find.text('Placeholder test B: 12 g/dL'), findsOneWidget);
    expect(find.text('Low'), findsOneWidget);
    expect(find.text('Placeholder test C: Placeholder text value'), findsOneWidget);
  });

  testWidgets('says when there are no results', (tester) async {
    await atLabResults(tester, const []);

    expect(find.text('You have no lab results yet.'), findsOneWidget);
  });

  testWidgets("shares a result's PDF, asked with the session's token, and offers none without one", (tester) async {
    final h = await atLabResults(tester, [labResult(report), labResult(second, hasPdf: false, results: const [])]);
    h.api.on(
      'GET /me/diagnostic-reports/$report/pdf',
      (_) async => http.Response.bytes([0x25, 0x50, 0x44, 0x46], 200, headers: {'content-type': 'application/pdf'}),
    );

    expect(find.text('Save or share the PDF'), findsOneWidget);
    await tapButton(tester, 'Save or share the PDF');

    expect(h.files.shared.single.$1, 'Placeholder panel.pdf');
    expect(h.files.shared.single.$2, [0x25, 0x50, 0x44, 0x46]);
    expect(h.api.last('GET /me/diagnostic-reports/$report/pdf').headers['Authorization'], 'Bearer $token');
  });

  testWidgets('says so when the PDF cannot be read', (tester) async {
    final h = await atLabResults(tester, [labResult(report)]);
    h.api.answer('GET /me/diagnostic-reports/$report/pdf', 404, error('not_found'));

    await tapButton(tester, 'Save or share the PDF');

    expect(h.files.shared, isEmpty);
    expect(find.text('Something went wrong. Try again.'), findsOneWidget);
  });
}
