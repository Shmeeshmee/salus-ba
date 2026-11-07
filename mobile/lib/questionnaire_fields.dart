import 'package:flutter/material.dart';

import 'l10n/app_localizations.dart';
import 'messages.dart';
import 'questionnaire.dart';
import 'questionnaire_files.dart';
import 'ui.dart' as ui;
import 'zones.dart';

// The widgets of the questionnaire screen: an item's text with its help behind an icon, each type's input
// (native Flutter widgets, the date picker included), and the submitted answers read only.

/// An item's text, "Required" for a required question, and the help icon that shows its help text.
class ItemHeader extends StatelessWidget {
  const ItemHeader({super.key, required this.item, required this.helpOpen, required this.onHelp});

  final Item item;
  final bool helpOpen;
  final VoidCallback onHelp;

  @override
  Widget build(BuildContext context) {
    final text = item.type == ItemType.group ? ui.Heading(item.text) : Text(item.text);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Row(
          children: [
            Expanded(child: text),
            if (item.help != null)
              IconButton(
                icon: const Icon(Icons.info_outline),
                tooltip: context.t.moreInformation,
                isSelected: helpOpen,
                onPressed: onHelp,
              ),
          ],
        ),
        if (item.required && item.isQuestion)
          Text(context.t.requiredQuestion, style: Theme.of(context).textTheme.bodySmall),
        if (helpOpen && item.help != null) Text(item.help!, style: Theme.of(context).textTheme.bodySmall),
      ],
    );
  }
}

/// One question as the patient answers it. Typed answers go to [onTyped] as typed; a choice, yes or no, or a date
/// goes to [onChosen] as the new values.
class QuestionField extends StatelessWidget {
  const QuestionField({
    super.key,
    required this.item,
    required this.values,
    required this.header,
    required this.enabled,
    required this.today,
    required this.onChosen,
    required this.onTyped,
    this.error,
    this.controller,
    this.focusNode,
  });

  final Item item;
  final List<Object> values;
  final Widget header;
  final bool enabled;

  /// The date the picker opens on when there is no answer yet.
  final DateTime today;
  final ValueChanged<List<Object>> onChosen;
  final ValueChanged<String> onTyped;
  final String? error;
  final TextEditingController? controller;
  final FocusNode? focusNode;

