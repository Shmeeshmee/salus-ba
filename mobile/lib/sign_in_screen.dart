import 'package:flutter/material.dart';
import 'package:salus_api/api.dart';

import 'messages.dart';
import 'session.dart';
import 'ui.dart' as ui;

/// E-mail and password, then the TOTP code when the account has MFA.
class SignInScreen extends StatefulWidget {
  const SignInScreen({
    super.key,
    required this.session,
    required this.onSignedIn,
    required this.onRegister,
    this.notice,
  });

  final Session session;
  final ValueChanged<Account> onSignedIn;
  final VoidCallback onRegister;
  final String? notice;

  @override
  State<SignInScreen> createState() => _SignInScreenState();
}

class _SignInScreenState extends State<SignInScreen> {
  final _form = GlobalKey<FormState>();
  final _email = TextEditingController();
  final _password = TextEditingController();
  final _code = TextEditingController();
  late String? _notice = widget.notice;
  String? _error;
  String? _challenge;
  bool _busy = false;

  @override
  void dispose() {
    _email.dispose();
    _password.dispose();
    _code.dispose();
    super.dispose();
  }

  void _submitPassword() {
    if (_form.currentState!.validate()) _run(() => widget.session.login(_email.text.trim(), _password.text));
  }

  void _submitCode() {
    final challenge = _challenge!;
    if (_form.currentState!.validate()) _run(() => widget.session.verify(challenge, _code.text.trim()));
  }

  void _restart() => setState(() {
    _challenge = null;
    _error = null;
  });

  Future<void> _run(Future<SignIn> Function() step) async {
    setState(() {
      _busy = true;
      _error = null;
      _notice = null;
    });
    try {
      final result = await step();
      if (!mounted) return;
      if (result case SignedIn(:final account)) return widget.onSignedIn(account);
      setState(() {
        switch (result) {
          case SignedIn():
            break;
          case CodeNeeded(:final challenge):
            _challenge = challenge;
            _password.clear();
            _code.clear();
          case StaffAccount():
            _challenge = null;
            _notice = staffOnly;
        }
      });
    } on AuthFailure catch (failure) {
      if (!mounted) return;
      setState(() {
        if (failure.code == 'invalid_challenge') {
          _challenge = null;
          _notice = challengeEnded;
        } else {
          _error = failureMessage(failure);
        }
      });
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return ui.Page(
      title: 'Sign in to Salus',
      children: [
        if (_notice != null) ui.Message(_notice!),
        Form(
          key: _form,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: _challenge == null ? _passwordStep() : _codeStep(),
          ),
        ),
        if (_error != null) ui.Message(_error!, error: true),
      ],
    );
  }

  List<Widget> _passwordStep() => [
    TextFormField(
      controller: _email,
      decoration: const InputDecoration(labelText: 'E-mail'),
      keyboardType: TextInputType.emailAddress,
      autofillHints: const [AutofillHints.email],
      autocorrect: false,
      validator: validateEmail,
    ),
    const SizedBox(height: 16),
    TextFormField(
      controller: _password,
      decoration: const InputDecoration(labelText: 'Password'),
      obscureText: true,
      autofillHints: const [AutofillHints.password],
      validator: validatePassword,
      onFieldSubmitted: (_) => _submitPassword(),
    ),
    const SizedBox(height: 24),
    ui.SubmitButton('Sign in', busy: _busy, onPressed: _submitPassword),
    const SizedBox(height: 8),
    TextButton(onPressed: _busy ? null : widget.onRegister, child: const Text('Create an account')),
  ];

  List<Widget> _codeStep() => [
    const Text('Enter the 6-digit code from your authenticator app.'),
    const SizedBox(height: 16),
    TextFormField(
      controller: _code,
      decoration: const InputDecoration(labelText: 'Code'),
      keyboardType: TextInputType.number,
      autofillHints: const [AutofillHints.oneTimeCode],
      validator: validateCode,
      onFieldSubmitted: (_) => _submitCode(),
    ),
    const SizedBox(height: 24),
    ui.SubmitButton('Continue', busy: _busy, onPressed: _submitCode),
    const SizedBox(height: 8),
    TextButton(onPressed: _busy ? null : _restart, child: const Text('Use another account')),
  ];
}
