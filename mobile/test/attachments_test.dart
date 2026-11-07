import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:salus/call.dart' show Access, Kind;
import 'package:salus/file_prep.dart';
import 'package:salus/files.dart';
import 'package:salus/questionnaire.dart';
import 'package:salus/booking.dart' show contentDisposition;
import 'package:salus/questionnaire_files.dart';
import 'package:salus/uploads.dart' show pollDelay;

import 'questionnaires.dart';
import 'support.dart';

// Files on a questionnaire (docs/attachments/SPEC.md §2 "Mobile"; COPY.md M1 to M20). Synthetic files and titles
// only; the pickers and the conversion are faked (fake_files.dart), the API is the generated client's MockClient.

const _upload = 'POST /questionnaire-responses/r1/items/files/attachments';
const _status = 'GET /binaries/b1';
const _submit = 'POST /questionnaire-responses/r1/submit';
const _start = '2026-11-05T08:00:00Z';

Map<String, Object?> _version({List<String> accept = const ['photo', 'document'], int maxFiles = 2}) => questionnaire(
  items: [
    attachmentItem('files', accept: accept, maxFiles: maxFiles, required: true, text: 'Placeholder photos of the area'),
    {'linkId': 'name', 'text': 'Placeholder short answer', 'type': 'string'},
  ],
);

Map<String, Object?> _binary(String status, {String id = 'b1', String title = 'synthetic-photo.jpg'}) => {
  'id': id,
  'contentType': 'image/jpeg',
  'size': 3000,
  'title': title,
  'status': status,
};

/// The questionnaire of [version] opened from its appointment card, with [r] as the patient's response.
Future<Harness> _opened(
  WidgetTester tester, {
  Map<String, Object?>? version,
  Map<String, Object?>? r,
  String button = 'Fill in',
}) async {
  final h = Harness(store: FakeTokenStore(token));
  h.api.answer('GET /auth/me', 200, account());
  answerDirectory(h.api);
  h.api.on(
    'GET /fhir/R4/Appointment',
    (_) async => fhirResponse(200, bundle([appointment(appointmentId, _start, '2026-11-05T08:30:00Z')])),
  );
  h.api.answer('GET /me/consents', 200, {
    'consents': [consent('c1')],
  });
  h.api.on(
    'GET /fhir/R4/QuestionnaireResponse',
    (_) async => fhirResponse(200, bundle([r ?? response('r1')], included: [version ?? _version()])),
  );
  h.api.answer(_upload, 201, _binary('quarantined'));
  h.api.answer(_status, 200, _binary('clean'));
  await h.pump(tester);
  await tapOutlined(tester, 'My appointments');
  await tapFound(tester, find.text(button));
  return h;
}

final _item = find.byKey(const ValueKey('answer-files'));

Finder _inItem(String text) => find.descendant(of: _item, matching: find.text(text));

Future<void> _addFrom(WidgetTester tester, String source) async {
  await tapFound(tester, find.widgetWithText(OutlinedButton, 'Add file'));
  await tapText(tester, source);
}

/// Waits out the status polls until nothing is pending.
Future<void> _settlePolls(WidgetTester tester) async {
  for (var i = 0; i < 3; i++) {
    await tester.pump(const Duration(seconds: 2));
    await tester.pumpAndSettle();
  }
}

List<http.Request> _uploads(Harness h) => [
  for (final r in h.api.requests)
    if ('${r.method} ${r.url.path}' == _upload) r,
];

/// Submit, scrolled to: the page is a lazy list.
Future<FilledButton> _submitButton(WidgetTester tester) async {
  final button = find.widgetWithText(FilledButton, 'Submit');
  final list = find.descendant(of: find.byType(ListView), matching: find.byType(Scrollable)).first;
  await tester.scrollUntilVisible(button, 200, scrollable: list);
  return tester.widget<FilledButton>(button);
}

