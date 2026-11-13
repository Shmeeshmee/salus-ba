import 'package:flutter/material.dart';

import 'appointments_screen.dart';
import 'booking.dart';
import 'call.dart';
import 'consent_screen.dart';
import 'fhir.dart';
import 'files.dart';
import 'loading.dart';
import 'messages.dart';
import 'profile_screen.dart';
import 'session.dart';
import 'sharing.dart';
import 'ui.dart' as ui;
import 'zones.dart';

/// How far ahead slots exist: the scheduler generates 28 days (docs/booking/SPEC.md "Slot generation").
const _days = 28;

/// A time that starts sooner than this is not offered (owner, 2026-11-07): the patient could not book and get
/// ready for it.
const _notBefore = Duration(minutes: 5);

/// A practitioner's free times on one day at a time, in the clinic's time zone.
class SlotsScreen extends StatefulWidget {
  const SlotsScreen({
    super.key,
    required this.booking,
    required this.now,
    required this.media,
    required this.files,
    required this.clinic,
    required this.role,
  });

  final Booking booking;
  final DateTime Function() now;
  final Media media;
  final Files files;
  final Organization clinic;
  final PractitionerRole role;

  @override
  State<SlotsScreen> createState() => _SlotsScreenState();
}

class _SlotsScreenState extends LoadingState<SlotsScreen, List<Slot>?> {
  late final Zone _zone = Zone(widget.clinic.timeZone);
  late final DateTime _today = _zone.day(widget.now());
  late DateTime _day = _today;
  bool _scheduleRead = false;
  String? _schedule;

  /// The day's free slots, or null when the practitioner has no schedule.
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
    await Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => BookScreen(
          booking: widget.booking,
          now: widget.now,
          media: widget.media,
          files: widget.files,
          clinic: widget.clinic,
          role: widget.role,
          slot: slot,
        ),
      ),
    );
    if (mounted) reload();
  }

  @override
  Widget build(BuildContext context) {
    final last = DateTime.utc(_today.year, _today.month, _today.day + _days - 1);
    final t = context.t;
    return ui.Page(
      title: widget.role.name,
      children: [
        Text(specialtyAtClinic(t, specialtyName(t, widget.role), widget.clinic.name)),
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
                    OutlinedButton(onPressed: () => _choose(slot), child: Text(_zone.time(slot.start))),
                ],
              ),
            ],
          ),
        ],
      ],
    );
  }
}

/// One free time, booked on confirmation: the profile and the clinic's consent are asked for when the API says
/// they are missing, then the booking is tried again.
class BookScreen extends StatefulWidget {
  const BookScreen({
    super.key,
    required this.booking,
    required this.now,
    required this.media,
    required this.files,
    required this.clinic,
    required this.role,
    required this.slot,
  });

  final Booking booking;
  final DateTime Function() now;
  final Media media;
  final Files files;
  final Organization clinic;
  final PractitionerRole role;
  final Slot slot;

  @override
  State<BookScreen> createState() => _BookScreenState();
}

class _BookScreenState extends State<BookScreen> {
  late final Zone _zone = Zone(widget.clinic.timeZone);
  String? _error;
  bool _taken = false;
  bool _busy = false;

  /// Sharing the history from other clinics (SAL-42): off unless the patient shares with the clinic already (the
  /// preview, read when the screen opens, says so), then on with what is shared ticked.
  bool _share = false;
  SharingChoice? _choice;
  Set<String> _selected = {};

  @override
  void initState() {
    super.initState();
    _loadSharing();
  }

  /// Reads what is shared with the clinic now; nothing changes on the screen when it fails or nothing is.
  Future<void> _loadSharing() async {
    try {
      final choice = await sharingChoice(widget.booking, widget.clinic.id);
      if (!mounted || _choice != null) return;
      setState(() {
        _choice = choice;
        _selected = {
          for (final c in choice.candidates)
            if (c.selected) c.encounterId,
        };
        _share = choice.version != null;
      });
    } on ApiFailure {
      // The toggle loads the preview again when turned on.
    }
  }

