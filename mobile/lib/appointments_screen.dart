import 'dart:async';

import 'package:flutter/material.dart';

import 'booking.dart';
import 'call.dart';
import 'call_screen.dart';
import 'documents_screen.dart';
import 'fhir.dart';
import 'files.dart';
import 'l10n/app_localizations.dart';
import 'loading.dart';
import 'messages.dart';
import 'questionnaire.dart';
import 'questionnaire_language.dart';
import 'questionnaire_screen.dart';
import 'session.dart';
import 'ui.dart' as ui;
import 'zones.dart';

/// The patient may cancel until this long before the start (docs/booking/SPEC.md "Cancellation").
const cancelWindow = Duration(hours: 24);

typedef _Questionnaires = ({List<Response> responses, Map<String, Version> versions});

typedef _Appointments = ({List<Appointment> all, Places places, _Questionnaires questionnaires});

const _none = (responses: <Response>[], versions: <String, Version>{});

/// The patient's appointments, upcoming and past, in each clinic's time zone; a booked one can be cancelled until
/// 24 hours before its start, and joined as a video consultation in its join window.
class AppointmentsScreen extends StatefulWidget {
  const AppointmentsScreen({
    super.key,
    required this.booking,
    required this.now,
    required this.media,
    required this.files,
    this.notice,
  });

  final Booking booking;
  final DateTime Function() now;
  final Media media;
  final Files files;

  /// What the previous screen did, such as a booking.
  final String? notice;

  @override
  State<AppointmentsScreen> createState() => _AppointmentsScreenState();
}

class _AppointmentsScreenState extends LoadingState<AppointmentsScreen, _Appointments> {
  late String? _message = widget.notice;
  String? _error;

  late final Timer _tick;

  /// Reads the clock again every 15 seconds, so Join appears and goes while the list is open.
  @override
  void initState() {
    super.initState();
    _tick = Timer.periodic(const Duration(seconds: 15), (_) => setState(() {}));
  }

  @override
  void dispose() {
    _tick.cancel();
    super.dispose();
  }

