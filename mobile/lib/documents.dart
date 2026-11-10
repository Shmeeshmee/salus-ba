// The patient's signed documents (docs/consultation/SPEC.md "Patient documents"): FHIR DocumentReference and
// Composition as the app shows them, read from the elements it needs. A document is shown as text, field by field
// and line by line, so each line is a Text of its own; its PDF is the signed original.

/// A signed document of an encounter: its DocumentReference (the PDF) and the Composition it was signed from.
class SignedDocument {
  const SignedDocument({
    required this.id,
    required this.compositionId,
    required this.title,
    required this.language,
    required this.signedAt,
  });

  factory SignedDocument.fromFhir(Map<String, dynamic> r) {
    final attachment = ((r['content'] as List?)?.firstOrNull as Map?)?['attachment'] as Map? ?? const {};
    final related = (((r['context'] as Map?)?['related'] as List?)?.firstOrNull as Map?)?['reference'];
    return SignedDocument(
      id: _string(r['id']),
      compositionId: _string(related).replaceFirst('Composition/', ''),
      title: _string(attachment['title']),
      language: _string(attachment['language']),
      signedAt: DateTime.parse(_string(r['date'])).toUtc(),
    );
  }

  /// The DocumentReference's id, which names the PDF.
  final String id;
  final String compositionId;
  final String title;

  /// The BCP 47 tag of the language it was written and signed in.
  final String language;
  final DateTime signedAt;
}

/// One field of a signed document: its label and its value's lines; a field left empty has the narrative's dash.
class DocumentSection {
  const DocumentSection({required this.title, required this.lines});

  final String title;
  final List<String> lines;
}

/// A signed Composition: its title and its fields in order.
class DocumentContent {
  const DocumentContent({required this.title, required this.sections});

  factory DocumentContent.fromFhir(Map<String, dynamic> r) => DocumentContent(
    title: _string(r['title']),
    sections: [
      for (final s in (r['section'] as List?) ?? const [])
        DocumentSection(title: _string(s['title']), lines: narrativeLines(_string((s['text'] as Map?)?['div']))),
    ],
  );

  final String title;
  final List<DocumentSection> sections;
}

/// The documents in [language] first, each group in the order given (the order they were signed).
List<SignedDocument> languageFirst(List<SignedDocument> documents, String language) => [
  for (final d in documents)
    if (d.language == language) d,
  for (final d in documents)
    if (d.language != language) d,
];

final _line = RegExp(r'<(p|li)>(.*?)</\1>', dotAll: true);

/// The lines of a section's XHTML narrative as the API writes it: a paragraph or a list item per line.
List<String> narrativeLines(String div) => [for (final m in _line.allMatches(div)) _unescape(m[2]!)];

String _unescape(String s) => s
    .replaceAll('&lt;', '<')
    .replaceAll('&gt;', '>')
    .replaceAll('&#34;', '"')
    .replaceAll('&quot;', '"')
    .replaceAll('&#39;', "'")
    .replaceAll('&amp;', '&');

/// The PDF's file name for the share sheet: the title without the characters a file name cannot hold.
String pdfName(String title) {
  final name = title.replaceAll(RegExp(r'[\\/:*?"<>|]'), '-').trim();
  return '${name.isEmpty ? 'document' : name}.pdf';
}

String _string(Object? v) => v is String ? v : '';