  Future<void> _toggleSharing(bool on) async {
    setState(() {
      _share = on;
      _error = null;
    });
    if (!on || _choice != null) return;
    try {
      final choice = await sharingChoice(widget.booking, widget.clinic.id);
      if (!mounted) return;
      setState(() {
        _choice = choice;
        _selected = {
          for (final c in choice.candidates)
            if (c.selected) c.encounterId,
        };
      });
    } on ApiFailure catch (failure) {
      if (mounted) setState(() => _error = bookingMessage(context.t, failure));
    }
  }

  /// Shares the visits ticked once the appointment is booked (the care consent is there then), and stops a sharing
  /// turned off; false when it failed.
  Future<bool> _shareTicked() async {
    final choice = _choice;
    final selected = _share ? _selected : const <String>{};
    if (choice == null || (selected.isEmpty && choice.version == null)) return true;
    try {
      await widget.booking.setSharing(widget.clinic.id, [
        for (final c in choice.candidates)
          if (selected.contains(c.encounterId)) c.encounterId,
      ], choice.version);
      return true;
    } on ApiFailure {
      return false;
    }
  }

  Future<void> _book() async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      if (await _bookWithSteps() && mounted) _booked(await _shareTicked());
    } on ApiFailure catch (failure) {
      if (!mounted) return;
      setState(() {
        _error = bookingMessage(context.t, failure);
        _taken = failure.code == 'slot_unavailable';
      });
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  /// Books, completing the profile or consenting first when the API asks; false when the patient stopped.
  Future<bool> _bookWithSteps() async {
    while (true) {
      try {
        await widget.booking.book(widget.slot.id);
        return true;
      } on ApiFailure catch (failure) {
        final step = switch (failure.code) {
          'profile_required' => _completeProfile,
          'consent_required' => _consent,
          _ => null,
        };
        if (step == null) rethrow;
        if (!await step()) return false;
      }
    }
  }

  Future<bool> _completeProfile() async {
    final saved = await Navigator.of(context).push(
      MaterialPageRoute<Object>(
        builder: (context) =>
            ProfileScreen(booking: widget.booking, now: widget.now, onSaved: (p) => Navigator.of(context).pop(p)),
      ),
    );
    return saved != null;
  }

  Future<bool> _consent() async {
    final agreed = await Navigator.of(context)
        .push(MaterialPageRoute<bool>(builder: (_) => ConsentScreen(clinicName: widget.clinic.name)));
    if (agreed != true) {
      if (mounted) setState(() => _error = context.t.consentNeeded);
      return false;
    }
    try {
      await widget.booking.grant(widget.clinic.id);
    } on ApiFailure catch (failure) {
      if (failure.code != 'consent_active') rethrow; // granted in the meantime: book
    }
    return true;
  }

  void _booked(bool shared) {
    if (!mounted) return;
    final navigator = Navigator.of(context);
    final booked = shared ? context.t.appointmentBooked : context.t.bookedNotShared;
    navigator.popUntil((route) => route.isFirst);
    navigator.push(
      MaterialPageRoute<void>(
        builder: (_) => AppointmentsScreen(
          booking: widget.booking,
          now: widget.now,
          media: widget.media,
          files: widget.files,
          notice: booked,
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final slot = widget.slot;
    final t = context.t;
    final specialty = specialtyName(t, widget.role);
    return ui.Page(
      title: t.bookThisTime,
      children: [
        ui.Heading(_zone.date(slot.start, t.localeName)),
        Text(t.timeRange(_zone.time(slot.start), _zone.time(slot.end), _zone.name)),
        Text(widget.role.name),
        if (specialty != null) Text(specialty),
        Text(widget.clinic.name),
        SwitchListTile(value: _share, onChanged: _busy ? null : _toggleSharing, title: Text(t.sharingToggle)),
        if (_share && _choice != null)
          SharingPicker(
            clinicName: widget.clinic.name,
            candidates: _choice!.candidates,
            selected: _selected,
            onChanged: (s) => setState(() => _selected = s),
          ),
        if (_taken)
          TextButton(onPressed: () => Navigator.of(context).pop(), child: Text(t.chooseAnotherTime))
        else
          ui.SubmitButton(t.book, busy: _busy, onPressed: _book),
        if (_error != null) ui.Message(_error!, error: true),
      ],
    );
  }
}
