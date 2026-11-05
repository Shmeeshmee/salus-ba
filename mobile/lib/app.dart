import 'package:flutter/material.dart';
import 'package:salus_api/api.dart';

import 'booking.dart';
import 'call.dart';
import 'home_screen.dart';
import 'messages.dart';
import 'privacy_notice.dart';
import 'register_screen.dart';
import 'session.dart';
import 'sign_in_screen.dart';
import 'ui.dart' as ui;

enum _Screen { loading, startFailed, signIn, register, home }

/// The patient app: restores the stored session, then shows sign-in, registration or the signed-in screens.
class SalusApp extends StatefulWidget {
  const SalusApp({
    super.key,
    required this.session,
    required this.booking,
    required this.now,
    required this.media,
    required this.privacyNoticeUrl,
    required this.openUrl,
  });

  final Session session;
  final Booking booking;

  /// The clock, injected so tests fix it.
  final DateTime Function() now;

  /// The camera, the microphone and LiveKit, for the video consultation.
  final Media media;

  /// The privacy notice (PRIVACY_NOTICE_URL), and how a link is opened outside the app; injected so tests see it.
  final Uri privacyNoticeUrl;
  final Future<bool> Function(Uri url) openUrl;

  @override
  State<SalusApp> createState() => _SalusAppState();
}

class _SalusAppState extends State<SalusApp> {
  _Screen _screen = _Screen.loading;
  Account? _account;
  String? _message;

  @override
  void initState() {
    super.initState();
    _restore();
  }

  Future<void> _restore() async {
    setState(() => _screen = _Screen.loading);
    try {
      final account = await widget.session.restore();
      if (!mounted) return;
      if (account == null) {
        _show(_Screen.signIn);
      } else {
        _signedIn(account);
      }
    } on ApiFailure catch (failure) {
      _show(_Screen.startFailed, failureMessage(failure));
    }
  }

  void _show(_Screen screen, [String? message]) {
    if (!mounted) return;
    setState(() {
      _screen = screen;
      _message = message;
      _account = null;
    });
  }

  void _signedIn(Account account) => setState(() {
    _screen = _Screen.home;
    _message = null;
    _account = account;
  });

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Salus',
      theme: ui.salusTheme,
      builder: (context, child) => PrivacyNotice(url: widget.privacyNoticeUrl, open: widget.openUrl, child: child!),
      home: _current(),
    );
  }

  Widget _current() {
    final session = widget.session;
    return switch (_screen) {
      _Screen.loading => const Scaffold(body: Center(child: CircularProgressIndicator())),
      _Screen.startFailed => ui.Page(
        title: 'Salus',
        children: [
          ui.Message(_message ?? somethingWrong, error: true),
          ui.SubmitButton('Try again', busy: false, onPressed: _restore),
        ],
      ),
      _Screen.signIn => SignInScreen(
        session: session,
        notice: _message,
        onSignedIn: _signedIn,
        onRegister: () => _show(_Screen.register),
      ),
      _Screen.register => RegisterScreen(
        session: session,
        onSignedIn: _signedIn,
        onRegistered: () => _show(_Screen.signIn, accountReady),
        onSignIn: () => _show(_Screen.signIn),
      ),
      _Screen.home => HomeScreen(
        session: session,
        booking: widget.booking,
        now: widget.now,
        media: widget.media,
        account: _account!,
        onSignedOut: () => _show(_Screen.signIn),
      ),
    };
  }
}
