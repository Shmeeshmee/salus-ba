import 'dart:async';

import 'package:flutter/material.dart';

import 'health_profile.dart';
import 'loading.dart';
import 'messages.dart';
import 'session.dart';
import 'ui.dart' as ui;

// What the allergies and the medications screens share (SAL-41): the list's actions on an entry, the form's save,
// and the field that searches the terminology.

/// A list of the patient's entries: changing their own, adding a clinic's to the profile, removing one. An entry
/// changed elsewhere (412) or gone (404) is read again, and the patient told.
abstract class EntriesState<W extends StatefulWidget, E> extends LoadingState<W, List<E>> {
  String? _message;
  String? _error;

  /// The last action's outcome, above the list.
  List<Widget> get outcome => [
    if (_message != null) ui.Message(_message!),
    if (_error != null) ui.Message(_error!, error: true),
  ];

  /// Opens an entry's form; it closes with what to tell the patient, and the list is read again.
  Future<void> openForm(Widget Function() form) async {
    final said = await Navigator.of(context).push<String>(MaterialPageRoute(builder: (_) => form()));
    if (said == null || !mounted) return;
    setState(() {
      _message = said;
      _error = null;
    });
    await reload();
  }

  Future<void> remove(Future<void> Function() call) async {
    final t = context.t;
    if (await _ask(t.removeEntryTitle, t.removeEntryBody, t.remove)) await _act(call, t.entryRemoved);
  }

  Future<void> addToProfile(Future<void> Function() call) async {
    final t = context.t;
    if (await _ask(t.addToProfileTitle, t.addToProfileBody, t.addToProfile)) await _act(call, t.addedToProfile);
  }

  Future<bool> _ask(String title, String body, String yes) async {
    final t = context.t;
    final answer = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(title),
        content: Text(body),
        actions: [
          TextButton(onPressed: () => Navigator.of(context).pop(false), child: Text(t.cancel)),
          FilledButton(onPressed: () => Navigator.of(context).pop(true), child: Text(yes)),
        ],
      ),
    );
    return answer == true && mounted;
  }

  Future<void> _act(Future<void> Function() call, String done) async {
    setState(() {
      _message = null;
      _error = null;
    });
    try {
      await call();
      if (!mounted) return;
      setState(() => _message = done);
      await reload();
    } on ApiFailure catch (failure) {
      if (!mounted) return;
      final said = entryMessage(context.t, failure);
      if (failure.status != 404 && failure.status != 412) {
        setState(() => _error = said);
        return;
      }
      setState(() => _message = said);
      await reload();
    }
  }

  /// An entry's card: its [lines], whether a clinician confirmed it or a clinic recorded it, and what the patient
  /// may do with it. Only the patient's own entry is changed by them; a clinic's can be added to the profile.
  Widget entryCard({
    required String title,
    required List<String> lines,
    required bool confirmed,
    required bool fromClinic,
    required VoidCallback onChange,
    required Future<void> Function() onRemove,
    required Future<void> Function() onAddToProfile,
  }) {
    final t = context.t;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            ui.Heading(title),
            for (final line in lines) Text(line),
            if (confirmed) Text(t.confirmedByClinician),
            if (fromClinic) Text(t.recordedByClinic),
            const SizedBox(height: 8),
            fromClinic
                ? OutlinedButton(onPressed: () => addToProfile(onAddToProfile), child: Text(t.addToProfile))
                : OutlinedButton(onPressed: onChange, child: Text(t.change)),
            TextButton(onPressed: () => remove(onRemove), child: Text(t.remove)),
          ],
        ),
      ),
    );
  }
}

/// An entry's form: saved, it closes with what to tell the patient; an entry changed elsewhere or gone closes it
/// too, so the list shows the current one. Any other refusal stays on the form.
abstract class EntryFormState<W extends StatefulWidget> extends State<W> {
  final form = GlobalKey<FormState>();
  bool busy = false;
  String? error;

  Future<void> save(Future<void> Function() call) async {
    if (!form.currentState!.validate()) return;
    setState(() {
      busy = true;
      error = null;
    });
    try {
      await call();
      if (mounted) Navigator.of(context).pop(context.t.entrySaved);
    } on ApiFailure catch (failure) {
      if (!mounted) return;
      if (failure.status == 404 || failure.status == 412) {
        Navigator.of(context).pop(entryMessage(context.t, failure));
        return;
      }
      setState(() {
        busy = false;
        error = entryMessage(context.t, failure);
      });
    }
  }
}

/// An entry's name: 1 to 200 characters once trimmed, as the API checks it.
String? validateEntryName(BuildContext context, String? value) {
  final length = (value ?? '').trim().runes.length;
  return length >= 1 && length <= 200 ? null : context.t.nameRequired;
}

/// A substance or medication: typed freely, or picked from the drugs the terminology search suggests. [onPicked]
/// hears the picked term's id, and null once the text no longer is its label.
class TermField extends StatefulWidget {
  const TermField({
    super.key,
    required this.controller,
    required this.label,
    required this.search,
    required this.onPicked,
  });

  final TextEditingController controller;
  final String label;
  final Future<List<TermChoice>> Function(String q) search;
  final ValueChanged<String?> onPicked;

  @override
  State<TermField> createState() => _TermFieldState();
}

class _TermFieldState extends State<TermField> {
  static const _pause = Duration(milliseconds: 300);

  Timer? _timer;
  List<TermChoice> _choices = const [];
  String? _picked;

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  void _changed(String text) {
    if (_picked != null && text != _picked) {
      _picked = null;
      widget.onPicked(null);
    }
    _timer?.cancel();
    final q = text.trim();
    if (q.length < 2 || q.length > 100) {
      setState(() => _choices = const []);
      return;
    }
    _timer = Timer(_pause, () => _search(q));
  }

  Future<void> _search(String q) async {
    List<TermChoice> found;
    try {
      found = await widget.search(q);
    } on ApiFailure {
      found = const []; // Without suggestions the patient writes it in their own words.
    }
    if (mounted && widget.controller.text.trim() == q) setState(() => _choices = found);
  }

  void _pick(TermChoice choice) {
    widget.controller.text = choice.label;
    setState(() {
      _picked = choice.label;
      _choices = const [];
    });
    widget.onPicked(choice.id);
  }

  @override
  Widget build(BuildContext context) => Column(
    crossAxisAlignment: CrossAxisAlignment.stretch,
    children: [
      TextFormField(
        controller: widget.controller,
        decoration: InputDecoration(labelText: widget.label, helperText: context.t.termSearchHint),
        onChanged: _changed,
        validator: (v) => validateEntryName(context, v),
      ),
      for (final choice in _choices) ListTile(title: Text(choice.label), onTap: () => _pick(choice)),
    ],
  );
}
