import 'package:flutter/material.dart';
import 'package:salus_api/api.dart' as api;

import 'booking.dart';
import 'documents.dart';
import 'files.dart';
import 'language_picker.dart' show languageNames;
import 'loading.dart';
import 'messages.dart';
import 'questionnaire_language.dart' show appLanguage;
import 'session.dart';
import 'ui.dart' as ui;
import 'zones.dart';

// The patient's documents of one appointment (docs/consultation/SPEC.md "Patient documents"): the signed documents
// of its consultation, each opened as text and shared as its PDF, and the transcript copy while the clinic shares
// it. Never a draft: the API shows the patient signed documents only.

typedef _Records = ({List<SignedDocument> documents, List<api.TranscriptEvent>? transcript});

/// The signed documents of the appointment [appointmentId], in the app's language first, and its transcript copy.
class DocumentsScreen extends StatefulWidget {
  const DocumentsScreen({
    super.key,
    required this.booking,
    required this.files,
    required this.appointmentId,
    required this.zone,
  });

  final Booking booking;
  final Files files;
  final String appointmentId;

  /// The clinic's time zone, the dates are shown in.
  final Zone zone;

  @override
  State<DocumentsScreen> createState() => _DocumentsScreenState();
}

class _DocumentsScreenState extends LoadingState<DocumentsScreen, _Records> {
  /// The consultation's records: none before it has an encounter.
  @override
  Future<_Records> load() async {
    final encounter = await widget.booking.fhir.encounterOf(widget.appointmentId);
    if (encounter == null) return (documents: const <SignedDocument>[], transcript: null);
    return (
      documents: await widget.booking.fhir.signedDocuments(encounter),
      transcript: await widget.booking.transcript(encounter),
    );
  }

  void _open(Widget Function() screen) => Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => screen()));

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    return ui.Page(
      title: t.documents,
      children: body(
        (data) => [
          if (data.documents.isEmpty) Text(t.noDocuments),
          for (final d in languageFirst(data.documents, appLanguage(context))) _card(d),
          if (data.transcript case final segments?)
            OutlinedButton(
              onPressed: () => _open(() => TranscriptScreen(files: widget.files, segments: segments)),
              child: Text(t.transcript),
            ),
        ],
      ),
    );
  }

  Widget _card(SignedDocument d) {
    final t = context.t;
    return Card(
      child: ListTile(
        title: Text(d.title),
        subtitle: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(t.documentSignedOn(widget.zone.date(d.signedAt, t.localeName))),
            if (d.language != appLanguage(context)) Text(t.documentLanguage(languageNames[d.language] ?? d.language)),
          ],
        ),
        onTap: () =>
            _open(() => DocumentScreen(booking: widget.booking, files: widget.files, document: d, zone: widget.zone)),
      ),
    );
  }
}

/// One signed document as text, field by field and line by line, each line a Text of its own; its PDF goes to the
/// share sheet.
class DocumentScreen extends StatefulWidget {
  const DocumentScreen({
    super.key,
    required this.booking,
    required this.files,
    required this.document,
    required this.zone,
  });

  final Booking booking;
  final Files files;
  final SignedDocument document;
  final Zone zone;

  @override
  State<DocumentScreen> createState() => _DocumentScreenState();
}

class _DocumentScreenState extends LoadingState<DocumentScreen, DocumentContent> {
  String? _error;
  var _busy = false;

  @override
  Future<DocumentContent> load() => widget.booking.fhir.composition(widget.document.compositionId);

  Future<void> _share() async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final bytes = await widget.booking.pdf(widget.document.id);
      await widget.files.sharePdf(pdfName(widget.document.title), bytes);
    } on ApiFailure catch (failure) {
      if (mounted) setState(() => _error = bookingMessage(context.t, failure));
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    final d = widget.document;
    final heading = Theme.of(context).textTheme.titleSmall;
    return ui.Page(
      title: d.title,
      children: [
        Text(t.documentSignedOn(widget.zone.date(d.signedAt, t.localeName))),
        ...body(
          (content) => [
            for (final s in content.sections)
              Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(s.title, style: heading),
                  for (final line in s.lines) Text(line),
                ],
              ),
            if (_error != null) ui.Message(_error!, error: true),
            FilledButton(onPressed: _busy ? null : _share, child: Text(t.sharePdf)),
          ],
        ),
      ],
    );
  }
}

/// The patient's copy of the consultation's transcript: who spoke, what they said and its translation.
class TranscriptScreen extends StatelessWidget {
  const TranscriptScreen({super.key, required this.files, required this.segments});

  final Files files;
  final List<api.TranscriptEvent> segments;

  static String _speaker(BuildContext context, api.TranscriptEvent s) =>
      s.speaker == api.TranscriptEventSpeakerEnum.patient ? context.t.speakerYou : context.t.speakerPractitioner;

  /// The transcript as plain text for the share sheet, in the same order and words as on the screen.
  String _text(BuildContext context) => [
    for (final s in segments)
      [
        '${_speaker(context, s)}: ${s.original}',
        if (s.translation.isNotEmpty) context.t.transcriptTranslation(s.translation),
      ].join('\n'),
  ].join('\n\n');

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    final heading = Theme.of(context).textTheme.titleSmall;
    return ui.Page(
      title: t.transcript,
      children: [
        Text(t.transcriptNote),
        if (segments.isEmpty) Text(t.transcriptEmpty),
        for (final s in segments)
          Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(_speaker(context, s), style: heading),
              Text(s.original),
              if (s.translation.isNotEmpty) Text(t.transcriptTranslation(s.translation)),
            ],
          ),
        if (segments.isNotEmpty)
          FilledButton(onPressed: () => files.shareText(_text(context)), child: Text(t.shareTranscript)),
      ],
    );
  }
}
