import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:salus_api/api.dart' as api;

import 'booking.dart';
import 'messages.dart';
import 'questionnaire_language.dart' show appLanguage;

// The plain-language explanations of the medical terms in a signed document and in the transcript copy (SAL-13,
// docs/terminology/SPEC.md §3 "Explanations"): the server tags the text where it is read (TermSpan, UTF-16 offsets,
// as Dart indexes strings), and a tap on a tagged term opens its explanation in the app's language, else English,
// labelled informational, never a diagnosis. An overlay: the text is shown as it was signed or said.

/// The explanations a screen shows, by term id.
typedef Explanations = Map<String, api.ExplainedTerm>;

/// A piece of a text: its words, and the term they name when they are a span's.
typedef Piece = ({String text, String? termId});

/// [text] cut at its [spans]: the parts between them and each span's words with its term. A span outside the text
/// or overlapping the one before is left out; joined, the pieces are the text.
List<Piece> pieces(String text, List<api.TermSpan> spans) {
  final out = <Piece>[];
  var at = 0;
  for (final s in [...spans]..sort((a, b) => a.start.compareTo(b.start))) {
    if (s.start < at || s.end > text.length || s.end <= s.start) continue;
    if (s.start > at) out.add((text: text.substring(at, s.start), termId: null));
    out.add((text: text.substring(s.start, s.end), termId: s.termId));
    at = s.end;
  }
  if (at < text.length || out.isEmpty) out.add((text: text.substring(at), termId: null));
  return out;
}

/// Loads the explanations in the app's language for a screen, again when the language changes; none until they
/// are read, and none when they cannot be (the text is then shown plain).
mixin Explaining<W extends StatefulWidget> on State<W> {
  Booking get booking;
  Explanations explanations = const {};
  String? _language;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final language = appLanguage(context);
    if (language == _language) return;
    _language = language;
    booking.explanations(language).then((got) {
      if (mounted && _language == language) setState(() => explanations = got);
    }, onError: (Object _) {});
  }
}

/// A line of text with its explained terms underlined, each opening its explanation on a tap. [prefix] and [suffix]
/// go around the text untagged (a label the line is shown in). A line without an explained term is a plain [Text].
class TermLine extends StatefulWidget {
  const TermLine(
    this.text, {
    super.key,
    this.spans = const [],
    this.explanations = const {},
    this.prefix = '',
    this.suffix = '',
  });

  final String text;
  final List<api.TermSpan> spans;
  final Explanations explanations;
  final String prefix;
  final String suffix;

  @override
  State<TermLine> createState() => _TermLineState();
}

class _TermLineState extends State<TermLine> {
  final _taps = <TapGestureRecognizer>[];

  void _forget() {
    for (final r in _taps) {
      r.dispose();
    }
    _taps.clear();
  }

  @override
  void dispose() {
    _forget();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    _forget();
    final parts = pieces(widget.text, widget.spans);
    if (!parts.any((p) => widget.explanations.containsKey(p.termId))) {
      final line = widget.prefix + widget.text + widget.suffix; // the text given, with its label
      return Text(line);
    }
    final marked = TextStyle(
      decoration: TextDecoration.underline,
      decorationStyle: TextDecorationStyle.dotted,
      color: Theme.of(context).colorScheme.primary,
    );
    return Text.rich(
      TextSpan(
        children: [
          if (widget.prefix.isNotEmpty) TextSpan(text: widget.prefix),
          for (final p in parts)
            if (widget.explanations[p.termId] case final e?)
              TextSpan(
                text: p.text,
                style: marked,
                recognizer: (TapGestureRecognizer()..onTap = () => showExplanation(context, p.text, e)).also(_taps.add),
              )
            else
              TextSpan(text: p.text),
          if (widget.suffix.isNotEmpty) TextSpan(text: widget.suffix),
        ],
      ),
    );
  }
}

extension on TapGestureRecognizer {
  TapGestureRecognizer also(void Function(TapGestureRecognizer) f) {
    f(this);
    return this;
  }
}

/// The explanation [e] of the term said as [words]: its text (left to right when it is the English fallback in a
/// right-to-left app), that it is informational and not a diagnosis, and "Not reviewed yet" until it is approved.
Future<void> showExplanation(BuildContext context, String words, api.ExplainedTerm e) => showDialog<void>(
  context: context,
  builder: (context) {
    final t = context.t;
    final fallback = e.language.toJson() != appLanguage(context);
    return AlertDialog(
      title: Text(words),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        spacing: 12,
        children: [
          if (fallback) Directionality(textDirection: TextDirection.ltr, child: Text(e.text)) else Text(e.text),
          Text(t.termExplanationNote, style: Theme.of(context).textTheme.bodySmall),
          if (e.reviewStatus != api.ExplainedTermReviewStatusEnum.approved)
            Text(t.termNotReviewed, style: Theme.of(context).textTheme.bodySmall),
        ],
      ),
      actions: [TextButton(onPressed: () => Navigator.of(context).pop(), child: Text(t.termClose))],
    );
  },
);
