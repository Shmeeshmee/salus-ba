import 'package:flutter/material.dart';
import 'package:salus_api/api.dart' show Quote, QuoteState;

import 'booking.dart';
import 'documents.dart';
import 'fhir.dart';
import 'files.dart';
import 'loading.dart';
import 'messages.dart';
import 'service_price.dart';
import 'session.dart';
import 'ui.dart' as ui;
import 'zones.dart';

/// The patient's treatment quotes from every clinic, newest first, each with its date, validity, lines, note and
/// total, and its PDF as the clinic printed it (docs/clinic-cms, package X1). A quote is not a receipt: nothing is
/// charged, which the screen says above the list. Each card names its clinic from the directory (the listed FHIR
/// Organizations, as the consents screen does). A quote carries no clinic zone: the clinics are in Bosnia and
/// Herzegovina.
class QuotesScreen extends StatefulWidget {
  const QuotesScreen({super.key, required this.booking, required this.files});

  final Booking booking;
  final Files files;

  @override
  State<QuotesScreen> createState() => _QuotesScreenState();
}

typedef _Quotes = ({List<Quote> all, Map<String, Organization> clinics});

class _QuotesScreenState extends LoadingState<QuotesScreen, _Quotes> {
  final _zone = Zone('Europe/Sarajevo');
  String? _error;
  String? _busy;

  @override
  Future<_Quotes> load() async {
    final all = await widget.booking.quotes();
    if (all.isEmpty) return (all: all, clinics: const <String, Organization>{});
    return (all: all, clinics: {for (final c in await widget.booking.fhir.listedClinics()) c.id: c});
  }

  Future<void> _share(Quote quote) async {
    final t = context.t;
    setState(() {
      _busy = quote.id;
      _error = null;
    });
    try {
      final bytes = await widget.booking.quotePdf(quote.id);
      await widget.files.sharePdf(pdfName(t.quoteTitle), bytes);
    } on ApiFailure catch (failure) {
      if (mounted) setState(() => _error = bookingMessage(context.t, failure));
    } finally {
      if (mounted) setState(() => _busy = null);
    }
  }

  @override
  Widget build(BuildContext context) => ui.Page(
    title: context.t.myQuotes,
    children: [
      if (_error != null) ui.Message(_error!, error: true),
      ...body(
        (data) => [
          Text(data.all.isEmpty ? context.t.noQuotes : context.t.quotesIntro),
          for (final quote in data.all) _card(quote, data.clinics[quote.organizationId]),
        ],
      ),
    ],
  );

  Widget _card(Quote quote, Organization? clinic) {
    final t = context.t;
    final note = quote.note ?? '';
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            ui.Heading(t.quoteTitle),
            Text(clinic?.name ?? t.unlistedClinic),
            Text(t.receiptIssuedOn(_zone.date(quote.issuedAt, t.localeName))),
            Text(
              quote.state == QuoteState.expired
                  ? t.quoteExpired
                  : t.quoteValidUntil(_zone.date(quote.validUntil, t.localeName)),
            ),
            for (final line in quote.lines) Text(t.receiptLine(line.name, line.quantity)),
            if (note.isNotEmpty) Text(note),
            Text(priceText(quote.totalMinor, quote.currency, t.localeName)),
            const SizedBox(height: 8),
            FilledButton(
              onPressed: _busy == null ? () => _share(quote) : null,
              child: Text(t.sharePdf),
            ),
          ],
        ),
      ),
    );
  }
}