void main() {
  group('the hint names what the item takes', () {
    for (final (accept, maxFiles, hint) in [
      ([FileKind.photo], 1, 'Photos up to 20 MB. 1 file.'),
      ([FileKind.document], 3, 'Documents (PDF) up to 20 MB. Up to 3 files.'),
      ([FileKind.video], 1, 'Videos up to 200 MB. 1 file.'),
      ([FileKind.video, FileKind.photo], 10, 'Photos up to 20 MB. Videos up to 200 MB. Up to 10 files.'),
      (
        [FileKind.document, FileKind.video, FileKind.photo],
        2,
        'Photos up to 20 MB. Documents (PDF) up to 20 MB. Videos up to 200 MB. Up to 2 files.',
      ),
    ]) {
      test(hint, () {
        final item = Item(linkId: 'f', type: ItemType.attachment, accept: accept.toSet(), maxFiles: maxFiles);
        expect(fileHint(item), hint);
      });
    }
  });

  group('the source sheet', () {
    for (final (accept, sources) in [
      (['photo'], ['Take a photo', 'Choose from photos']),
      (['video'], ['Record a video', 'Choose from photos']),
      (['document'], ['Choose a document']),
      (['photo', 'document', 'video'], ['Take a photo', 'Record a video', 'Choose from photos', 'Choose a document']),
    ]) {
      testWidgets('for ${accept.join(' and ')} offers only ${sources.join(', ')}', (tester) async {
        await _opened(tester, version: _version(accept: accept));
        expect(_inItem('Placeholder photos of the area'), findsOneWidget);
        await tapFound(tester, find.widgetWithText(OutlinedButton, 'Add file'));

        final all = ['Take a photo', 'Record a video', 'Choose from photos', 'Choose a document'];
        for (final s in all) {
          expect(find.text(s), sources.contains(s) ? findsOneWidget : findsNothing, reason: s);
        }
        expect(find.text('Cancel'), findsOneWidget);
        await tapText(tester, 'Cancel');
        expect(find.text('Take a photo'), findsNothing);
      });
    }

    testWidgets('the library is asked for what the item accepts', (tester) async {
      final h = await _opened(tester, version: _version(accept: const ['photo', 'video']));
      await _addFrom(tester, 'Choose from photos');
      expect(h.files.picks.single, (Source.library, true, true));
    });

    testWidgets('a cancelled pick adds nothing', (tester) async {
      final h = await _opened(tester);
      h.files.next = null;
      await _addFrom(tester, 'Choose from photos');
      expect(_uploads(h), isEmpty);
      expect(_inItem('Preparing…'), findsNothing);
    });
  });

  group('uploading', () {
    testWidgets('prepares, uploads the whole file streamed with its length and name, then checks it', (tester) async {
      final h = await _opened(tester);
      h.files.preparing = Completer();
      h.files.name = 'Synthetic photo (1) ü.jpg';
      await _addFrom(tester, 'Choose from photos');

      expect(_inItem('Preparing…'), findsOneWidget);
      expect((await _submitButton(tester)).onPressed, isNull, reason: 'Submit waits for the file');
      h.files.preparing!.complete();
      await tester.pumpAndSettle();

      final request = _uploads(h).single;
      expect(request.bodyBytes, [...h.files.chunks.expand((c) => c)]);
      expect(h.api.declared.where((d) => d.$1 == _upload).single.$2, 3000, reason: 'Content-Length, not chunked');
      expect(request.headers['Content-Type'], 'application/octet-stream');
      expect(
        request.headers['Content-Disposition'],
        "attachment; filename*=UTF-8''Synthetic%20photo%20%281%29%20%C3%BC.jpg",
      );
      expect(request.headers['Authorization'], 'Bearer $token');
      expect(request.url.query, isEmpty, reason: 'the name never in the query string');
      expectNoCsrfHeader(h.api);

      expect(_inItem('Checking the file…'), findsOneWidget);
      expect(find.text('Some files are still being checked. You can submit once they are ready.'), findsOneWidget);
      expect((await _submitButton(tester)).onPressed, isNull);
      expect(h.files.discarded, 1, reason: 'the copies are deleted once uploaded');

      h.api.answer(_status, 200, _binary('quarantined'));
      await tester.pump(const Duration(seconds: 2));
      await tester.pumpAndSettle();
      expect(h.api.calls.where((c) => c == _status), hasLength(1));
      expect(_inItem('Checking the file…'), findsOneWidget);

      h.api.answer(_status, 200, _binary('clean'));
      await tester.pump(const Duration(seconds: 2));
      await tester.pumpAndSettle();
      expect(_inItem('Added'), findsOneWidget);
      expect(_inItem('Photo, 3 KB'), findsOneWidget);
      expect(_inItem('Synthetic photo (1) ü.jpg'), findsOneWidget);
      expect(find.text('Some files are still being checked. You can submit once they are ready.'), findsNothing);
      expect((await _submitButton(tester)).onPressed, isNotNull);
    });

    testWidgets('shows the progress while the bytes go', (tester) async {
      final h = await _opened(tester);
      h.files.gate = [Completer(), Completer(), Completer()];
      final semantics = tester.ensureSemantics();
      await _addFrom(tester, 'Choose from photos');

      expect(_inItem('Uploading… 0%'), findsOneWidget);
      h.files.gate![0].complete();
      await tester.pump();
      await tester.pump();
      expect(_inItem('Uploading… 33%'), findsOneWidget);
      expect(find.bySemanticsLabel('Uploading synthetic-photo.jpg, 33 percent'), findsOneWidget);
      h.files.gate![1].complete();
      h.files.gate![2].complete();
      await tester.pumpAndSettle();
      expect(_inItem('Checking the file…'), findsOneWidget);
      expect(find.bySemanticsLabel('Photo, synthetic-photo.jpg, Checking the file…'), findsOneWidget);
      semantics.dispose();
    });

    testWidgets('the polls slow down from every 2 s to every 10 s', (tester) async {
      expect([for (var n = 0; n < 10; n++) pollDelay(n).inSeconds], [2, 2, 2, 2, 2, 4, 6, 8, 10, 10]);
    });

    testWidgets('a failed upload says so; Try again sends the whole file again', (tester) async {
      final h = await _opened(tester);
      h.api.on(_upload, (_) async => throw http.ClientException('network'));
      await _addFrom(tester, 'Choose from photos');

      expect(_inItem('Not uploaded. Check your connection.'), findsOneWidget);
      expect(h.files.discarded, 0, reason: 'kept for the retry');
      h.api.answer(_upload, 200, _binary('clean'));
      await tapFound(tester, find.descendant(of: _item, matching: find.widgetWithText(TextButton, 'Try again')));

      expect(_uploads(h), hasLength(2));
      expect(_uploads(h).last.bodyBytes, hasLength(3000));
      expect(_inItem('Added'), findsOneWidget, reason: 'a retried upload the server already had: 200, clean');
    });

    testWidgets('an upload that stalls stops after 60 s without progress and says so; Try again sends it again', (
      tester,
    ) async {
      final h = await _opened(tester);
      h.files.gate = [Completer()..complete(), Completer(), Completer()];
      await _addFrom(tester, 'Choose from photos');
      await tester.pump();
      expect(_inItem('Uploading… 33%'), findsOneWidget);

      await tester.pump(const Duration(seconds: 59));
      expect(_inItem('Uploading… 33%'), findsOneWidget, reason: 'slow is not stalled');
      await tester.pump(const Duration(seconds: 1));
      await tester.pumpAndSettle();
      expect(_inItem('Not uploaded. Check your connection.'), findsOneWidget);
      expect(find.descendant(of: _item, matching: find.widgetWithText(TextButton, 'Remove')), findsOneWidget);
      expect(h.files.discarded, 0, reason: 'kept for the retry');

      h.files.gate = null;
      await tapFound(tester, find.descendant(of: _item, matching: find.widgetWithText(TextButton, 'Try again')));
      expect(_inItem('Checking the file…'), findsOneWidget);
      await _settlePolls(tester);
      expect(_inItem('Added'), findsOneWidget);
    });

    testWidgets('an upload whose answer does not come stops 60 s after its last byte', (tester) async {
      final h = await _opened(tester);
      h.api.on(_upload, (_) => Completer<http.Response>().future);
      await _addFrom(tester, 'Choose from photos');
      await tester.pump(const Duration(seconds: 60));
      await tester.pumpAndSettle();
      expect(_inItem('Not uploaded. Check your connection.'), findsOneWidget);
    });

    testWidgets('a failed upload can be removed without asking: it is nowhere but here', (tester) async {
      final h = await _opened(tester);
      h.api.answer(_upload, 503, error('unavailable'));
      await _addFrom(tester, 'Choose from photos');

      await tapFound(tester, find.descendant(of: _item, matching: find.widgetWithText(TextButton, 'Remove')));
      expect(_inItem('Not uploaded. Check your connection.'), findsNothing);
      expect(h.files.discarded, 1);
      expect(h.api.calls.where((c) => c.startsWith('DELETE')), isEmpty);
    });

    for (final (status, code, text) in [
      (413, 'too_large', 'This file is larger than 20 MB.'),
      (415, 'unsupported_type', 'This kind of file cannot be added to this question.'),
      (409, 'too_many_files', 'You can add up to 2 files to this question.'),
      (413, 'response_full', 'The files in this questionnaire have reached 500 MB. Remove a file to add another.'),
      (400, 'invalid_request', 'The file could not be uploaded. Try again.'),
      (422, 'not_attachment_item', 'The file could not be uploaded. Try again.'),
    ]) {
      testWidgets('$code says "$text" on the item', (tester) async {
        final h = await _opened(tester);
        h.api.answer(_upload, status, error(code));
        await _addFrom(tester, 'Choose from photos');

        expect(_inItem(text), findsOneWidget);
        expect(_inItem('synthetic-photo.jpg'), findsNothing, reason: 'nothing was added');
        expect(h.files.discarded, 1);
        expect(find.widgetWithText(OutlinedButton, 'Add file'), findsOneWidget, reason: 'the patient chooses another');
      });
    }

    testWidgets('a video too large says the video limit', (tester) async {
      final h = await _opened(tester, version: _version(accept: const ['video']));
      h.files.kind = FileKind.video;
      h.api.answer(_upload, 413, error('too_large'));
      await _addFrom(tester, 'Choose from photos');
      expect(_inItem('This file is larger than 200 MB.'), findsOneWidget);
    });

    for (final (problem, text) in [
      (const CannotPrepare(), 'This file could not be prepared for upload. Choose another file.'),
      (const TooLarge(FileKind.photo), 'This file is larger than 20 MB.'),
    ]) {
      testWidgets('a file that cannot be prepared says "$text"', (tester) async {
        final h = await _opened(tester);
        h.files.prepareError = problem;
        await _addFrom(tester, 'Choose from photos');
        expect(_inItem(text), findsOneWidget);
        expect(_uploads(h), isEmpty);
      });
    }

    testWidgets('a refusal of the response ends filling in, as for the answers', (tester) async {
      final h = await _opened(tester);
      h.api.answer(_upload, 409, error('submitted'));
      await _addFrom(tester, 'Choose from photos');

      expect(find.text('You have already submitted this questionnaire.'), findsOneWidget);
      expect(find.widgetWithText(FilledButton, 'Submit'), findsNothing);
      expect(find.widgetWithText(OutlinedButton, 'Add file'), findsNothing);
    });

    testWidgets('a file the scan rejected is gone, with M7', (tester) async {
      final h = await _opened(tester);
      h.api.answer(_status, 200, _binary('rejected'));
      await _addFrom(tester, 'Choose from photos');
      await tester.pump(const Duration(seconds: 2));
      await tester.pumpAndSettle();

      expect(_inItem('This file did not pass the safety check and was deleted. Choose another file.'), findsOneWidget);
      expect(_inItem('synthetic-photo.jpg'), findsNothing);
      expect((await _submitButton(tester)).onPressed, isNotNull);
    });

    testWidgets('one file at a time: the second waits for the first', (tester) async {
      final h = await _opened(tester);
      final first = Completer<http.Response>();
      var n = 0;
      h.api.on(_upload, (_) => n++ == 0 ? first.future : Future.value(jsonResponse(201, _binary('clean', id: 'b2'))));
      await _addFrom(tester, 'Choose from photos');
      await _addFrom(tester, 'Choose from photos');
      expect(_uploads(h), hasLength(1));

      first.complete(jsonResponse(201, _binary('clean')));
      await tester.pumpAndSettle();
      expect(_uploads(h), hasLength(2));
      expect(_inItem('Added'), findsNWidgets(2));
      expect(find.widgetWithText(OutlinedButton, 'Add file'), findsNothing, reason: 'the item holds its 2 files');
    });

    testWidgets('prints nothing: no title, no file in any log', (tester) async {
      final printed = <String?>[];
      final original = debugPrint;
      debugPrint = (String? message, {int? wrapWidth}) => printed.add(message);
      try {
        final h = await _opened(tester);
        h.api.on(_upload, (_) async => throw http.ClientException('network'));
        await _addFrom(tester, 'Choose from photos');
        h.api.answer(_upload, 201, _binary('quarantined'));
        await tapFound(tester, find.descendant(of: _item, matching: find.widgetWithText(TextButton, 'Try again')));
        await _settlePolls(tester);
      } finally {
        debugPrint = original;
      }
      expect(printed, isEmpty);
    });
  });

  group('removing and submitting', () {
    testWidgets('an added file is removed after asking, and deleted on the server', (tester) async {
      final h = await _opened(tester);
      h.api.answer('DELETE /binaries/b1', 204);
      await _addFrom(tester, 'Choose from photos');
      await _settlePolls(tester);

      await tapFound(tester, find.descendant(of: _item, matching: find.byTooltip('Remove file')));
      expect(find.text('Remove this file?'), findsOneWidget);
      expect(find.text('It is deleted from your answers.'), findsOneWidget);
      await tapLink(tester, 'Cancel');
      expect(h.api.calls.where((c) => c.startsWith('DELETE')), isEmpty);

      await tapFound(tester, find.descendant(of: _item, matching: find.byTooltip('Remove file')));
      await tapLink(tester, 'Remove');
      expect(h.api.calls.last, 'DELETE /binaries/b1');
      expect(_inItem('synthetic-photo.jpg'), findsNothing);
    });

    testWidgets('a required file upload item without a file is required on submit', (tester) async {
      await _opened(tester);
      await tapFound(tester, find.widgetWithText(FilledButton, 'Submit'));
      expect(_inItem('This question is required.'), findsOneWidget);
    });

    testWidgets('a 409 attachments_pending says M10 and filling in goes on', (tester) async {
      final h = await _opened(tester);
      await _addFrom(tester, 'Choose from photos');
      await _settlePolls(tester);
      h.api.answer(_submit, 409, error('attachments_pending'));

      await tapFound(tester, find.widgetWithText(FilledButton, 'Submit'));
      await tapFound(tester, find.descendant(of: find.byType(AlertDialog), matching: find.text('Submit')));

      expect(find.text('Some files are still being checked. You can submit once they are ready.'), findsOneWidget);
      expect(find.widgetWithText(FilledButton, 'Submit'), findsOneWidget, reason: 'not closed');
    });

    testWidgets('submitted, the files are listed read only', (tester) async {
      final h = await _opened(tester);
      await _addFrom(tester, 'Choose from photos');
      await _settlePolls(tester);
      h.api.answer(_submit, 200, saved('r1', status: 'completed', authored: '2026-11-04T09:00:00Z'));

      await tapFound(tester, find.widgetWithText(FilledButton, 'Submit'));
      await tapFound(tester, find.descendant(of: find.byType(AlertDialog), matching: find.text('Submit')));

      expect(find.text('Answers submitted'), findsOneWidget);
      expect(find.text('1 file'), findsOneWidget);
      expect(find.text('synthetic-photo.jpg'), findsOneWidget);
      expect(find.text('Photo, 3 KB'), findsOneWidget);
    });

    testWidgets('leaving during an upload asks first; leaving stops it', (tester) async {
      final h = await _opened(tester);
      final held = Completer<http.Response>();
      h.api.on(_upload, (_) => held.future);
      await _addFrom(tester, 'Choose from photos');

      await tester.tap(find.byType(BackButton));
      await tester.pumpAndSettle();
      expect(find.text('A file is still uploading. Leave anyway? The upload stops.'), findsOneWidget);
      await tapLink(tester, 'Stay');
      expect(find.text('Placeholder intake questionnaire'), findsOneWidget);

      await tester.tap(find.byType(BackButton));
      await tester.pumpAndSettle();
      await tapLink(tester, 'Leave');
      expect(find.text('My appointments'), findsOneWidget);
      held.complete(jsonResponse(201, _binary('quarantined')));
      await tester.pumpAndSettle();
      expect(h.files.discarded, 1);
      expect(tester.takeException(), isNull);
    });
  });

  group('files already there', () {
    testWidgets('an in-progress response lists its files with their status', (tester) async {
      final h = await _opened(
        tester,
        button: 'Continue',
        r: response(
          'r1',
          items: [
            {
              'linkId': 'files',
              'answer': [fileAnswer('b1', title: 'synthetic-earlier.jpg')],
            },
          ],
        ),
      );
      expect(h.api.calls, contains(_status));
      expect(_inItem('synthetic-earlier.jpg'), findsOneWidget);
      expect(_inItem('Added'), findsOneWidget);
    });

    testWidgets('a submitted response lists title, kind and size; none is Not answered', (tester) async {
      await _opened(
        tester,
        button: 'View answers',
        version: _version(),
        r: response(
          'r1',
          status: 'completed',
          authored: '2026-11-03T08:00:00Z',
          items: [
            {
              'linkId': 'files',
              'answer': [
                fileAnswer('b1', title: 'synthetic-photo.jpg'),
                fileAnswer('b2', contentType: 'application/pdf', size: 1250000, title: 'synthetic-letter.pdf'),
              ],
            },
          ],
        ),
      );
      expect(find.text('2 files'), findsOneWidget);
      expect(find.text('synthetic-photo.jpg'), findsOneWidget);
      expect(find.text('Photo, 245 KB'), findsOneWidget);
      expect(find.text('synthetic-letter.pdf'), findsOneWidget);
      expect(find.text('Document, 1.3 MB'), findsOneWidget);
      expect(find.text('Not answered'), findsOneWidget, reason: 'the short answer');
    });
  });

  group('permissions', () {
    tearDown(() => debugDefaultTargetPlatformOverride = null);

    testWidgets('on Android the camera is explained before the system asks (M18)', (tester) async {
      debugDefaultTargetPlatformOverride = TargetPlatform.android;
      final h = await _opened(tester);
      h.media.state[Kind.camera] = Access.unasked;

      await _addFrom(tester, 'Take a photo');
      expect(find.text('Allow camera access'), findsOneWidget);
      expect(
        find.text('Salus needs the camera to take the photo or video you add to this questionnaire.'),
        findsOneWidget,
      );
      await tapLink(tester, 'Not now');
      expect(h.media.asked, isEmpty);
      expect(h.files.picks, isEmpty);

      await _addFrom(tester, 'Take a photo');
      await tapLink(tester, 'Continue');
      expect(h.media.asked, [Kind.camera]);
      expect(h.files.picks.single.$1, Source.camera);
      debugDefaultTargetPlatformOverride = null;
    });

    testWidgets('a refused camera says so and opens the settings (M19)', (tester) async {
      final h = await _opened(tester);
      h.media.state[Kind.camera] = Access.refused;

      await _addFrom(tester, 'Take a photo');
      expect(
        find.text(
          'Salus does not have access to your camera. You can allow it in your device settings, or choose a file '
          'instead.',
        ),
        findsOneWidget,
      );
      await tapLink(tester, 'Open settings');
      expect(h.media.settingsOpened, 1);
      expect(h.files.picks, isEmpty);
    });

    testWidgets('the picker refused the camera or the photos (M19, M20)', (tester) async {
      final h = await _opened(tester);
      h.files.pickError = const AccessRefused(Refused.photos);
      await _addFrom(tester, 'Choose from photos');
      expect(
        find.text('Salus does not have access to your photos. You can allow it in your device settings.'),
        findsOneWidget,
      );
      await tapLink(tester, 'Cancel');

      h.files.pickError = const AccessRefused(Refused.camera);
      await _addFrom(tester, 'Take a photo');
      expect(find.textContaining('Salus does not have access to your camera.'), findsOneWidget);
    });
  });

  test('sizes are written in KB under 1 MB, in MB with one decimal from there', () {
    expect(
      [
        for (final n in [1, 999, 245000, 999499, 1000000, 1250000, 200000000]) fileSize(n),
      ],
      ['1 KB', '1 KB', '245 KB', '999 KB', '1.0 MB', '1.3 MB', '200.0 MB'],
    );
  });

  test('a content disposition names the file in RFC 6266 form', () {
    expect(jsonEncode(contentDisposition("a'b*c!.pdf")), jsonEncode("attachment; filename*=UTF-8''a%27b%2Ac%21.pdf"));
  });
}
