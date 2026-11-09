import 'package:flutter/material.dart';

import 'messages.dart';

// The colours of the web app (SPEC §2 "Web").
const _page = Color(0xFFF6F7F5);
const _ink = Color(0xFF0E1512);
const _brand = Color(0xFF0A6E4F);
const _line = Color(0xFFDDE3DF);

final salusTheme = ThemeData(
  colorScheme: ColorScheme.fromSeed(seedColor: _brand, primary: _brand, surface: Colors.white, onSurface: _ink),
  scaffoldBackgroundColor: _page,
  inputDecorationTheme: const InputDecorationTheme(border: OutlineInputBorder(), filled: true, fillColor: Colors.white),
  // Flat white cards on the page colour, as in the web app.
  cardTheme: const CardThemeData(
    color: Colors.white,
    elevation: 0,
    margin: EdgeInsets.zero,
    shape: RoundedRectangleBorder(
      side: BorderSide(color: _line),
      borderRadius: BorderRadius.all(Radius.circular(12)),
    ),
  ),
);

/// One screen: a title over a narrow column that scrolls when the keyboard is up. A screen whose one sentence says
/// it all has no title.
class Page extends StatelessWidget {
  const Page({super.key, required this.title, required this.children});

  final String? title;
  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(24),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 420),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  if (Navigator.of(context).canPop())
                    const Align(alignment: AlignmentDirectional.centerStart, child: BackButton()),
                  if (title != null) ...[
                    Text(title!, style: Theme.of(context).textTheme.headlineSmall),
                    const SizedBox(height: 24),
                  ],
                  for (final child in children) ...[child, const SizedBox(height: 16)],
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// A fixed sentence about the last action; errors in the theme's error colour. Read out when it changes.
class Message extends StatelessWidget {
  const Message(this.text, {super.key, this.error = false});

  final String text;
  final bool error;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    return Semantics(
      liveRegion: true,
      child: Text(text, style: TextStyle(color: error ? colors.error : colors.onSurface)),
    );
  }
}

/// The screen's main button; disabled while its request runs.
class SubmitButton extends StatelessWidget {
  const SubmitButton(this.label, {super.key, required this.busy, required this.onPressed});

  final String label;
  final bool busy;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) => FilledButton(onPressed: busy ? null : onPressed, child: Text(label));
}

/// A section heading inside a page.
class Heading extends StatelessWidget {
  const Heading(this.text, {super.key});

  final String text;

  @override
  Widget build(BuildContext context) =>
      Semantics(header: true, child: Text(text, style: Theme.of(context).textTheme.titleMedium));
}

/// While a page's data loads.
class Loading extends StatelessWidget {
  const Loading({super.key});

  @override
  Widget build(BuildContext context) => const Center(child: CircularProgressIndicator());
}

/// A load that failed: the fixed sentence and a way to try again.
class LoadFailed extends StatelessWidget {
  const LoadFailed(this.message, {super.key, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) => Column(
    crossAxisAlignment: CrossAxisAlignment.stretch,
    children: [
      Message(message, error: true),
      const SizedBox(height: 16),
      SubmitButton(context.t.tryAgain, busy: false, onPressed: onRetry),
    ],
  );
}

/// A card that opens something: a title and an optional second line.
class Choice extends StatelessWidget {
  const Choice(this.title, {super.key, this.subtitle, required this.onTap});

  final String title;
  final String? subtitle;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) => Card(
    child: ListTile(
      title: Text(title),
      subtitle: subtitle == null ? null : Text(subtitle!),
      trailing: const Icon(Icons.chevron_right),
      onTap: onTap,
    ),
  );
}
