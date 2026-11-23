import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

import 'support.dart';

const first = 'd1000000-0000-4000-8000-000000000001';
const reversal = 'd1000000-0000-4000-8000-000000000002';

// Synthetic receipts, newest first: a reversal, and the receipt it reverses.
Map<String, Object?> receipt(String id, int number, int totalMinor, {String? reverses}) => {
  'id': id,
  'kind': 'receipt',
  'number': number,
  'issuedAt': '2026-11-02T09:30:00Z',
  'paymentMethod': 'cash',
  'totalMinor': totalMinor,
  'currency': 'BAM',
  'fiscalStatus': 'fiscalized',
  'reversesId': reverses,
  'lines': [
    {'name': 'Placeholder examination', 'quantity': totalMinor < 0 ? -1 : 1, 'unitPriceMinor': 10670, 'discountMinor': 0, 'totalMinor': totalMinor},
  ],
};

Future<Harness> atReceipts(WidgetTester tester, List<Map<String, Object?>> receipts) async {
  final h = Harness(store: FakeTokenStore(token));
  h.api.answer('GET /auth/me', 200, account());
  answerDirectory(h.api);
  h.api.answer('GET /me/receipts', 200, receipts);
  await h.pump(tester);
  await tapOutlined(tester, 'My receipts');
  return h;
}

void main() {
  testWidgets('lists the receipts with their number, date, total and lines, a reversal marked', (tester) async {
    await atReceipts(tester, [receipt(reversal, 8, -10670, reverses: first), receipt(first, 7, 10670)]);

    expect(find.text('Receipt no. 7'), findsOneWidget);
    expect(find.text('Receipt no. 8'), findsOneWidget);
    expect(find.text('Issued on Monday 2 November 2026'), findsNWidgets(2));
    expect(find.text('106.70 BAM'), findsOneWidget);
    expect(find.text('-106.70 BAM'), findsOneWidget);
    expect(find.text('Reversal of an earlier receipt'), findsOneWidget);
    expect(find.text('Placeholder examination × 1'), findsOneWidget);
  });

  testWidgets('says when there are no receipts', (tester) async {
    await atReceipts(tester, const []);

    expect(find.text('You have no receipts yet.'), findsOneWidget);
  });

  testWidgets("shares a receipt's PDF, asked with the session's token", (tester) async {
    final h = await atReceipts(tester, [receipt(first, 7, 10670)]);
    h.api.on(
      'GET /me/receipts/$first/pdf',
      (_) async => http.Response.bytes([0x25, 0x50, 0x44, 0x46], 200, headers: {'content-type': 'application/pdf'}),
    );

    await tapButton(tester, 'Save or share the PDF');

    expect(h.files.shared.single.$1, 'Receipt no. 7.pdf');
    expect(h.files.shared.single.$2, [0x25, 0x50, 0x44, 0x46]);
    expect(h.api.last('GET /me/receipts/$first/pdf').headers['Authorization'], 'Bearer $token');
  });

  testWidgets('says so when the PDF cannot be read', (tester) async {
    final h = await atReceipts(tester, [receipt(first, 7, 10670)]);
    h.api.answer('GET /me/receipts/$first/pdf', 404, error('not_found'));

    await tapButton(tester, 'Save or share the PDF');

    expect(h.files.shared, isEmpty);
    expect(find.text("Something went wrong. Try again."), findsOneWidget);
  });
}