  @override
  Widget build(BuildContext context) {
    final typed = controller != null;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        header,
        if (item.type == ItemType.choice) Text(item.repeats ? context.t.selectAll : context.t.selectOne),
        _input(context),
        if (!typed && error != null) Text(error!, style: TextStyle(color: Theme.of(context).colorScheme.error)),
      ],
    );
  }

  Widget _input(BuildContext context) => switch (item.type) {
    ItemType.boolean => RadioGroup<bool>(
      groupValue: values.firstOrNull as bool?,
      onChanged: (v) => onChosen([?v]),
      child: Column(
        children: [
          RadioListTile<bool>(value: true, title: Text(context.t.yes), enabled: enabled),
          RadioListTile<bool>(value: false, title: Text(context.t.no), enabled: enabled),
        ],
      ),
    ),
    ItemType.choice when !item.repeats => RadioGroup<String>(
      groupValue: values.firstOrNull as String?,
      onChanged: (v) => onChosen([?v]),
      child: Column(
        children: [for (final o in item.options) RadioListTile<String>(value: o, title: Text(o), enabled: enabled)],
      ),
    ),
    ItemType.choice => Column(
      children: [
        for (final o in item.options)
          CheckboxListTile(
            value: values.contains(o),
            title: Text(o),
            controlAffinity: ListTileControlAffinity.leading,
            onChanged: enabled ? (on) => onChosen(_toggled(o, on == true)) : null,
          ),
      ],
    ),
    // The button says the answer or "Choose a date"; the question goes with it for TalkBack and VoiceOver. A typed
    // answer's field and a choice's options get it from the header already.
    ItemType.date => MergeSemantics(
      child: Semantics(
        label: item.text,
        child: OutlinedButton(
          onPressed: enabled ? () => _pick(context) : null,
          child: Text(
            values.isEmpty ? context.t.chooseDate : Zone.longDate(values.first as DateTime, context.t.localeName),
          ),
        ),
      ),
    ),
    _ => TextField(
      controller: controller,
      focusNode: focusNode,
      enabled: enabled,
      onChanged: onTyped,
      keyboardType: switch (item.type) {
        ItemType.integer => const TextInputType.numberWithOptions(signed: true),
        ItemType.decimal => const TextInputType.numberWithOptions(signed: true, decimal: true),
        ItemType.text => TextInputType.multiline,
        _ => TextInputType.text,
      },
      minLines: item.type == ItemType.text ? 3 : 1,
      maxLines: item.type == ItemType.text ? null : 1,
      decoration: InputDecoration(hintText: context.t.yourAnswer, errorText: error, errorMaxLines: 3),
    ),
  };

  /// The options chosen, in the item's order, with [option] on or off.
  List<Object> _toggled(String option, bool on) => [
    for (final o in item.options)
      if (o == option ? on : values.contains(o)) o,
  ];

  Future<void> _pick(BuildContext context) async {
    final t = context.t;
    final picked = await showDatePicker(
      context: context,
      initialDate: values.firstOrNull as DateTime? ?? today,
      currentDate: today,
      firstDate: DateTime(1900),
      lastDate: DateTime(2100, 12, 31),
      // Calendar only: typed entry would ask for the US mm/dd/yyyy.
      initialEntryMode: DatePickerEntryMode.calendarOnly,
      helpText: t.chooseDate,
      cancelText: t.cancel,
      confirmText: t.done,
      errorFormatText: t.enterValidDate,
      errorInvalidText: t.enterValidDate,
    );
    if (picked != null) onChosen([DateTime.utc(picked.year, picked.month, picked.day)]);
  }
}

/// The message for a question the API named as invalid in a save (a 422): the check of its type.
String invalidAnswer(AppLocalizations t, Item item) => switch (item.type) {
  ItemType.string => t.upToCharacters(maxString),
  ItemType.text => t.upToCharacters(maxText),
  ItemType.integer => t.enterWholeNumber,
  ItemType.decimal => t.enterNumber,
  ItemType.date => t.enterValidDate,
  ItemType.choice when item.repeats => t.selectAll,
  _ => t.selectOne,
};

/// How an answer of [item] is written: in a text field, and read only.
String answerText(AppLocalizations t, Item item, List<Object> values) => switch (item.type) {
  ItemType.boolean => values.first == true ? t.yes : t.no,
  ItemType.date => Zone.longDate(values.first as DateTime, t.localeName),
  _ => values.join(', '),
};

/// A submitted response read only (M14): every shown section and question with its answer, or "Not answered",
/// one row each for the page's lazy list; a file upload answer as its files (docs/attachments/COPY.md M12).
List<Widget> answersRead(
  BuildContext context, {
  required List<Item> items,
  required Answers answers,
  required DateTime? authored,
  required Zone zone,
}) {
  final on = enabled(items, answers);
  final label = Theme.of(context).textTheme.labelLarge;
  final t = context.t;
  return [
    ui.Heading(t.yourAnswers),
    if (authored != null) Text(t.submittedAt(t.dateAndTime(zone.date(authored, t.localeName), zone.time(authored)))),
    for (final it in flatten(items))
      if (on[it.linkId]! && it.type == ItemType.group)
        ui.Heading(it.text)
      else if (on[it.linkId]! && it.isQuestion)
        Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(it.text, style: label),
            if ((answers[it.linkId] ?? const []).isEmpty)
              Text(t.notAnswered)
            else if (it.type == ItemType.attachment)
              ...filesRead(context, answers[it.linkId]!.cast<Attached>())
            else
              Text(answerText(t, it, answers[it.linkId]!)),
          ],
        ),
  ];
}
