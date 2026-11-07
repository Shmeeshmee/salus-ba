import 'package:flutter/material.dart';
import 'package:intl/intl.dart' show DateFormat;
import 'package:salus_api/api.dart';

import 'booking.dart';
import 'call.dart';
import 'files.dart';
import 'home_screen.dart';
import 'l10n/app_localizations.dart';
import 'l10n/language.dart';
import 'messages.dart';
import 'privacy_notice.dart';
import 'register_screen.dart';
import 'session.dart';
import 'sign_in_screen.dart';
import 'ui.dart' as ui;

enum _Screen { loading, startFailed, signIn, register, home }

/// The app's theme with the bundled Noto Sans Arabic after its own font, for Arabic letters (SPEC §3 "Fonts").
final _theme = ui.salusTheme.copyWith(
  textTheme: ui.salusTheme.textTheme.apply(fontFamilyFallback: const [arabicFont]),
  primaryTextTheme: ui.salusTheme.primaryTextTheme.apply(fontFamilyFallback: const [arabicFont]),
);

/// The patient app: restores the stored session, then shows sign-in, registration or the signed-in screens.
class SalusApp extends StatefulWidget {
  const SalusApp({
    super.key,
    required this.session,
    required this.booking,
    required this.now,
    required this.media,
    required this.files,
    required this.languages,
    required this.privacyNoticeUrl,
    required this.openUrl,
  });

  final Session session;
  final Booking booking;

  /// The clock, injected so tests fix it.
  final DateTime Function() now;

  /// The camera, the microphone and LiveKit, for the video consultation.
  final Media media;
  final Files files;

  /// The language the patient chose on this device, if any.
  final LanguageStore languages;

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

  /// The language chosen on this device, one of [languages]; null follows the device's languages.
  String? _language;

  @override
  void initState() {
    super.initState();
    // Latin digits in every language, Arabic included, so times, codes and phone numbers look the same everywhere
    // (SPEC §3 "Formatting"); intl writes Arabic-Indic digits for `ar` otherwise.
    DateFormat.useNativeDigitsByDefaultFor('ar', false);
    _readLanguage();
    _restore();
  }

  Future<void> _readLanguage() async {
    final String? saved;
    try {
      saved = await widget.languages.read();
    } on Exception {
      return; // Storage that cannot be read leaves the device's language.
    }
    if (mounted && languages.contains(saved)) setState(() => _language = saved);
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
      theme: _theme,
      // Material lays Arabic out right to left from the locale (SPEC §3 "Right to left").
      locale: _language == null ? null : localeOf(_language!),
      localeListResolutionCallback: (device, _) => localeOf(matchLanguage(device ?? const [])),
      supportedLocales: AppLocalizations.supportedLocales,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      builder: (context, child) => PrivacyNotice(url: widget.privacyNoticeUrl, open: widget.openUrl, child: child!),
      home: _current(),
    );
  }

  Widget _current() {
    final session = widget.session;
    return switch (_screen) {
      _Screen.loading => const Scaffold(body: Center(child: CircularProgressIndicator())),
      _Screen.startFailed => Builder(
        builder: (context) => ui.Page(
          title: 'Salus',
          children: [
            ui.Message(_message ?? somethingWrong, error: true),
            ui.SubmitButton(AppLocalizations.of(context).tryAgain, busy: false, onPressed: _restore),
          ],
        ),
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
        files: widget.files,
        account: _account!,
        onSignedOut: () => _show(_Screen.signIn),
      ),
    };
  }
}
