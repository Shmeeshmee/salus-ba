import 'package:flutter/material.dart';

import 'booking.dart';
import 'fhir.dart';
import 'loading.dart';
import 'messages.dart';
import 'session.dart';
import 'ui.dart' as ui;
import 'zones.dart';

/// How far ahead slots exist: the scheduler generates 28 days (docs/booking/SPEC.md "Slot generation").
const _days = 28;

/// A time that starts sooner than this is not offered, as when booking (slots_screen.dart).
const _notBefore = Duration(minutes: 5);

/// The free times of the appointment's own practitioner role, one day at a time in the clinic's zone, to move the
/// appointment to (SAL-34). Moving asks for a confirmation; the screen closes with true once moved.
class RescheduleScreen extends StatefulWidget {
  const RescheduleScreen({
    super.key,
    required this.booking,
    required this.now,
    required this.appointment,
    required this.clinic,
    required this.role,
  });

  final Booking booking;
  final DateTime Function() now;
  final Appointment appointment;
  final Organization clinic;
  final PractitionerRole role;

  @override
  State<RescheduleScreen> createState() => _RescheduleScreenState();
}

class _RescheduleScreenState extends LoadingState<RescheduleScreen, List<Slot>?> {
  late final Zone _zone = Zone(widget.clinic.timeZone);
  late final DateTime _today = _zone.day(widget.now());
  late DateTime _day = _today;
  bool _scheduleRead = false;
  String? _schedule;
  String? _error;
  bool _busy = false;

  /// The day's free slots of the role, or null when it has no schedule.
  @override
  Future<List<Slot>?> load() async {
    if (!_scheduleRead) {
      _schedule = await widget.booking.fhir.scheduleOf(widget.role.id);
      _scheduleRead = true;
    }
    final schedule = _schedule;
    if (schedule == null) return null;
    final next = DateTime.utc(_day.year, _day.month, _day.day + 1);
    final earliest = widget.now().add(_notBefore);
    final slots = await widget.booking.fhir.freeSlots(schedule, _zone.startOf(_day), _zone.startOf(next));
    return [
      for (final s in slots)
        if (!s.start.isBefore(earliest)) s,
    ];
  }

  void _move(int days) {
    _day = DateTime.utc(_day.year, _day.month, _day.day + days);
    reload();
  }

  Future<void> _choose(Slot slot) async {
    final t = context.t;
    final sure = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(t.rescheduleConfirmTitle),
        content: Text(t.rescheduleConfirmBody(_zone.date(slot.start, t.localeName), _zone.time(slot.start))),
        actions: [
          TextButton(onPressed: () => Navigator.of(context).pop(false), child: Text(t.keepIt)),
          FilledButton(onPressed: () => Navigator.of(context).pop(true), child: Text(t.moveAppointment)),
        ],
      ),
    );
    if (sure != true || !mounted) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await widget.booking.reschedule(widget.appointment.id, slot.id);
      if (mounted) Navigator.of(context).pop(true);
    } on ApiFailure catch (failure) {
      if (!mounted) return;
      setState(() => _error = changeMessage(context.t, failure));
      if (failure.code == 'slot_unavailable') await reload();
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final last = DateTime.utc(_today.year, _today.month, _today.day + _days - 1);
    final t = context.t;
    final a = widget.appointment;
    return ui.Page(
      title: t.rescheduleTitle,
      children: [
        Text(t.rescheduleCurrent(_zone.date(a.start, t.localeName), _zone.time(a.start))),
        if (_error != null) ui.Message(_error!, error: true),
        if (_scheduleRead && _schedule == null)
          Text(t.noOpenTimes)
        else ...[
          Text(t.timesIn(_zone.name)),
          Row(
            children: [
              TextButton(onPressed: _day.isAfter(_today) ? () => _move(-1) : null, child: Text(t.previousDay)),
              const Spacer(),
              TextButton(onPressed: _day.isBefore(last) ? () => _move(1) : null, child: Text(t.nextDay)),
            ],
          ),
          ui.Heading(Zone.longDate(_day, t.localeName)),
          ...body(
            (slots) => [
              if (slots == null || slots.isEmpty) Text(t.noFreeTimes),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  for (final slot in slots ?? const <Slot>[])
                    OutlinedButton(onPressed: _busy ? null : () => _choose(slot), child: Text(_zone.time(slot.start))),
                ],
              ),
            ],
          ),
        ],
      ],
    );
  }
}
