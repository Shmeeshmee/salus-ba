import 'package:flutter/material.dart';
import 'package:salus_api/api.dart' as api;

import 'booking.dart';
import 'fhir.dart';
import 'messages.dart';
import 'reschedule_screen.dart';
import 'session.dart';
import 'zones.dart';

/// What the patient may do with an upcoming booked appointment besides cancelling (SAL-34): confirm that they will
/// come, and change its time to another free time of the same practitioner, as `GET /appointments/{id}/actions`
/// says by the clinic's rules. Shows "Confirmed" once confirmed. Nothing more shows while the actions cannot be read:
/// the server decides each action again.
class AppointmentActionsPanel extends StatefulWidget {
  const AppointmentActionsPanel({
    super.key,
    required this.booking,
    required this.now,
    required this.appointment,
    required this.zone,
    required this.onDone,
    required this.onFailed,
    this.clinic,
    this.role,
  });

  final Booking booking;
  final DateTime Function() now;
  final Appointment appointment;
  final Zone zone;

  /// The clinic and the practitioner role of the appointment; without them (a clinic no longer listed) the time
  /// cannot be changed here.
  final Organization? clinic;
  final PractitionerRole? role;

  /// An action succeeded, with the sentence to show; the list is read again.
  final Future<void> Function(String message) onDone;

  /// An action was refused, with the sentence to show.
  final void Function(String message) onFailed;

  @override
  State<AppointmentActionsPanel> createState() => _AppointmentActionsPanelState();
}

class _AppointmentActionsPanelState extends State<AppointmentActionsPanel> {
  api.AppointmentActions? _actions;
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final actions = await widget.booking.actions(widget.appointment.id);
      if (mounted) setState(() => _actions = actions);
    } on ApiFailure {
      // Not offered: the card shows without these actions.
    }
  }

  Future<void> _confirm() async {
    setState(() => _busy = true);
    try {
      await widget.booking.confirm(widget.appointment.id);
      if (mounted) await widget.onDone(context.t.appointmentConfirmedNotice);
    } on ApiFailure catch (failure) {
      if (mounted) widget.onFailed(changeMessage(context.t, failure));
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _reschedule(Organization clinic, PractitionerRole role) async {
    final moved = await Navigator.of(context).push<bool>(
      MaterialPageRoute<bool>(
        builder: (_) => RescheduleScreen(
          booking: widget.booking,
          now: widget.now,
          appointment: widget.appointment,
          clinic: clinic,
          role: role,
        ),
      ),
    );
    if (moved == true && mounted) await widget.onDone(context.t.appointmentMoved);
  }

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    final actions = _actions;
    final until = actions?.rescheduleUntil;
    final clinic = widget.clinic;
    final role = widget.role;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (widget.appointment.confirmed) Text(t.appointmentConfirmedStatus),
        if (actions != null && actions.confirm && !widget.appointment.confirmed) ...[
          const SizedBox(height: 8),
          FilledButton(onPressed: _busy ? null : _confirm, child: Text(t.confirmAttendance)),
        ],
        if (actions != null && actions.reschedule && clinic != null && role != null) ...[
          const SizedBox(height: 8),
          OutlinedButton(onPressed: _busy ? null : () => _reschedule(clinic, role), child: Text(t.changeTime)),
          if (until != null)
            Text(t.rescheduleUntil(widget.zone.date(until, t.localeName), widget.zone.time(until))),
          Text(t.reschedulesLeft(actions.reschedulesLeft)),
        ],
      ],
    );
  }
}
