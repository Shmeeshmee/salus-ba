import 'package:flutter/material.dart';

// The colours of the web app (SPEC §2 "Web").
const _page = Color(0xFFF6F7F5);
const _ink = Color(0xFF0E1512);
const _brand = Color(0xFF0A6E4F);

final salusTheme = ThemeData(
  colorScheme: ColorScheme.fromSeed(seedColor: _brand, primary: _brand, surface: Colors.white, onSurface: _ink),
  scaffoldBackgroundColor: _page,
  inputDecorationTheme: const InputDecorationTheme(border: OutlineInputBorder(), filled: true, fillColor: Colors.white),
);

/// One screen: a title over a narrow column that scrolls when the keyboard is up.
class Page extends StatelessWidget {
  const Page({super.key, required this.title, required this.children});

  final String title;
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
                  Text(title, style: Theme.of(context).textTheme.headlineSmall),
                  const SizedBox(height: 24),
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
