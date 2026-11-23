import 'package:flutter/material.dart';
import 'package:salus_api/api.dart' show Receipt;

import 'booking.dart';
import 'documents.dart';
import 'files.dart';
import 'loading.dart';
import 'messages.dart';
import 'service_price.dart';
import 'session.dart';
import 'ui.dart' as ui;
import 'zones.dart';

/// The patient's receipts from every clinic, newest first, each with its number, date, total and lines, and its PDF
/// as the clinic printed it (docs/plans/clinic-cms.md, package H). The patient paid at the clinic; nothing here is
/// charged. A receipt carries no clinic zone: the clinics are in Bosnia and Herzegovina.
class ReceiptsScreen extends StatefulWidget {
  const ReceiptsScreen({super.key, required this.booking, required this.files});

  final Booking booking;
  final Files files;

  @override
  State<ReceiptsScreen> createState() => _ReceiptsScreenState();
}

class _ReceiptsScreenState extends LoadingState<ReceiptsScreen, List<Receipt>> {
  final _zone = Zone('Europe/Sarajevo');
  String? _error;
  String? _busy;

  @override
  Future<List<Receipt>> load() => widget.booking.receipts();

  Future<void> _share(Receipt receipt) async {
    final t = context.t;
    setState(() {
      _busy = receipt.id;
      _error = null;
    });
    try {
      final bytes = await widget.booking.receiptPdf(receipt.id);
      await widget.files.sharePdf(pdfName(t.receiptNumber(receipt.number)), bytes);
    } on ApiFailure catch (failure) {
      if (mounted) setState(() => _error = bookingMessage(context.t, failure));
    } finally {
      if (mounted) setState(() => _busy = null);
    }
  }

  @override
  Widget build(BuildContext context) => ui.Page(
    title: context.t.myReceipts,
    children: [
      if (_error != null) ui.Message(_error!, error: true),
      ...body(
        (all) => [
          Text(all.isEmpty ? context.t.noReceipts : context.t.receiptsIntro),
          for (final receipt in all) _card(receipt),
        ],
      ),
    ],
  );

  Widget _card(Receipt receipt) {
    final t = context.t;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            ui.Heading(t.receiptNumber(receipt.number)),
            Text(t.receiptIssuedOn(_zone.date(receipt.issuedAt, t.localeName))),
            if (receipt.reversesId != null) Text(t.receiptReversal),
            for (final line in receipt.lines) Text(t.receiptLine(line.name, line.quantity)),
            Text(priceText(receipt.totalMinor, receipt.currency, t.localeName)),
            const SizedBox(height: 8),
            FilledButton(
              onPressed: _busy == null ? () => _share(receipt) : null,
              child: Text(t.sharePdf),
            ),
          ],
        ),
      ),
    );
  }
}
