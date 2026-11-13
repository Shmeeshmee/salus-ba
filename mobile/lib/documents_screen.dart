import 'package:flutter/material.dart';
import 'package:salus_api/api.dart' as api;

import 'booking.dart';
import 'documents.dart';
import 'explain.dart';
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

/// The signed documents of the appointment [appointmentId], or of the visit [encounterId] (the patient's history,
/// SAL-39), in the app's language first, and its transcript copy.
class DocumentsScreen extends StatefulWidget {
  const DocumentsScreen({
    super.key,
    required this.booking,
    required this.files,
    this.appointmentId,
    this.encounterId,
    required this.zone,
  }) : assert((appointmentId == null) != (encounterId == null), 'an appointment or a visit');

  final Booking booking;
  final Files files;
  final String? appointmentId;
  final String? encounterId;

  /// The clinic's time zone, the dates are shown in.
  final Zone zone;

  @override
  State<DocumentsScreen> createState() => _DocumentsScreenState();
}

class _DocumentsScreenState extends LoadingState<DocumentsScreen, _Records> {
  /// The consultation's records: none before it has an encounter.
  @override
  Future<_Records> load() async {
    final encounter = widget.encounterId ?? await widget.booking.fhir.encounterOf(widget.appointmentId!);
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
              onPressed: () =>
                  _open(() => TranscriptScreen(booking: widget.booking, files: widget.files, segments: segments)),
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

typedef _Document = ({DocumentContent content, api.DocumentTerms? terms});

class _DocumentScreenState extends LoadingState<DocumentScreen, _Document> with Explaining {
  String? _error;
  var _busy = false;

  @override
  Booking get booking => widget.booking;

  /// The document and its explained terms (SAL-13); without the terms when they cannot be read, as plain text.
  @override
  Future<_Document> load() async {
    final id = widget.document.compositionId;
    final terms = widget.booking.documentTerms(id).then<api.DocumentTerms?>((t) => t, onError: (Object _) => null);
    return (content: await widget.booking.fhir.composition(id), terms: await terms);
  }

  /// The spans of line [j] of section [i]; none when the terms do not have it.
  List<api.TermSpan> _spans(api.DocumentTerms? terms, int i, int j) {
    final sections = terms?.sections ?? const <api.DocumentTermsSection>[];
    if (i >= sections.length || j >= sections[i].lines.length) return const [];
    return sections[i].lines[j];
  }

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
          (doc) => [
            for (final (i, s) in doc.content.sections.indexed)
              Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(s.title, style: heading),
                  for (final (j, line) in s.lines.indexed)
                    TermLine(line, spans: _spans(doc.terms, i, j), explanations: explanations),
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

/// The patient's copy of the consultation's transcript: who spoke, what they said and its translation, the
/// explained terms of each opening their explanation on a tap (SAL-13).
class TranscriptScreen extends StatefulWidget {
  const TranscriptScreen({super.key, required this.booking, required this.files, required this.segments});

  final Booking booking;
  final Files files;
  final List<api.TranscriptEvent> segments;

  @override
  State<TranscriptScreen> createState() => _TranscriptScreenState();
}

class _TranscriptScreenState extends State<TranscriptScreen> with Explaining {
  @override
  Booking get booking => widget.booking;

  static String _speaker(BuildContext context, api.TranscriptEvent s) =>
      s.speaker == api.TranscriptEventSpeakerEnum.patient ? context.t.speakerYou : context.t.speakerPractitioner;

  /// The transcript as plain text for the share sheet, in the same order and words as on the screen.
  String _text(BuildContext context) => [
    for (final s in widget.segments)
      [
        '${_speaker(context, s)}: ${s.original}',
        if (s.translation.isNotEmpty) context.t.transcriptTranslation(s.translation),
      ].join('\n'),
  ].join('\n\n');

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    final heading = Theme.of(context).textTheme.titleSmall;
    // The translation line's label around its text, so the text's spans keep their offsets.
    final (before, after) = switch (t.transcriptTranslation('\u0000').split('\u0000')) {
      [final b, final a] => (b, a),
      _ => ('', ''),
    };
    return ui.Page(
      title: t.transcript,
      children: [
        Text(t.transcriptNote),
        if (widget.segments.isEmpty) Text(t.transcriptEmpty),
        for (final s in widget.segments)
          Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(_speaker(context, s), style: heading),
              TermLine(s.original, spans: s.terms?.original ?? const [], explanations: explanations),
              if (s.translation.isNotEmpty)
                TermLine(
                  s.translation,
                  spans: s.terms?.translation ?? const [],
                  explanations: explanations,
                  prefix: before,
                  suffix: after,
                ),
            ],
          ),
        if (widget.segments.isNotEmpty)
          FilledButton(onPressed: () => widget.files.shareText(_text(context)), child: Text(t.shareTranscript)),
      ],
    );
  }
}
