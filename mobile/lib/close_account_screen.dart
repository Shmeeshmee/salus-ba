import 'package:flutter/material.dart';

import 'data_requests.dart';
import 'messages.dart';
import 'session.dart';
import 'ui.dart' as ui;

/// Closing the patient's account (docs/clinic-cms/WAVE5.md S5, S6): what happens at once, what is erased after 21
/// days and what stays, confirmed with the password. Once closed, [onClosed] ends the session on the device.
class CloseAccountScreen extends StatefulWidget {
  const CloseAccountScreen({super.key, required this.privacy, required this.onClosed});

  final DataRequests privacy;
  final VoidCallback onClosed;

  @override
  State<CloseAccountScreen> createState() => _CloseAccountScreenState();
}

class _CloseAccountScreenState extends State<CloseAccountScreen> {
  final _form = GlobalKey<FormState>();
  final _password = TextEditingController();
  bool _busy = false;
  bool _closed = false;
  bool _leaving = false;
  String? _error;

  @override
  void dispose() {
    _password.dispose();
    super.dispose();
  }

  Future<void> _close() async {
    if (!_form.currentState!.validate()) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await widget.privacy.closeAccount(_password.text);
      _password.clear();
      if (mounted) setState(() => _closed = true);
    } on ApiFailure catch (failure) {
      if (mounted) {
        setState(() {
          _busy = false;
          _error = failureMessage(context.t, failure);
        });
      }
    }
  }

  /// Leaves for sign-in once, however often "Done" or back is pressed.
  void _leave() {
    if (_leaving) return;
    setState(() => _leaving = true);
    widget.onClosed();
  }

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    if (_closed) {
      // The session is gone: leaving this screen goes to sign-in, not back into the app.
      return PopScope(
        canPop: false,
        onPopInvokedWithResult: (didPop, _) {
          if (!didPop) _leave();
        },
        child: ui.Page(
          title: t.closeMyAccount,
          children: [
            ui.Message(t.accountClosed),
            ui.SubmitButton(t.done, busy: _leaving, onPressed: _leave),
          ],
        ),
      );
    }
    return ui.Page(
      title: t.closeMyAccount,
      children: [
        Text(t.closeAccountNow),
        Text(t.closeAccountErased),
        Text(t.closeAccountKept),
        Text(t.closeAccountCopyFirst),
        Form(
          key: _form,
          child: TextFormField(
            controller: _password,
            obscureText: true,
            autocorrect: false,
            enableSuggestions: false,
            autofillHints: const [AutofillHints.password],
            decoration: InputDecoration(labelText: t.password, helperText: t.closeAccountConfirm),
            validator: (v) => validatePassword(t, v),
          ),
        ),
        if (_error != null) ui.Message(_error!, error: true),
        ui.SubmitButton(t.closeMyAccount, busy: _busy, onPressed: _close),
      ],
    );
  }
}
