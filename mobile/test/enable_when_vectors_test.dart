import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:salus/questionnaire.dart';

// The shared enableWhen vectors of test/testdata/enablewhen (docs/clinic-cms, package Q1): the server and the web
// builder's preview run the same files, so the app shows the path the server accepts. Each questionnaire and its
// answers are read the way the app reads them from the API.
void main() {
  final files = Directory('../test/testdata/enablewhen').listSync().whereType<File>().where((f) => f.path.endsWith('.json')).toList()
    ..sort((a, b) => a.path.compareTo(b.path));

  test('the vectors are there', () => expect(files, isNotEmpty));

  for (final file in files) {
    final vector = jsonDecode(file.readAsStringSync()) as Map<String, dynamic>;
    final items = Version.fromFhir(vector['questionnaire'] as Map<String, dynamic>).items;
    group(vector['name'], () {
      for (final c in (vector['cases'] as List).cast<Map<String, dynamic>>()) {
        test(c['name'], () {
          final response = Response.fromFhir({
            'item': [
              for (final MapEntry(:key, :value) in (c['answers'] as Map<String, dynamic>).entries)
                {'linkId': key, 'answer': value},
            ],
          });
          final on = enabled(items, response.answers);
          expect([for (final it in flatten(items)) if (on[it.linkId]!) it.linkId], c['enabled']);
        });
      }
    });
  }
}
