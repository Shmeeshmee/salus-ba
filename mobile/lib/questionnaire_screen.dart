import 'dart:async';

import 'package:flutter/material.dart';

import 'booking.dart';
import 'messages.dart';
import 'questionnaire.dart';
import 'questionnaire_fields.dart';
import 'session.dart';
import 'ui.dart' as ui;
import 'zones.dart';

/// How long typing pauses before the answers are saved (docs/questionnaire/SPEC.md §2 "Patient flow").
const typingPause = Duration(seconds: 2);

/// How long after a failed save it is tried again.
const retryAfter = Duration(seconds: 5);

enum _Saving { none, saving, saved, failed }

/// One questionnaire on one scrolling page (SPEC §2 "Mobile"): sections as headings, help behind an icon, the
/// answers saved as the patient goes, "Submit" with a confirmation, and the answers read only once submitted.
class QuestionnaireScreen extends StatefulWidget {
  const QuestionnaireScreen({
    super.key,
    required this.booking,
    required this.now,
    required this.version,
    required this.response,
    required this.clinic,
    required this.zone,
  });

  final Booking booking;
  final DateTime Function() now;
  final Version version;
  final Response response;

  /// The clinic's name (M5, M11) and time zone.
  final String clinic;
  final Zone zone;

  @override
  State<QuestionnaireScreen> createState() => _QuestionnaireScreenState();
}

class _QuestionnaireScreenState extends State<QuestionnaireScreen> {
  late Response _response = widget.response;
  late Answers _answers = {...widget.response.answers};
  final _errors = <String, String>{};
  final _controllers = <String, TextEditingController>{};
  final _focus = <String, FocusNode>{};
  final _helpOpen = <String>{};

  /// Changes made, and how many of them the server has: a save is pending while they differ.
  var _edits = 0;
  var _savedEdits = 0;
  var _saving = _Saving.none;
  Future<void>? _inFlight;
  Timer? _typing;
  Timer? _retry;

  /// The last refusal (M12, M16).
  String? _message;

  /// A refusal ended filling in: no more saves.
  var _closed = false;
  var _submitting = false;
  var _submitted = false;
  var _leaving = false;

  List<Item> get _items => widget.version.items;
  bool get _pending => _edits != _savedEdits;

  static bool _isTyped(Item it) =>
      const {ItemType.string, ItemType.text, ItemType.integer, ItemType.decimal}.contains(it.type);

  @override
  void initState() {
    super.initState();
    for (final it in flatten(_items).where(_isTyped)) {
      _controllers[it.linkId] = TextEditingController(text: _shown(it));
      _focus[it.linkId] = FocusNode()
        ..addListener(() {
          if (!_focus[it.linkId]!.hasFocus) _saveNow();
        });
    }
  }

  @override
  void dispose() {
    _typing?.cancel();
    _retry?.cancel();
    for (final c in _controllers.values) {
      c.dispose();
    }
    for (final f in _focus.values) {
      f.dispose();
    }
    super.dispose();
  }

  String _shown(Item it) => (_answers[it.linkId] ?? const []).isEmpty ? '' : answerText(it, _answers[it.linkId]!);

  void _set(String linkId, List<Object>? values) {
    if (values == null || values.isEmpty) {
      _answers.remove(linkId);
    } else {
      _answers[linkId] = values;
    }
    _edits++;
  }

  /// A typed answer: checked at once, saved once typing pauses or the field is left. One that is not valid is
  /// shown on its field and left out of the answers.
  void _typed(Item it, String text) {
    final read = readTyped(it.type, text);
    setState(() {
      _set(it.linkId, read.value == null ? null : [read.value!]);
      if (read.error == null) {
        _errors.remove(it.linkId);
      } else {
        _errors[it.linkId] = read.error!;
      }
    });
    _typing?.cancel();
    _typing = Timer(typingPause, _saveNow);
  }

  /// A choice, yes or no, or a date: complete as it is made, so saved at once.
  void _chosen(Item it, List<Object> values) {
    setState(() {
      _set(it.linkId, values);
      _errors.remove(it.linkId);
    });
    _saveNow();
  }

  /// Saves now, or after the save already running; saves again while answers changed in the meantime.
  Future<void> _saveNow() async {
    _typing?.cancel();
    _retry?.cancel();
    if (_inFlight != null) return _inFlight;
    if (!mounted || _closed || !_pending) return;
    final run = _saveWhilePending();
    _inFlight = run;
    await run;
    if (identical(_inFlight, run)) _inFlight = null;
  }

