import 'package:flutter/material.dart';
import 'package:salus_api/api.dart';

import 'language_picker.dart';
import 'messages.dart';
import 'session.dart';
import 'ui.dart' as ui;

/// A new patient account, then straight into it.
class RegisterScreen extends StatefulWidget {
  const RegisterScreen({
    super.key,
    required this.session,
    required this.onSignedIn,
    required this.onRegistered,
    required this.onSignIn,
  });

  final Session session;
  final ValueChanged<Account> onSignedIn;

  /// The account exists but signing in to it did not finish here.
  final VoidCallback onRegistered;
  final VoidCallback onSignIn;

  @override
  State<RegisterScreen> createState() => _RegisterScreenState();
}

class _RegisterScreenState extends State<RegisterScreen> {
  final _form = GlobalKey<FormState>();
  final _email = TextEditingController();
  final _password = TextEditingController();
  String? _error;
  bool _busy = false;

  @override
  void dispose() {
    _email.dispose();
    _password.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_form.currentState!.validate()) return;
    final email = _email.text.trim();
    final password = _password.text;
    final choice = LanguageChoice.of(context);
    final (language, detected) = (choice.chosen, choice.detected);
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await widget.session.register(email, password, language: language, detected: detected);
    } on ApiFailure catch (failure) {
      if (mounted) {
        setState(() {
          _busy = false;
          _error = failureMessage(context.t, failure);
        });
      }
      return;
    }
    SignIn? result;
    try {
      result = await widget.session.login(email, password, language: language, detected: detected);
    } on ApiFailure {
      result = null;
    }
    if (!mounted) return;
    if (result case SignedIn(:final account)) {
      widget.onSignedIn(account);
    } else {
      widget.onRegistered();
    }
  }

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    return ui.Page(
      title: t.createAnAccount,
      children: [
        const LanguageButton(),
        Form(
          key: _form,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              TextFormField(
                controller: _email,
                decoration: InputDecoration(labelText: t.email),
                keyboardType: TextInputType.emailAddress,
                textDirection: TextDirection.ltr,
                autofillHints: const [AutofillHints.email],
                autocorrect: false,
                validator: (v) => validateEmail(t, v),
              ),
              const SizedBox(height: 16),
              TextFormField(
                controller: _password,
                decoration: InputDecoration(labelText: t.password, helperText: t.passwordHint),
                obscureText: true,
                autofillHints: const [AutofillHints.newPassword],
                validator: (v) => validateNewPassword(t, v),
                onFieldSubmitted: (_) => _submit(),
              ),
              const SizedBox(height: 24),
              ui.SubmitButton(t.createAccount, busy: _busy, onPressed: _submit),
              const SizedBox(height: 8),
              TextButton(onPressed: _busy ? null : widget.onSignIn, child: Text(t.haveAccount)),
            ],
          ),
        ),
        if (_error != null) ui.Message(_error!, error: true),
      ],
    );
  }
}
