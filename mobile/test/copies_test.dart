import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:salus/copies.dart';

// The copies the pickers and the conversion leave on the device (docs/attachments/SPEC.md §2 "Mobile"): each is
// deleted once its upload ends, and what an earlier run left is swept at start. Laid out as on the devices, in a
// throwaway directory; the files are synthetic.

void main() {
  late Directory root;

  setUp(() => root = Directory.systemTemp.createTempSync('copies-test-'));
  tearDown(() => root.deleteSync(recursive: true));

  File file(String path) => File('${root.path}/$path')
    ..createSync(recursive: true)
    ..writeAsStringSync('synthetic');

  bool exists(String path) => FileSystemEntity.typeSync('${root.path}/$path') != FileSystemEntityType.notFound;

  group('on Android, the temporary directory is code_cache and the pickers copy into cache beside it', () {
    late Directory temp;
    setUp(() {
      temp = Directory('${root.path}/data/code_cache')..createSync(recursive: true);
      Directory('${root.path}/data/cache').createSync();
    });

    for (final (what, path, gone) in [
      ('a camera photo', 'data/cache/0b7e6c1a-2f4d-4e8a-9c3b-5d6e7f8a9b0c123.jpg', <String>[]),
      (
        'a photo from the library, with its folder',
        'data/cache/0b7e6c1a-2f4d-4e8a-9c3b-5d6e7f8a9b0c/x.jpg',
        ['data/cache/0b7e6c1a-2f4d-4e8a-9c3b-5d6e7f8a9b0c'],
      ),
      (
        'a document, with its folders',
        'data/cache/file_picker/1730000000000/x.pdf',
        ['data/cache/file_picker/1730000000000', 'data/cache/file_picker'],
      ),
      ('a copy in the temporary directory', 'data/code_cache/salus-picked-1.pdf', <String>[]),
    ]) {
      test('$what is deleted', () async {
        final f = file(path);
        expect(isAppCopy(f.path, temp), isTrue);
        await forgetCopy(f.path, temp);
        expect(exists(path), isFalse);
        for (final dir in gone) {
          expect(exists(dir), isFalse, reason: dir);
        }
        expect(exists('data/cache'), isTrue, reason: 'the cache directory itself stays');
      });
    }

    test('a folder that still holds another copy stays', () async {
      final f = file('data/cache/file_picker/1/a.pdf');
      file('data/cache/file_picker/2/b.pdf');
      await forgetCopy(f.path, temp);
      expect(exists('data/cache/file_picker/1'), isFalse);
      expect(exists('data/cache/file_picker/2/b.pdf'), isTrue);
    });

    for (final (what, path) in [
      ("the patient's own photo", 'storage/emulated/0/DCIM/Camera/IMG_0001.jpg'),
      ("the patient's own document", 'storage/emulated/0/Download/x.pdf'),
      ('a file elsewhere in the app', 'data/app_flutter/x.jpg'),
    ]) {
      test('$what is never touched', () async {
        final f = file(path);
        expect(isAppCopy(f.path, temp), isFalse);
        await forgetCopy(f.path, temp);
        expect(exists(path), isTrue);
      });
    }

    test('a path that climbs out of the cache is never touched', () async {
      file('storage/x.jpg');
      final climbing = '${root.path}/data/cache/../../storage/x.jpg';
      expect(isAppCopy(climbing, temp), isFalse);
      await forgetCopy(climbing, temp);
      expect(exists('storage/x.jpg'), isTrue);
    });

    test('a directory beside the cache whose name starts the same is not the cache', () {
      expect(isAppCopy(file('data/cache-other/x.jpg').path, temp), isFalse);
    });

    test('at start, what an earlier run left is swept from both; the rest stays', () async {
      final left = [
        'data/code_cache/salus-upload-1-0.jpg',
        'data/code_cache/salus-picked-2.pdf',
        'data/cache/salus-converted-3.mp4',
        'data/cache/0b7e6c1a-2f4d-4e8a-9c3b-5d6e7f8a9b0c123.jpg',
        'data/cache/0B7E6C1A-2F4D-4E8A-9C3B-5D6E7F8A9B0C/x.jpg',
        'data/cache/scaled_x.jpg',
        'data/cache/file_picker/1/x.pdf',
      ];
      final kept = ['data/cache/other.bin', 'data/code_cache/flutter_engine/x', 'storage/DCIM/salus-upload-9.jpg'];
      for (final p in [...left, ...kept]) {
        file(p);
      }
      await sweepCopies(temp);
      for (final p in left) {
        expect(exists(p), isFalse, reason: p);
      }
      expect(exists('data/cache/file_picker'), isFalse);
      expect(exists('data/cache/0B7E6C1A-2F4D-4E8A-9C3B-5D6E7F8A9B0C'), isFalse);
      for (final p in kept) {
        expect(exists(p), isTrue, reason: p);
      }
    });

    test('the sweep is quiet when there is nothing to sweep', () async {
      Directory('${root.path}/data/cache').deleteSync(recursive: true);
      await sweepCopies(temp);
      await sweepCopies(Directory('${root.path}/missing/tmp'));
    });
  });

  group('on iOS the pickers copy into tmp, and the caches are in Library/Caches beside it', () {
    late Directory temp;
    setUp(() => temp = Directory('${root.path}/container/tmp')..createSync(recursive: true));

    test('copies in either are deleted; the documents are not', () async {
      final photo = file('container/tmp/image_picker_ABC.jpg');
      final document = file('container/tmp/7D3C1B2A-0000-4000-8000-000000000001/x.pdf');
      final cached = file('container/Library/Caches/salus-converted-1.jpg');
      final own = file('container/Documents/x.pdf');
      for (final f in [photo, document, cached, own]) {
        await forgetCopy(f.path, temp);
      }
      expect(exists('container/tmp/image_picker_ABC.jpg'), isFalse);
      expect(exists('container/tmp/7D3C1B2A-0000-4000-8000-000000000001'), isFalse);
      expect(exists('container/Library/Caches/salus-converted-1.jpg'), isFalse);
      expect(exists('container/Documents/x.pdf'), isTrue);
    });

    test('at start, image_picker and file_picker leftovers are swept from tmp', () async {
      file('container/tmp/image_picker_ABC.jpg');
      file('container/tmp/7D3C1B2A-0000-4000-8000-000000000001/x.pdf');
      file('container/tmp/salus-upload-1-0.jpg');
      await sweepCopies(temp);
      expect(temp.listSync(), isEmpty);
    });
  });
}
