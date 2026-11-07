import 'package:flutter_test/flutter_test.dart';
import 'package:salus/files.dart';
import 'package:salus/questionnaire.dart';
import 'package:salus/uploads.dart';

import 'support.dart';

// The files of a response as the answers see them (uploads.dart). Synthetic files and titles only.

void main() {
  for (final type in ['image/png', 'image/jpeg', 'application/pdf', 'video/mp4']) {
    test('an uploaded file keeps the type the server detected: $type', () async {
      final h = Harness()..files.kind = kindOf(type)!;
      h.api.answer('POST /questionnaire-responses/r1/items/files/attachments', 201, {
        'id': 'b1',
        'contentType': type,
        'size': 3000,
        'title': 'synthetic-file',
        'status': 'clean',
      });
      final uploads = Uploads(
        booking: h.booking,
        files: h.files,
        responseId: 'r1',
        items: const {},
        attached: const {},
        onEnded: (_) {},
      );
      addTearDown(uploads.dispose);
      await uploads.add('files', const Picked(path: '/synthetic/cache/file', name: 'synthetic-file'));
      await pumpEventQueue();

      final attached = uploads.answers['files']!.single as Attached;
      expect(attached.contentType, type);
    });
  }
}
