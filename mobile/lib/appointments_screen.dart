import 'dart:async';

import 'package:flutter/material.dart';

import 'booking.dart';
import 'call.dart';
import 'call_screen.dart';
import 'fhir.dart';
import 'loading.dart';
import 'messages.dart';
import 'session.dart';
import 'ui.dart' as ui;
import 'zones.dart';

/// The patient may cancel until this long before the start (docs/booking/SPEC.md "Cancellation").
const cancelWindow = Duration(hours: 24);

typedef _Appointments = ({List<Appointment> all, Places places});

/// The patient's appointments, upcoming and past, in each clinic's time zone; a booked one can be cancelled until
/// 24 hours before its start, and joined as a video consultation in its join window.
class AppointmentsScreen extends StatefulWidget {
  const AppointmentsScreen({super.key, required this.booking, required this.now, required this.media, this.notice});

  final Booking booking;
  final DateTime Function() now;
  final Media media;

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

  /// The appointments with the clinics they are in: every booking needs a consent, so the consents name them.
  @override
  Future<_Appointments> load() async {
    final me = await widget.booking.me();
    if (me == null) return (all: const <Appointment>[], places: const Places({}, {}));
    final all = await widget.booking.fhir.appointments(me.id);
    if (all.isEmpty) return (all: all, places: const Places({}, {}));
    final consents = await widget.booking.consents();
    return (all: all, places: await widget.booking.places(consents.map((c) => c.organizationId)));
  }

  Future<void> _cancel(Appointment a, Zone zone) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Cancel this appointment?'),
        content: Text(
          'Your appointment on ${zone.date(a.start)} at ${zone.time(a.start)} will be cancelled. You will get an '
          'e-mail to confirm it.',
        ),
        actions: [
          TextButton(onPressed: () => Navigator.of(context).pop(false), child: const Text('Keep it')),
          FilledButton(onPressed: () => Navigator.of(context).pop(true), child: const Text('Cancel appointment')),
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
      setState(() => _message = appointmentCancelled);
      await reload();
    } on ApiFailure catch (failure) {
      if (mounted) setState(() => _error = bookingMessage(failure));
    }
  }

  @override
  Widget build(BuildContext context) {
    final now = widget.now();
    return ui.Page(
      title: 'My appointments',
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
            if (data.all.isEmpty) const Text('You have no appointments yet.'),
            if (upcoming.isNotEmpty) const ui.Heading('Upcoming'),
            for (final a in upcoming) _card(a, data.places, now),
            if (past.isNotEmpty) const ui.Heading('Past'),
            for (final a in past) _card(a, data.places, now),
          ];
        }),
      ],
    );
  }

  Widget _card(Appointment a, Places places, DateTime now) {
    final place = places.roles[a.roleId];
    final zone = Zone(place?.$1.timeZone ?? 'UTC');
    final open = a.status == 'booked' && a.start.isAfter(now);
    final cancellable = open && a.start.difference(now) >= cancelWindow;
    final window = joinWindow(a, now);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            ui.Heading(zone.date(a.start)),
            Text('${zone.time(a.start)} to ${zone.time(a.end)} (${zone.name})'),
            Text(place == null ? unlistedClinic : [place.$2.name, place.$1.name].where((s) => s.isNotEmpty).join(', ')),
            Text(_status(a.status)),
            if (window == JoinWindow.open) ...[
              const SizedBox(height: 8),
              FilledButton(onPressed: () => _join(a), child: const Text(joinConsultation)),
            ],
            if (window == JoinWindow.before) const Text(joinFrom),
            if (open) ...[
              const SizedBox(height: 8),
              OutlinedButton(
                onPressed: cancellable ? () => _cancel(a, zone) : null,
                child: const Text('Cancel appointment'),
              ),
              if (!cancellable) const Text(cancelWindowHint),
            ],
          ],
        ),
      ),
    );
  }

  static String _status(String status) => switch (status) {
    'booked' => 'Booked',
    'cancelled' => 'Cancelled',
    'fulfilled' => 'Attended',
    'noshow' => 'Not attended',
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
