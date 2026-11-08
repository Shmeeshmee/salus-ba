import 'package:flutter/material.dart';
import 'package:salus_api/api.dart';

import 'language_picker.dart';
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
    if (!_form.currentState!.validate()) return;
    final choice = LanguageChoice.of(context);
    _run(
      () =>
          widget.session.login(_email.text.trim(), _password.text, language: choice.chosen, detected: choice.detected),
    );
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
            _notice = context.t.staffOnly;
        }
      });
    } on ApiFailure catch (failure) {
      if (!mounted) return;
      setState(() {
        if (failure.code == 'invalid_challenge') {
          _challenge = null;
          _notice = context.t.challengeEnded;
        } else {
          _error = failureMessage(context.t, failure);
        }
      });
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return ui.Page(
      title: context.t.signInTitle,
      children: [
        const LanguageButton(),
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

  List<Widget> _passwordStep() {
    final t = context.t;
    return [
      TextFormField(
        controller: _email,
        decoration: InputDecoration(labelText: t.email),
        keyboardType: TextInputType.emailAddress,
        // An address reads left to right in every language (docs/i18n/SPEC.md §3 "Right to left").
        textDirection: TextDirection.ltr,
        autofillHints: const [AutofillHints.email],
        autocorrect: false,
        validator: (v) => validateEmail(t, v),
      ),
      const SizedBox(height: 16),
      TextFormField(
        controller: _password,
        decoration: InputDecoration(labelText: t.password),
        obscureText: true,
        autofillHints: const [AutofillHints.password],
        validator: (v) => validatePassword(t, v),
        onFieldSubmitted: (_) => _submitPassword(),
      ),
      const SizedBox(height: 24),
      ui.SubmitButton(t.signIn, busy: _busy, onPressed: _submitPassword),
      const SizedBox(height: 8),
      TextButton(onPressed: _busy ? null : widget.onRegister, child: Text(t.createAnAccount)),
    ];
  }

  List<Widget> _codeStep() {
    final t = context.t;
    return [
      Text(t.codePrompt),
      const SizedBox(height: 16),
      TextFormField(
        controller: _code,
        decoration: InputDecoration(labelText: t.code),
        keyboardType: TextInputType.number,
        textDirection: TextDirection.ltr,
        autofillHints: const [AutofillHints.oneTimeCode],
        validator: (v) => validateCode(t, v),
        onFieldSubmitted: (_) => _submitCode(),
      ),
      const SizedBox(height: 24),
      ui.SubmitButton(t.continueLabel, busy: _busy, onPressed: _submitCode),
      const SizedBox(height: 8),
      TextButton(onPressed: _busy ? null : _restart, child: Text(t.useAnotherAccount)),
    ];
  }
}
