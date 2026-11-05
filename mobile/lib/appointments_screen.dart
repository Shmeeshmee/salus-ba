import 'package:flutter/material.dart';

import 'booking.dart';
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
/// 24 hours before its start.
class AppointmentsScreen extends StatefulWidget {
  const AppointmentsScreen({super.key, required this.booking, required this.now, this.notice});

  final Booking booking;
  final DateTime Function() now;

  /// What the previous screen did, such as a booking.
  final String? notice;

  @override
  State<AppointmentsScreen> createState() => _AppointmentsScreenState();
}

class _AppointmentsScreenState extends LoadingState<AppointmentsScreen, _Appointments> {
  late String? _message = widget.notice;
  String? _error;

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
