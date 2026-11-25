import 'package:flutter/material.dart';
import 'package:intl/intl.dart' show NumberFormat;
import 'package:salus_api/api.dart' show DiagnosticReport, DiagnosticReportResult;

import 'booking.dart';
import 'documents.dart';
import 'files.dart';
import 'l10n/app_localizations.dart';
import 'loading.dart';
import 'messages.dart';
import 'session.dart';
import 'ui.dart' as ui;
import 'zones.dart';

/// The patient's lab results from every clinic, newest first, each with its sample date, the values the clinic typed
/// in (with their reference range and flag as the lab printed them) and the lab's PDF (docs/clinic-cms, package LB2,
/// U5). The server lists only the results a practitioner released; nothing here interprets them. A result carries no
/// clinic zone: the clinics are in Bosnia and Herzegovina.
class LabResultsScreen extends StatefulWidget {
  const LabResultsScreen({super.key, required this.booking, required this.files});

  final Booking booking;
  final Files files;

  @override
  State<LabResultsScreen> createState() => _LabResultsScreenState();
}

class _LabResultsScreenState extends LoadingState<LabResultsScreen, List<DiagnosticReport>> {
  final _zone = Zone('Europe/Sarajevo');
  String? _error;
  String? _busy;

  @override
  Future<List<DiagnosticReport>> load() => widget.booking.labResults();

  Future<void> _share(DiagnosticReport report) async {
    setState(() {
      _busy = report.id;
      _error = null;
    });
    try {
      final bytes = await widget.booking.labResultPdf(report.id);
      await widget.files.sharePdf(pdfName(report.title), bytes);
    } on ApiFailure catch (failure) {
      if (mounted) setState(() => _error = bookingMessage(context.t, failure));
    } finally {
      if (mounted) setState(() => _busy = null);
    }
  }

  @override
  Widget build(BuildContext context) => ui.Page(
    title: context.t.myLabResults,
    children: [
      if (_error != null) ui.Message(_error!, error: true),
      ...body(
        (all) => [
          Text(all.isEmpty ? context.t.noLabResults : context.t.labResultsIntro),
          for (final report in all) _card(report),
        ],
      ),
    ],
  );

  Widget _card(DiagnosticReport report) {
    final t = context.t;
    final effective = report.effectiveDate;
    final released = report.releasedAt;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            ui.Heading(report.title),
            if (effective != null) Text(t.labResultSampleDate(Zone.longDate(effective, t.localeName))),
            if (released != null) Text(t.labResultReleasedOn(_zone.date(released, t.localeName))),
            for (final result in report.results) ..._result(t, result),
            if (report.hasPdf) ...[
              const SizedBox(height: 8),
              FilledButton(
                onPressed: _busy == null ? () => _share(report) : null,
                child: Text(t.sharePdf),
              ),
            ],
          ],
        ),
      ),
    );
  }

  List<Widget> _result(AppLocalizations t, DiagnosticReportResult result) {
    final range = result.referenceRange;
    final flag = result.interpretation;
    return [
      const SizedBox(height: 8),
      Text(t.labResultValue(result.test, _value(t, result)), style: const TextStyle(fontWeight: FontWeight.w600)),
      if (range != null) Text(t.labResultRange(range)),
      if (flag != null) Text(t.labResultFlag(flag.toString())),
    ];
  }

  /// The value as the clinic typed it: a number in the UI language with its UCUM unit (a code, not translated), or the
  /// text.
  static String _value(AppLocalizations t, DiagnosticReportResult result) {
    final number = result.value;
    if (number == null) return result.valueText ?? '';
    final shown = NumberFormat.decimalPattern(t.localeName).format(number);
    final unit = result.unit;
    return unit == null ? shown : '$shown $unit';
  }
}