  /// The video consultation; the list is read again after it, as its status may have changed.
  Future<void> _join(Appointment a) async {
    await Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => CallScreen(booking: widget.booking, media: widget.media, appointmentId: a.id),
      ),
    );
    if (mounted) await reload();
  }

  /// The signed documents of an appointment that has started (SAL-17).
  void _documents(Appointment a, Zone zone) => Navigator.of(context).push(
    MaterialPageRoute<void>(
      builder: (_) => DocumentsScreen(booking: widget.booking, files: widget.files, appointmentId: a.id, zone: zone),
    ),
  );

  /// The appointments with the clinics they are in: every booking needs a consent, so the consents name them.
  /// With the questionnaires asked for them and the versions they answer.
  @override
  Future<_Appointments> load() async {
    final me = await widget.booking.me();
    if (me == null) return (all: const <Appointment>[], places: const Places({}, {}), questionnaires: _none);
    final all = await widget.booking.fhir.appointments(me.id);
    if (all.isEmpty) return (all: all, places: const Places({}, {}), questionnaires: _none);
    final consents = await widget.booking.consents();
    return (
      all: all,
      places: await widget.booking.places(consents.map((c) => c.organizationId)),
      questionnaires: await _questionnaires(me.id),
    );
  }

  /// The questionnaires, or none when they cannot be read: the appointments (and Join) still show.
  Future<_Questionnaires> _questionnaires(String patientId) async {
    try {
      return await widget.booking.fhir.questionnaireResponses(patientId);
    } on ApiFailure {
      return _none;
    }
  }

  /// The questionnaire screen; the list is read again after it, as the response's status may have changed.
  Future<void> _fill(Response r, Version v, String clinic, Zone zone) async {
    await Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => QuestionnaireLanguage(
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
      ),
    );
    if (mounted) await reload();
  }

  Future<void> _cancel(Appointment a, Zone zone) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(context.t.cancelTitle),
        content: Text(context.t.cancelBody(zone.date(a.start, context.t.localeName), zone.time(a.start))),
        actions: [
          TextButton(onPressed: () => Navigator.of(context).pop(false), child: Text(context.t.keepIt)),
          FilledButton(onPressed: () => Navigator.of(context).pop(true), child: Text(context.t.cancelAppointment)),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    setState(() {
      _message = null;
      _error = null;
    });
    try {
      await widget.booking.cancel(a.id);
      if (!mounted) return;
      setState(() => _message = context.t.appointmentCancelled);
      await reload();
    } on ApiFailure catch (failure) {
      if (mounted) setState(() => _error = bookingMessage(context.t, failure));
    }
  }

  @override
  Widget build(BuildContext context) {
    final now = widget.now();
    final t = context.t;
    return ui.Page(
      title: t.myAppointments,
      children: [
        if (_message != null) ui.Message(_message!),
        if (_error != null) ui.Message(_error!, error: true),
        ...body((data) {
          final upcoming = [
            for (final a in data.all)
              if (a.end.isAfter(now)) a,
          ];
          final past = [
            for (final a in data.all.reversed)
              if (!a.end.isAfter(now)) a,
          ];
          return [
            if (data.all.isEmpty) Text(t.noAppointments),
            if (upcoming.isNotEmpty) ui.Heading(t.upcoming),
            for (final a in upcoming) _card(a, data, now),
            if (past.isNotEmpty) ui.Heading(t.past),
            for (final a in past) _card(a, data, now),
          ];
        }),
      ],
    );
  }

  Widget _card(Appointment a, _Appointments data, DateTime now) {
    final place = data.places.roles[a.roleId];
    final zone = Zone(place?.$1.timeZone ?? 'UTC');
    final open = a.status == 'booked' && a.start.isAfter(now);
    final cancellable = open && a.start.difference(now) >= cancelWindow;
    final window = joinWindow(a, now);
    final t = context.t;
    final clinic = place?.$1.name ?? t.unlistedClinic;
    final questionnaires = a.status == 'cancelled'
        ? const <(Response, Version)>[]
        : [
            for (final r in data.questionnaires.responses)
              if (r.appointmentId == a.id && data.questionnaires.versions[r.questionnaire] != null)
                (r, data.questionnaires.versions[r.questionnaire]!),
          ];
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            ui.Heading(zone.date(a.start, t.localeName)),
            Text(t.timeRange(zone.time(a.start), zone.time(a.end), zone.name)),
            Text(
              place == null ? t.unlistedClinic : [place.$2.name, place.$1.name].where((s) => s.isNotEmpty).join(', '),
            ),
            Text(_status(t, a.status)),
            if (window == JoinWindow.open) ...[
              const SizedBox(height: 8),
              FilledButton(onPressed: () => _join(a), child: Text(t.joinConsultation)),
            ],
            if (window == JoinWindow.before) Text(t.joinFrom),
            if (open) ...[
              const SizedBox(height: 8),
              OutlinedButton(onPressed: cancellable ? () => _cancel(a, zone) : null, child: Text(t.cancelAppointment)),
              if (!cancellable) Text(t.cancelWindowHint),
            ],
            if (a.status != 'cancelled' && !a.start.isAfter(now)) ...[
              const SizedBox(height: 8),
              OutlinedButton(onPressed: () => _documents(a, zone), child: Text(t.documents)),
            ],
            for (final (r, v) in questionnaires) ...[
              const SizedBox(height: 16),
              _questionnaire(a, r, v, now, () => _fill(r, v, clinic, zone)),
            ],
          ],
        ),
      ),
    );
  }

  /// A questionnaire asked for the appointment (M1 to M4): to fill in, in progress, submitted, or closed once the
  /// appointment has started without it.
  Widget _questionnaire(Appointment a, Response r, Version v, DateTime now, VoidCallback open) {
    final submitted = r.status == ResponseStatus.completed;
    final closed = !submitted && !a.start.isAfter(now);
    final started = r.answers.isNotEmpty;
    final t = context.t;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(_title(v, r), style: Theme.of(context).textTheme.titleSmall),
        Text(
          submitted
              ? t.submittedStatus
              : closed
              ? t.closedStatus
              : started
              ? t.inProgress
              : t.toFillIn,
        ),
        if (closed) Text(t.questionnaireClosed),
        if (!submitted && !closed) ...[
          Text(t.submitBeforeStart),
          const SizedBox(height: 8),
          FilledButton(onPressed: open, child: Text(started ? t.continueFilling : t.fillIn)),
        ],
        if (submitted) ...[const SizedBox(height: 8), OutlinedButton(onPressed: open, child: Text(t.viewAnswers))],
      ],
    );
  }

  /// The title in the language the questionnaire opens in.
  String _title(Version v, Response r) {
    final tag = readIn(v, r, appLanguage(context));
    return tag == null ? v.title : v.inLanguage(tag).title;
  }

  static String _status(AppLocalizations t, String status) => switch (status) {
    'booked' => t.statusBooked,
    'cancelled' => t.statusCancelled,
    'fulfilled' => t.statusAttended,
    'noshow' => t.statusNotAttended,
    _ => status,
  };
}

enum JoinWindow { before, open, closed }

/// Where [now] stands against a booked appointment's join window: from 10 minutes before the start to 30 minutes
/// after the end (docs/consultation/SPEC.md "Join button"). It only decides whether Join shows; the server decides
/// whether a token is issued.
JoinWindow joinWindow(Appointment a, DateTime now) {
  if (a.status != 'booked') return JoinWindow.closed;
  if (now.isBefore(a.start.subtract(const Duration(minutes: 10)))) return JoinWindow.before;
  if (!now.isBefore(a.end.add(const Duration(minutes: 30)))) return JoinWindow.closed;
  return JoinWindow.open;
}