  Future<void> _saveWhilePending() async {
    while (mounted && !_closed && _pending && !(_typing?.isActive ?? false)) {
      if (!await _save()) return;
    }
  }

  Future<bool> _save() async {
    final edits = _edits;
    setState(() => _saving = _Saving.saving);
    try {
      final r = await widget.booking.saveAnswers(_response.id, _response.versionId, toApi(_items, _answers));
      if (!mounted) return false;
      setState(() {
        _response = _withVersion(r.versionId);
        _savedEdits = edits;
        _saving = _Saving.saved;
      });
      return true;
    } on ApiFailure catch (f) {
      if (mounted) _refused(f);
      return false;
    }
  }

  Response _withVersion(int versionId) => Response(
    id: _response.id,
    appointmentId: _response.appointmentId,
    questionnaire: _response.questionnaire,
    status: _response.status,
    versionId: versionId,
    answers: _response.answers,
  );

  /// A refused save: a lost connection is tried again, and anything else as [_refusedFinal] says.
  void _refused(ApiFailure f) {
    if (f.unreachable || f.status >= 500) {
      setState(() => _saving = _Saving.failed);
      _retry = Timer(retryAfter, _saveNow);
      return;
    }
    setState(() => _saving = _Saving.none);
    _refusedFinal(f, invalid: invalidAnswer);
  }

  /// A refused save or submit: a 412 reads the latest answers, a 422 is shown on the questions it names with
  /// [invalid], and anything else is said; a 404 or 409 ends filling in.
  void _refusedFinal(ApiFailure f, {required String Function(Item) invalid}) {
    if (f.status == 412) {
      _reload();
      return;
    }
    final items = {for (final it in flatten(_items)) it.linkId: it};
    setState(() {
      if (f.status == 422) {
        for (final id in f.linkIds.where(items.containsKey)) {
          _errors[id] = invalid(items[id]!);
        }
        return;
      }
      _message = answersMessage(f);
      _closed = f.status == 404 || f.status == 409;
    });
  }

  /// The answers as the server has them now, after they were changed elsewhere (M16, 412).
  Future<void> _reload() async {
    try {
      final r = await widget.booking.fhir.questionnaireResponse(_response.id);
      if (!mounted) return;
      setState(() {
        _response = r;
        _answers = {...r.answers};
        _errors.clear();
        _savedEdits = _edits;
        _saving = _Saving.none;
        _closed = r.status == ResponseStatus.stopped;
        _message = _closed ? withdrawnByClinic : changedElsewhere;
        for (final it in flatten(_items).where(_isTyped)) {
          _controllers[it.linkId]!.text = _shown(it);
        }
      });
    } on ApiFailure catch (f) {
      if (mounted) setState(() => _message = answersMessage(f));
    }
  }

  Future<void> _submit() async {
    final on = enabled(_items, _answers);
    final unanswered = missing(_items, _answers);
    if (unanswered.isNotEmpty || _errors.keys.any((id) => on[id] == true)) {
      setState(() {
        for (final id in unanswered) {
          _errors.putIfAbsent(id, () => questionRequired);
        }
        _message = unanswered.isEmpty ? null : requiredMissing;
      });
      return;
    }
    if (await _confirmSubmit() != true || !mounted) return;
    setState(() {
      _submitting = true;
      _message = null;
    });
    await _saveNow();
    if (!mounted) return;
    if (_pending || _closed) {
      setState(() => _submitting = false);
      return;
    }
    try {
      final r = await widget.booking.submitAnswers(_response.id, _response.versionId);
      if (!mounted) return;
      setState(() {
        _response = Response(
          id: r.id,
          appointmentId: r.appointmentId,
          questionnaire: _response.questionnaire,
          status: ResponseStatus.completed,
          versionId: r.versionId,
          answers: fromApi(_items, r.answers),
          authored: r.authored?.toUtc(),
        );
        _submitted = true;
      });
    } on ApiFailure catch (f) {
      if (!mounted) return;
      setState(() {
        _submitting = false;
        if (f.status == 422) _message = requiredMissing;
      });
      _refusedFinal(f, invalid: (_) => questionRequired);
    }
  }

