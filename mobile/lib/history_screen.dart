import 'dart:async';

import 'package:flutter/material.dart';
import 'package:salus_api/api.dart' as api;

import 'booking.dart';
import 'call.dart';
import 'documents.dart';
import 'documents_screen.dart';
import 'files.dart';
import 'l10n/app_localizations.dart';
import 'messages.dart';
import 'questionnaire_language.dart';
import 'questionnaire_screen.dart';
import 'session.dart';
import 'ui.dart' as ui;
import 'zones.dart';

// The patient's history across every clinic (SAL-39, docs/plans/patient-history.md): their visits, signed
// documents, completed questionnaire answers and the files of those, newest first, one page at a time as the list
// scrolls; filters by clinic, type and date; each item opens where the app already shows it. Never a draft: the API
// lists none. Chats are not listed yet.

/// The types the filter offers, in order: chats (`communication`) are not served yet.
const historyTypes = [
  api.HistoryItemType.visit,
  api.HistoryItemType.document,
  api.HistoryItemType.questionnaireResponse,
  api.HistoryItemType.attachment,
];

/// The patient's history, newest first.
class HistoryScreen extends StatefulWidget {
  const HistoryScreen({
    super.key,
    required this.booking,
    required this.media,
    required this.files,
    required this.now,
  });

  final Booking booking;
  final Media media;
  final Files files;
  final DateTime Function() now;

  @override
  State<HistoryScreen> createState() => _HistoryScreenState();
}

class _HistoryScreenState extends State<HistoryScreen> {
  final _items = <api.HistoryItem>[];
  String? _next;
  var _loading = false;

  /// The first page failed (the whole screen says so), or a later one did (said under the list).
  String? _error;
  var _moreFailed = false;
  String? _openFailed;

  /// The clinics the filter offers, by id, and the zones of the listed ones.
  final _clinics = <String, String>{};
  var _zones = <String, String>{};

  String? _clinic;
  final _types = <api.HistoryItemType>{};
  DateTimeRange? _range; // local days, the end included

  /// Each load's number: an answer to a load the filter has since replaced is dropped.
  var _generation = 0;

  @override
  void initState() {
    super.initState();
    unawaited(_loadClinics());
    unawaited(_reload());
  }

  /// The clinics of the patient's consents (every booking needs one), named as listed; the history's own items
  /// name an unlisted one as they load.
  Future<void> _loadClinics() async {
    try {
      final ids = [for (final c in await widget.booking.consents()) c.organizationId];
      final places = await widget.booking.places(const []);
      if (!mounted) return;
      setState(() {
        for (final id in ids) {
          _clinics.putIfAbsent(id, () => places.clinics[id]?.name ?? '');
        }
        _zones = {for (final c in places.clinics.values) c.id: c.timeZone};
      });
    } on ApiFailure {
      // The filter offers the clinics the items name; the dates show in UTC until the zones are known.
    }
  }

  Future<void> _reload() {
    _items.clear();
    _next = null;
    _error = null;
    return _more(first: true);
  }

  /// The next page, after the last one read.
  Future<void> _more({bool first = false}) async {
    final generation = first ? ++_generation : _generation;
    setState(() {
      _loading = true;
      _moreFailed = false;
    });
    try {
      final range = _range;
      final page = await widget.booking.history(
        cursor: first ? null : _next,
        clinicId: _clinic,
        types: _types.isEmpty ? null : [for (final t in historyTypes) if (_types.contains(t)) t],
        from: range?.start,
        to: range?.end.add(const Duration(days: 1)),
      );
      if (!mounted || generation != _generation) return;
      setState(() {
        _items.addAll(page.items);
        _next = page.next;
        for (final it in page.items) {
          if ((_clinics[it.clinic.id] ?? '').isEmpty) _clinics[it.clinic.id] = it.clinic.name;
        }
      });
    } on ApiFailure catch (failure) {
      if (!mounted || generation != _generation) return;
      setState(() => first ? _error = bookingMessage(context.t, failure) : _moreFailed = true);
    } finally {
      if (mounted && generation == _generation) setState(() => _loading = false);
    }
  }

  void _filter(void Function() change) {
    setState(change);
    unawaited(_reload());
  }

  Future<void> _pickDates() async {
    final now = widget.now();
    final picked = await showDateRangePicker(
      context: context,
      firstDate: DateTime(2000),
      lastDate: DateTime(now.year, now.month, now.day),
      initialDateRange: _range,
    );
    if (picked != null) _filter(() => _range = picked);
  }

  /// Loads the next page once the list is scrolled near its end ("Show more" does it too, for a list that does not
  /// scroll).
  bool _scrolled(ScrollUpdateNotification n) {
    if (n.metrics.extentAfter < 400 && _next != null && !_loading && !_moreFailed) unawaited(_more());
    return false;
  }

