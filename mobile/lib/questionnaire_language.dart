import 'package:flutter/material.dart';

import 'booking.dart';
import 'l10n/language.dart' show localeOf;
import 'language_picker.dart';
import 'messages.dart';
import 'questionnaire.dart';
import 'session.dart';
import 'ui.dart' as ui;

/// The app's language: the one chosen (signed in, the account's), else the device's.
String appLanguage(BuildContext context) {
  final choice = LanguageChoice.of(context);
  return choice.chosen ?? choice.detected;
}

/// The language [r] is read in: the one it was started in, else [app] when [v] has it; null for the original.
String? readIn(Version v, Response r, String app) {
  final started = r.language;
  if (started != null) return v.languages.contains(started) ? started : null;
  return v.languages.contains(app) ? app : null;
}

/// Opens a questionnaire in one of its languages (docs/i18n/SPEC.md §3 "The patient's language"): the one the
/// response was started in; else the app's language when the questionnaire has it, started at once; else the
/// patient is told which languages it has and chooses one (COPY.md M1 to M3), English preselected when it is one of
/// them, else the original. [builder] then shows the questionnaire in that language with the response as started.
class QuestionnaireLanguage extends StatefulWidget {
  const QuestionnaireLanguage({
    super.key,
    required this.booking,
    required this.version,
    required this.response,
    required this.builder,
  });

  final Booking booking;
  final Version version;
  final Response response;
  final Widget Function(Version version, Response response) builder;

  @override
  State<QuestionnaireLanguage> createState() => _QuestionnaireLanguageState();
}

class _QuestionnaireLanguageState extends State<QuestionnaireLanguage> {
  ({Version version, Response response})? _ready;

  /// The languages the patient may choose; one the API refused is taken out (M4).
  late var _offered = widget.version.languages;
  String? _chosen;
  String? _message;
  var _starting = false;

  /// A refusal that no other language changes: nothing to continue.
  var _closed = false;
  var _decided = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (_decided) return;
    _decided = true;
    final r = widget.response;
    final app = appLanguage(context);
    // Submitted, already started, or a questionnaire that names no language: nothing to choose.
    if (r.status != ResponseStatus.inProgress || r.language != null || _offered.isEmpty) {
      _ready = (version: _version(readIn(widget.version, r, app)), response: r);
    } else if (_offered.contains(app)) {
      _starting = true;
      _send(app);
    } else {
      _chosen = _preselected();
    }
  }

  Version _version(String? tag) => tag == null ? widget.version : widget.version.inLanguage(tag);

  String? _preselected() => _offered.contains('en') ? 'en' : _offered.firstOrNull;

  void _start(String tag) {
    setState(() {
      _starting = true;
      _message = null;
    });
    _send(tag);
  }

  Future<void> _send(String tag) async {
    try {
      final started = await widget.booking.start(widget.response.id, tag);
      if (!mounted) return;
      final r = widget.response;
      setState(
        () => _ready = (
          version: _version(tag),
          response: Response(
            id: r.id,
            appointmentId: r.appointmentId,
            questionnaire: r.questionnaire,
            status: r.status,
            versionId: started.versionId,
            answers: r.answers,
            language: tag,
          ),
        ),
      );
    } on ApiFailure catch (f) {
      if (!mounted) return;
      if (f.status == 409 && f.code == 'language_locked') return _reload();
      _refused(tag, f);
    }
  }

  void _refused(String tag, ApiFailure f) => setState(() {
    _starting = false;
    _message = answersMessage(context.t, f);
    _chosen = tag;
    if (f.code == 'language_unavailable') {
      _offered = [
        for (final l in _offered)
          if (l != tag) l,
      ];
      _chosen = _preselected();
    }
    _closed = _chosen == null || f.status == 404 || f.status == 409;
  });

  /// An answer or a file was saved meanwhile, so the language is set: the response as the server has it.
  Future<void> _reload() async {
    try {
      final r = await widget.booking.fhir.questionnaireResponse(widget.response.id);
      if (!mounted) return;
      setState(() => _ready = (version: _version(readIn(widget.version, r, appLanguage(context))), response: r));
    } on ApiFailure catch (f) {
      if (mounted) {
        setState(() {
          _starting = false;
          _message = answersMessage(context.t, f);
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final ready = _ready;
    if (ready != null) return widget.builder(ready.version, ready.response);
    final t = context.t;
    final app = appLanguage(context);
    final chosen = _chosen;
    return ui.Page(
      title: widget.version.title,
      children: [
        if (chosen == null && _starting) const ui.Loading(),
        if (!_offered.contains(app)) ...[
          Text(t.questionnaireNotInLanguage(languageNames[app]!)),
          Text(t.questionnaireLanguages(_offered.map((l) => languageNames[l]!).join(', '))),
        ],
        if (_offered.isNotEmpty && (chosen != null || !_starting))
          RadioGroup<String>(
            groupValue: chosen,
            onChanged: (tag) => setState(() => _chosen = tag),
            child: Column(
              children: [
                for (final tag in _offered)
                  RadioListTile<String>(
                    value: tag,
                    title: Text(languageNames[tag]!, locale: localeOf(tag)),
                    enabled: !_starting,
                  ),
              ],
            ),
          ),
        if (_message != null) ui.Message(_message!, error: true),
        if (chosen != null && !_closed)
          ui.SubmitButton(
            t.continueInLanguage(languageNames[chosen]!),
            busy: _starting,
            onPressed: () => _start(chosen),
          ),
      ],
    );
  }
}
