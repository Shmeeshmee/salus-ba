import 'package:flutter_test/flutter_test.dart';
import 'package:salus/explain.dart';
import 'package:salus_api/api.dart' as api;

// SAL-13: a text cut at its term spans, UTF-16 offsets as Dart indexes strings; the text itself is never changed.
void main() {
  test('cuts a text at its spans, leaving the text as it is', () {
    const text = '😀 jetra i čir.';
    final got = pieces(text, [
      api.TermSpan(termId: 't-ulcer', start: 11, end: 14),
      api.TermSpan(termId: 't-liver', start: 3, end: 8),
    ]);
    expect(got, [
      (text: '😀 ', termId: null),
      (text: 'jetra', termId: 't-liver'),
      (text: ' i ', termId: null),
      (text: 'čir', termId: 't-ulcer'),
      (text: '.', termId: null),
    ]);
    expect(got.map((p) => p.text).join(), text);
  });

  test('drops a span outside the text or overlapping the one before', () {
    expect(pieces('jetra', [api.TermSpan(termId: 'a', start: 0, end: 9)]), [(text: 'jetra', termId: null)]);
    expect(
      pieces('jetra bol', [api.TermSpan(termId: 'a', start: 0, end: 5), api.TermSpan(termId: 'b', start: 3, end: 9)]),
      [(text: 'jetra', termId: 'a'), (text: ' bol', termId: null)],
    );
  });
}
