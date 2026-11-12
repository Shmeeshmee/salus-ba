import 'package:flutter/material.dart';
import 'package:salus_api/api.dart';

import 'appointments_screen.dart';
import 'booking.dart';
import 'call.dart';
import 'files.dart';
import 'history_screen.dart';
import 'clinics_screen.dart';
import 'consents_screen.dart';
import 'language_picker.dart';
import 'messages.dart';
import 'profile_screen.dart';
import 'session.dart';
import 'ui.dart' as ui;

/// The signed-in patient: the profile first when there is none, then booking, appointments, their history,
/// consents and sign-out.
class HomeScreen extends StatefulWidget {
  const HomeScreen({
    super.key,
    required this.session,
    required this.booking,
    required this.now,
    required this.media,
    required this.files,
    required this.account,
    required this.onSignedOut,
  });

  final Session session;
  final Booking booking;
  final DateTime Function() now;
  final Media media;
  final Files files;
  final Account account;
  final VoidCallback onSignedOut;

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  bool _loading = true;
  String? _loadError;
  Patient? _profile;
  bool _later = false;
  String? _error;
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _loadError = null;
    });
    try {
      final profile = await widget.booking.me();
      if (mounted) setState(() => _profile = profile);
    } on ApiFailure catch (failure) {
      if (mounted) setState(() => _loadError = bookingMessage(context.t, failure));
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _signOut() async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await widget.session.signOut();
      if (mounted) widget.onSignedOut();
    } on ApiFailure catch (failure) {
      if (mounted) {
        setState(() {
          _busy = false;
          _error = failure.unreachable ? context.t.signOutFailed : context.t.somethingWrong;
        });
      }
    }
  }

  /// Opens [screen]; a booking there may have completed the profile, so a missing one is read again after.
  Future<void> _open(Widget Function() screen) async {
    await Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => screen()));
    if (_profile != null) return;
    try {
      final profile = await widget.booking.me();
      if (mounted) setState(() => _profile = profile);
    } on ApiFailure {
      // Keeps the reminder; the next booking asks for the profile again if it is still missing.
    }
  }

  Future<void> _editProfile() async {
    final saved = await Navigator.of(context).push(
      MaterialPageRoute<Patient>(
        builder: (context) => ProfileScreen(
          booking: widget.booking,
          now: widget.now,
          profile: _profile,
          onSaved: (p) => Navigator.of(context).pop(p),
        ),
      ),
    );
    if (saved != null && mounted) setState(() => _profile = saved);
  }

  @override
  Widget build(BuildContext context) {
    if (_loading) return const Scaffold(body: ui.Loading());
    if (_loadError != null) {
      return ui.Page(
        title: 'Salus',
        children: [ui.LoadFailed(_loadError!, onRetry: _load)],
      );
    }
    if (_profile == null && !_later) {
      return ProfileScreen(
        booking: widget.booking,
        now: widget.now,
        onSaved: (p) => setState(() => _profile = p),
        onLater: () => setState(() => _later = true),
      );
    }
    final booking = widget.booking;
    final now = widget.now;
    final t = context.t;
    return ui.Page(
      title: 'Salus',
      children: [
        const LanguageButton(),
        Text(t.signedInAs(widget.account.email)),
        if (_profile == null) ...[
          ui.Message(t.completeProfileFirst),
          OutlinedButton(onPressed: _editProfile, child: Text(t.completeProfile)),
        ],
        FilledButton(
          onPressed: () =>
              _open(() => ClinicsScreen(booking: booking, now: now, media: widget.media, files: widget.files)),
          child: Text(t.bookAppointment),
        ),
        OutlinedButton(
          onPressed: () =>
              _open(() => AppointmentsScreen(booking: booking, now: now, media: widget.media, files: widget.files)),
          child: Text(t.myAppointments),
        ),
        OutlinedButton(
          onPressed: () =>
              _open(() => HistoryScreen(booking: booking, media: widget.media, files: widget.files, now: now)),
          child: Text(t.myHistory),
        ),
        OutlinedButton(
          onPressed: () => _open(() => ConsentsScreen(booking: booking)),
          child: Text(t.myConsents),
        ),
        if (_profile != null) OutlinedButton(onPressed: _editProfile, child: Text(t.myProfile)),
        TextButton(onPressed: _busy ? null : _signOut, child: Text(t.signOut)),
        if (_error != null) ui.Message(_error!, error: true),
      ],
    );
  }
}
