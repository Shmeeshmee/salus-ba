import 'package:flutter/material.dart';
import 'package:salus_api/api.dart';

import 'messages.dart';
import 'session.dart';
import 'ui.dart' as ui;

/// The signed-in patient: their account e-mail and sign-out.
class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key, required this.session, required this.account, required this.onSignedOut});

  final Session session;
  final Account account;
  final VoidCallback onSignedOut;

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  String? _error;
  bool _busy = false;

  Future<void> _signOut() async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await widget.session.signOut();
      if (mounted) widget.onSignedOut();
    } on AuthFailure catch (failure) {
      if (mounted) {
        setState(() {
          _busy = false;
          _error = failure.unreachable ? signOutFailed : somethingWrong;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return ui.Page(
      title: 'Salus',
      children: [
        Text('Signed in as ${widget.account.email}'),
        ui.SubmitButton('Sign out', busy: _busy, onPressed: _signOut),
        if (_error != null) ui.Message(_error!, error: true),
      ],
    );
  }
}
