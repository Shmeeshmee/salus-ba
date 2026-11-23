import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

import 'support.dart';

const sent = 'f1000000-0000-4000-8000-000000000001';
const expired = 'f1000000-0000-4000-8000-000000000002';

// A synthetic treatment quote (clinic-cms X1): an examination with a discount, valid 30 days.
Map<String, Object?> quote(String id, String state, {String note = '', String clinic = clinicId}) => {
  'id': id,
  'organizationId': clinic,
  'patientId': 'd1000000-0000-4000-8000-0000000000d1',
  'issuedAt': '2026-11-02T09:30:00Z',
  'validUntil': '2026-12-02T09:30:00Z',
  'state': state,
  'note': note,
  'currency': 'BAM',
  'totalMinor': 9500,
  'lines': [
    {'name': 'Placeholder examination', 'quantity': 2, 'unitPriceMinor': 5000, 'discountMinor': 500, 'totalMinor': 9500},
  ],
};

Future<Harness> atQuotes(WidgetTester tester, List<Map<String, Object?>> quotes) async {
  final h = Harness(store: FakeTokenStore(token));
  h.api.answer('GET /auth/me', 200, account());
  answerDirectory(h.api);
  h.api.answer('GET /me/quotes', 200, quotes);
  await h.pump(tester);
  await tapOutlined(tester, 'My quotes');
  return h;
}

void main() {
  testWidgets('lists the quotes with their date, validity, lines, note and total, and says a quote is no receipt', (tester) async {
    await atQuotes(tester, [quote(sent, 'sent', note: 'Placeholder note'), quote(expired, 'expired')]);

    expect(find.textContaining('A quote is not a receipt: nothing has been charged.'), findsOneWidget);
    expect(find.text('Quote'), findsNWidgets(2));
    expect(find.text('Issued on Monday 2 November 2026'), findsNWidgets(2));
    expect(find.text('Valid until Wednesday 2 December 2026'), findsOneWidget);
    expect(find.text('Expired'), findsOneWidget);
    expect(find.text('Placeholder examination × 2'), findsNWidgets(2));
    expect(find.text('Placeholder note'), findsOneWidget);
    expect(find.text('95.00 BAM'), findsNWidgets(2));
  });

  testWidgets('names the clinic of each quote from the directory, or says it is no longer listed', (tester) async {
    await atQuotes(tester, [
      quote(sent, 'sent'),
      quote(expired, 'expired', clinic: 'a1000000-0000-4000-8000-0000000000a1'),
    ]);

    expect(find.text(clinicName), findsOneWidget);
    expect(find.text('A clinic no longer listed'), findsOneWidget);
  });

  testWidgets('says when there are no quotes', (tester) async {
    await atQuotes(tester, const []);

    expect(find.text('You have no quotes yet.'), findsOneWidget);
  });

  testWidgets("shares a quote's PDF, asked with the session's token", (tester) async {
    final h = await atQuotes(tester, [quote(sent, 'sent')]);
    h.api.on(
      'GET /me/quotes/$sent/pdf',
      (_) async => http.Response.bytes([0x25, 0x50, 0x44, 0x46], 200, headers: {'content-type': 'application/pdf'}),
    );

    await tapButton(tester, 'Save or share the PDF');

    expect(h.files.shared.single.$1, 'Quote.pdf');
    expect(h.files.shared.single.$2, [0x25, 0x50, 0x44, 0x46]);
    expect(h.api.last('GET /me/quotes/$sent/pdf').headers['Authorization'], 'Bearer $token');
  });

  testWidgets('says so when the PDF cannot be read', (tester) async {
    final h = await atQuotes(tester, [quote(sent, 'sent')]);
    h.api.answer('GET /me/quotes/$sent/pdf', 404, error('not_found'));

    await tapButton(tester, 'Save or share the PDF');

    expect(h.files.shared, isEmpty);
    expect(find.text('Something went wrong. Try again.'), findsOneWidget);
  });
}
