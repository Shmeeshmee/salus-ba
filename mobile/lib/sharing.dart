import 'package:flutter/material.dart';
import 'package:salus_api/api.dart' as api;

import 'booking.dart';
import 'fhir.dart';
import 'l10n/app_localizations.dart';
import 'loading.dart';
import 'messages.dart';
import 'session.dart';
import 'ui.dart' as ui;
import 'zones.dart';

/// The patient's choice of previous visits at other clinics to share with one clinic (SAL-42, ADR 0014): each visit
/// as the clinic will see it (its date, clinic, specialty and title), with a checkbox, select all and select none.
/// Nothing is ticked unless it is shared now; an unticked visit is not shown to the clinic at all.
class SharingPicker extends StatelessWidget {
  const SharingPicker({
    super.key,
    required this.clinicName,
    required this.candidates,
    required this.selected,
    required this.onChanged,
  });

  final String clinicName;
  final List<api.SharingCandidate> candidates;
  final Set<String> selected;
  final ValueChanged<Set<String>> onChanged;

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    if (candidates.isEmpty) return Text(t.sharingNoVisits);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(t.sharingIntro(clinicName)),
        Text(t.sharingChangeLater),
        Wrap(
          spacing: 8,
          children: [
            TextButton(
              onPressed: () => onChanged({for (final c in candidates) c.encounterId}),
              child: Text(t.sharingSelectAll),
            ),
            TextButton(onPressed: () => onChanged(const {}), child: Text(t.sharingSelectNone)),
          ],
        ),
        for (final c in candidates) _visit(t, c),
      ],
    );
  }

  Widget _visit(AppLocalizations t, api.SharingCandidate c) {
    final specialty = specialtyOfCode(t, c.specialty);
    final title = c.title;
    return CheckboxListTile(
      value: selected.contains(c.encounterId),
      onChanged: (on) => onChanged(on == true ? {...selected, c.encounterId} : ({...selected}..remove(c.encounterId))),
      title: Text(Zone(c.timeZone).date(c.date, t.localeName)), // the visit's own clinic's day
      subtitle: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(c.organizationName),
          if (specialty != null) Text(specialty),
          if (title != null && title.isNotEmpty) Text(title),
        ],
      ),
    );
  }
}

typedef _Clinics = ({List<String> clinics, Map<String, Organization> places, Map<String, api.HistorySharing> sharing});

/// Shared history (SAL-42): each clinic the patient has an active care consent with, and what of their history from
/// other clinics they share with it; each opens [SharingScreen].
class SharedHistoryScreen extends StatefulWidget {
  const SharedHistoryScreen({super.key, required this.booking});

  final Booking booking;

  @override
  State<SharedHistoryScreen> createState() => _SharedHistoryScreenState();
}

class _SharedHistoryScreenState extends LoadingState<SharedHistoryScreen, _Clinics> {
  @override
  Future<_Clinics> load() async {
    final consents = await widget.booking.consents();
    final clinics = [
      for (final c in consents)
        if (c.kind == api.ConsentKind.care && c.status == api.ConsentStatusEnum.active) c.organizationId,
    ];
    if (clinics.isEmpty) {
      return (clinics: clinics, places: const <String, Organization>{}, sharing: const <String, api.HistorySharing>{});
    }
    final sharing = {for (final s in await widget.booking.mySharing()) s.organizationId: s};
    final places = {for (final c in await widget.booking.fhir.listedClinics()) c.id: c};
    return (clinics: clinics, places: places, sharing: sharing);
  }

  Future<void> _open(String clinicId, Organization? clinic) async {
    await Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => SharingScreen(
          booking: widget.booking,
          clinicId: clinicId,
          clinicName: clinic?.name ?? context.t.unlistedClinic,
        ),
      ),
    );
    if (mounted) reload();
  }

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    return ui.Page(
      title: t.sharedHistory,
      children: body(
        (data) => [
          Text(data.clinics.isEmpty ? t.sharedHistoryNone : t.sharedHistoryIntro),
          for (final id in data.clinics)
            ui.Choice(
              data.places[id]?.name ?? t.unlistedClinic,
              subtitle: t.sharedVisits(data.sharing[id]?.shared.length ?? 0),
              onTap: () => _open(id, data.places[id]),
            ),
        ],
      ),
    );
  }
}

/// The visits the patient may share with a clinic, and the version of what they share with it now (null for nothing).
typedef SharingChoice = ({List<api.SharingCandidate> candidates, String? version});

/// The visits shared with one clinic, to change (saved at the version read) or stop sharing.
class SharingScreen extends StatefulWidget {
  const SharingScreen({
    super.key,
    required this.booking,
    required this.clinicId,
    required this.clinicName,
  });

  final Booking booking;
  final String clinicId;
  final String clinicName;

  @override
  State<SharingScreen> createState() => _SharingScreenState();
}

class _SharingScreenState extends LoadingState<SharingScreen, SharingChoice> {
  Set<String> _selected = {};
  String? _message;
  String? _error;
  bool _busy = false;

  @override
  Future<SharingChoice> load() async {
    final choice = await sharingChoice(widget.booking, widget.clinicId);
    _selected = {
      for (final c in choice.candidates)
        if (c.selected) c.encounterId,
    };
    return choice;
  }

  Future<void> _save(String? version) => _run(
    () => widget.booking.setSharing(widget.clinicId, [
      for (final c in data!.candidates)
        if (_selected.contains(c.encounterId)) c.encounterId,
    ], version),
    context.t.sharingSaved,
  );

  Future<void> _stop() async {
    final t = context.t;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(t.sharingStopTitle),
        content: Text(t.sharingStopBody(widget.clinicName)),
        actions: [
          TextButton(onPressed: () => Navigator.of(context).pop(false), child: Text(t.sharingKeep)),
          FilledButton(onPressed: () => Navigator.of(context).pop(true), child: Text(t.sharingStop)),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    await _run(() => widget.booking.stopSharing(widget.clinicId), t.sharingStopped);
  }

  Future<void> _run(Future<void> Function() call, String done) async {
    setState(() {
      _busy = true;
      _message = null;
      _error = null;
    });
    try {
      await call();
      if (mounted) setState(() => _message = done);
    } on ApiFailure catch (failure) {
      if (mounted) {
        setState(
          () => _error = failure.code == 'version_conflict'
              ? context.t.sharingChanged
              : bookingMessage(context.t, failure),
        );
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
    if (mounted) await reload();
  }

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    return ui.Page(
      title: widget.clinicName,
      children: [
        if (_message != null) ui.Message(_message!),
        if (_error != null) ui.Message(_error!, error: true),
        ...body(
          (data) => [
            SharingPicker(
              clinicName: widget.clinicName,
              candidates: data.candidates,
              selected: _selected,
              onChanged: (s) => setState(() => _selected = s),
            ),
            if (data.candidates.isNotEmpty) ui.SubmitButton(t.save, busy: _busy, onPressed: () => _save(data.version)),
            if (data.version != null) OutlinedButton(onPressed: _busy ? null : _stop, child: Text(t.sharingStop)),
          ],
        ),
      ],
    );
  }
}

/// The [SharingChoice] of the clinic: the candidates and the version of the sharing, read together, so the version
/// is the one the ticks show.
Future<SharingChoice> sharingChoice(Booking booking, String clinicId) async {
  final preview = await booking.sharingPreview(clinicId);
  return (candidates: preview.candidates, version: preview.versionId);
}