  Zone _zone(String clinicId) => Zone(_zones[clinicId] ?? 'UTC');

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    return NotificationListener<ScrollUpdateNotification>(
      onNotification: _scrolled,
      child: ui.Page(
        title: t.myHistory,
        children: [
          ..._filters(t),
          if (_error != null)
            ui.LoadFailed(_error!, onRetry: _reload)
          else ...[
            if (_items.isEmpty && !_loading && _next == null) Text(_filtered ? t.historyNoMatch : t.historyEmpty),
            for (final it in _items) _card(t, it),
            if (_loading) const ui.Loading(),
            if (_moreFailed) ...[
              ui.Message(t.historyMoreFailed, error: true),
              OutlinedButton(onPressed: _more, child: Text(t.tryAgain)),
            ] else if (_next != null && !_loading)
              TextButton(onPressed: _more, child: Text(t.historyShowMore)),
            if (_openFailed != null) ui.Message(_openFailed!, error: true),
          ],
        ],
      ),
    );
  }

  bool get _filtered => _clinic != null || _types.isNotEmpty || _range != null;

  List<Widget> _filters(AppLocalizations t) {
    final range = _range;
    final locale = t.localeName;
    return [
      DropdownButtonFormField<String?>(
        initialValue: _clinic,
        isExpanded: true,
        decoration: InputDecoration(labelText: t.historyClinic),
        items: [
          DropdownMenuItem(value: null, child: Text(t.historyAllClinics)),
          for (final e in _clinics.entries)
            DropdownMenuItem(
              value: e.key,
              child: Text(e.value.isEmpty ? t.unlistedClinic : e.value, overflow: TextOverflow.ellipsis),
            ),
        ],
        onChanged: (id) => _filter(() => _clinic = id),
      ),
      Wrap(
        spacing: 8,
        runSpacing: 8,
        children: [
          for (final type in historyTypes)
            FilterChip(
              label: Text(_typeName(t, type)),
              selected: _types.contains(type),
              onSelected: (on) => _filter(() => on ? _types.add(type) : _types.remove(type)),
            ),
        ],
      ),
      Row(
        children: [
          Expanded(
            child: OutlinedButton(
              onPressed: _pickDates,
              child: Text(
                range == null
                    ? t.historyAnyDate
                    : t.historyDateRange(
                        Zone.longDate(DateTime.utc(range.start.year, range.start.month, range.start.day), locale),
                        Zone.longDate(DateTime.utc(range.end.year, range.end.month, range.end.day), locale),
                      ),
              ),
            ),
          ),
          if (range != null) TextButton(onPressed: () => _filter(() => _range = null), child: Text(t.historyClearDates)),
        ],
      ),
    ];
  }

  static String _typeName(AppLocalizations t, api.HistoryItemType type) => switch (type) {
    api.HistoryItemType.visit => t.historyVisits,
    api.HistoryItemType.document => t.documents,
    api.HistoryItemType.questionnaireResponse => t.historyAnswers,
    _ => t.historyFiles,
  };

  Widget _card(AppLocalizations t, api.HistoryItem it) {
    final clinic = it.clinic.name.isEmpty ? t.unlistedClinic : it.clinic.name;
    final title = switch (it.type) {
      api.HistoryItemType.visit => t.historyVisit,
      api.HistoryItemType.questionnaireResponse => t.historyAnswers,
      api.HistoryItemType.attachment => (it.title ?? '').isEmpty ? t.historyFile : it.title!,
      _ => (it.title ?? '').isEmpty ? t.documents : it.title!,
    };
    return Card(
      child: ListTile(
        title: Text(title),
        subtitle: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(_zone(it.clinic.id).date(it.date, t.localeName)),
            Text(specialtyAtClinic(t, specialtyOf(t, it.specialty), clinic)),
          ],
        ),
        onTap: () => _open(it, clinic),
      ),
    );
  }

  /// Opens the item where the app shows it: a visit's documents, a signed document, the answers, the file in the
  /// device's viewer.
  Future<void> _open(api.HistoryItem it, String clinic) async {
    setState(() => _openFailed = null);
    final id = it.reference.substring(it.reference.indexOf('/') + 1);
    final zone = _zone(it.clinic.id);
    try {
      switch (it.type) {
        case api.HistoryItemType.visit:
          _push(() => DocumentsScreen(booking: widget.booking, files: widget.files, encounterId: id, zone: zone));
        case api.HistoryItemType.document:
          await _openDocument(id, it.encounterId ?? '', zone);
        case api.HistoryItemType.questionnaireResponse:
          await _openAnswers(id, clinic, zone);
        case api.HistoryItemType.attachment:
          final (bytes, type) = await widget.booking.file(id);
          await widget.files.shareFile(it.title ?? '', bytes, type);
        default:
      }
    } on ApiFailure {
      if (mounted) setState(() => _openFailed = context.t.historyOpenFailed);
    }
  }

  void _push(Widget Function() screen) =>
      unawaited(Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => screen())));

  Future<void> _openDocument(String compositionId, String encounterId, Zone zone) async {
    final all = await widget.booking.fhir.signedDocuments(encounterId);
    final SignedDocument? doc = all.where((d) => d.compositionId == compositionId).firstOrNull;
    if (doc == null) throw const ApiFailure(status: 404);
    if (mounted) {
      _push(() => DocumentScreen(booking: widget.booking, files: widget.files, document: doc, zone: zone));
    }
  }

  Future<void> _openAnswers(String responseId, String clinic, Zone zone) async {
    final r = await widget.booking.fhir.questionnaireResponse(responseId);
    final v = await widget.booking.fhir.questionnaireVersion(r.questionnaire);
    if (v == null) throw const ApiFailure(status: 404);
    if (!mounted) return;
    _push(
      () => QuestionnaireLanguage(
        booking: widget.booking,
        version: v,
        response: r,
        builder: (v, r) => QuestionnaireScreen(
          booking: widget.booking,
          media: widget.media,
          files: widget.files,
          now: widget.now,
          version: v,
          response: r,
          clinic: clinic,
          zone: zone,
        ),
      ),
    );
  }
}