  Future<bool?> _confirmSubmit() => showDialog<bool>(
    context: context,
    builder: (context) => AlertDialog(
      title: const Text(submitTitle),
      content: Text(submitBody(widget.clinic)),
      actions: [
        TextButton(onPressed: () => Navigator.of(context).pop(false), child: const Text(cancel)),
        FilledButton(onPressed: () => Navigator.of(context).pop(true), child: const Text(submit)),
      ],
    ),
  );

  /// Leaving while a save is pending asks first (M15).
  Future<void> _onPop(bool didPop, Object? _) async {
    if (didPop) return;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        content: const Text(leaveUnsaved),
        actions: [
          TextButton(onPressed: () => Navigator.of(context).pop(false), child: const Text(stay)),
          TextButton(onPressed: () => Navigator.of(context).pop(true), child: const Text(leave)),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    setState(() => _leaving = true);
    Navigator.of(context).pop();
  }

  @override
  Widget build(BuildContext context) {
    final done = _response.status == ResponseStatus.completed;
    return PopScope(
      canPop: _leaving || done || _closed || !_pending,
      onPopInvokedWithResult: _onPop,
      child: _page(done ? _view() : _form(), footer: done ? const [] : _footer()),
    );
  }

  /// ui.Page's look as a lazy list: only the rows on screen are built. A 200-item questionnaire's first frame
  /// took about 620 ms in a Column and 45 ms this way (docs/questionnaire/PROGRESS.md, package 10).
  Widget _page(List<Widget> rows, {required List<Widget> footer}) {
    final head = [
      if (Navigator.of(context).canPop()) const Align(alignment: Alignment.centerLeft, child: BackButton()),
      Text(widget.version.title, style: Theme.of(context).textTheme.headlineSmall),
      const SizedBox(height: 8),
    ];
    final all = [...head, ...rows];
    return Scaffold(
      bottomNavigationBar: footer.isEmpty
          ? null
          : SafeArea(
              child: Padding(
                padding: const EdgeInsets.fromLTRB(24, 8, 24, 8),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: footer,
                ),
              ),
            ),
      body: SafeArea(
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 468),
            child: ListView.builder(
              padding: const EdgeInsets.all(24),
              itemCount: all.length,
              itemBuilder: (context, i) => Padding(
                padding: EdgeInsets.only(bottom: i < head.length ? 0 : 16),
                child: all[i],
              ),
            ),
          ),
        ),
      ),
    );
  }

  List<Widget> _view() => [
    if (_submitted) ...[const ui.Heading(answersSubmitted), const ui.Message(answersSubmittedBody)],
    ...answersRead(context, items: _items, answers: _response.answers, authored: _response.authored, zone: widget.zone),
  ];

  List<Widget> _form() {
    final on = enabled(_items, _answers);
    return [
      Text(savedAsYouGo(widget.clinic)),
      for (final it in flatten(_items))
        if (on[it.linkId]! && it.type != ItemType.unsupported) _item(it),
      if (!_closed) ui.SubmitButton(submit, busy: _submitting, onPressed: _submit),
    ];
  }

  /// Under the list, so it stays in view however far the page is scrolled: the saved state (M6) and the last
  /// refusal (M12, M16).
  List<Widget> _footer() => [
    if (_message != null) ui.Message(_message!, error: true),
    if (_saving != _Saving.none)
      ui.Message(switch (_saving) {
        _Saving.saving => saving,
        _Saving.failed => notSaved,
        _ => saved,
      }, error: _saving == _Saving.failed),
  ];

  Widget _item(Item it) {
    final header = ItemHeader(
      item: it,
      helpOpen: _helpOpen.contains(it.linkId),
      onHelp: () =>
          setState(() => _helpOpen.contains(it.linkId) ? _helpOpen.remove(it.linkId) : _helpOpen.add(it.linkId)),
    );
    if (!it.isQuestion) return header;
    return QuestionField(
      key: ValueKey('answer-${it.linkId}'),
      item: it,
      values: _answers[it.linkId] ?? const [],
      header: header,
      enabled: !_closed && !_submitting,
      today: widget.zone.day(widget.now()),
      error: _errors[it.linkId],
      controller: _controllers[it.linkId],
      focusNode: _focus[it.linkId],
      onChosen: (values) => _chosen(it, values),
      onTyped: (text) => _typed(it, text),
    );
  }
}
