import 'package:flutter/material.dart';

import 'l10n/language.dart';
import 'messages.dart';

/// Each language by its own name (docs/i18n/COPY.md L1), the same in every language of the app.
const languageNames = {
  'bs': 'Bosanski',
  'hr': 'Hrvatski',
  'sr-Latn': 'Srpski (latinica)',
  'sr-Cyrl': 'Српски (ћирилица)',
  'en': 'English',
  'de': 'Deutsch',
  'ar': 'العربية',
  'tr': 'Türkçe',
  'it': 'Italiano',
  'fr': 'Français',
};

/// The language the patient chose and how to change it, for every screen under the app (docs/i18n/SPEC.md §3
/// "Picking and saving").
class LanguageChoice extends InheritedWidget {
  const LanguageChoice({
    super.key,
    required this.chosen,
    required this.detected,
    required this.choose,
    required super.child,
  });

  /// One of [languages], or null while the app follows the device's languages.
  final String? chosen;

  /// The first of the device's languages the app has, sent with sign-in and registration as `detectedLanguage`.
  final String detected;

  /// Switches the app to a language, or back to the device's for null, and saves the choice.
  final void Function(String? tag) choose;

  static LanguageChoice of(BuildContext context) => context.dependOnInheritedWidgetOfExactType<LanguageChoice>()!;

  @override
  bool updateShouldNotify(LanguageChoice oldWidget) =>
      chosen != oldWidget.chosen || detected != oldWidget.detected || choose != oldWidget.choose;
}

/// The globe and the shown language's own name (L2 as its tooltip); opens the picker.
class LanguageButton extends StatelessWidget {
  const LanguageButton({super.key});

  @override
  Widget build(BuildContext context) {
    final choice = LanguageChoice.of(context);
    final shown = choice.chosen ?? choice.detected;
    return Align(
      alignment: AlignmentDirectional.centerEnd,
      child: Tooltip(
        message: context.t.language,
        child: TextButton.icon(
          onPressed: () => _pick(context, choice),
          icon: const Icon(Icons.language),
          label: Text(languageNames[shown]!, locale: localeOf(shown)),
        ),
      ),
    );
  }

  /// The device's language first (L3), then the ten by their own names; the current choice is ticked.
  Future<void> _pick(BuildContext context, LanguageChoice choice) async {
    final picked = await showDialog<(String?,)>(
      context: context,
      builder: (context) => SimpleDialog(
        title: Text(context.t.language),
        children: [
          _option(context, context.t.useDeviceLanguage, null, choice.chosen == null),
          for (final tag in languages) _option(context, languageNames[tag]!, tag, choice.chosen == tag),
        ],
      ),
    );
    if (picked != null) choice.choose(picked.$1);
  }

  Widget _option(BuildContext context, String label, String? tag, bool selected) => SimpleDialogOption(
    onPressed: () => Navigator.of(context).pop((tag,)),
    child: Row(
      children: [
        Expanded(child: Text(label, locale: tag == null ? null : localeOf(tag))),
        if (selected) const Icon(Icons.check),
      ],
    ),
  );
}
