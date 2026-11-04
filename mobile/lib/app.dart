import 'package:flutter/material.dart';
import 'package:salus_api/api.dart';

import 'home_screen.dart';
import 'messages.dart';
import 'register_screen.dart';
import 'session.dart';
import 'sign_in_screen.dart';
import 'ui.dart' as ui;

enum _Screen { loading, startFailed, signIn, register, home }

/// The patient app: restores the stored session, then shows sign-in, registration or the signed-in screen.
class SalusApp extends StatefulWidget {
  const SalusApp({super.key, required this.session});

  final Session session;

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
    } on AuthFailure catch (failure) {
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
    return MaterialApp(title: 'Salus', theme: ui.salusTheme, home: _current());
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
      _Screen.home => HomeScreen(session: session, account: _account!, onSignedOut: () => _show(_Screen.signIn)),
    };
  }
}
