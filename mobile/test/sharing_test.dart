import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'book_test.dart' show answerAppointments, atBook, appointmentId, count;
import 'support.dart';

// SAL-42: the patient shares chosen visits from other clinics with a clinic, at booking (off by default) and under
// Shared history. Synthetic visits only.

const visit1 = 'e1000000-0000-4000-8000-0000000000e1';
const visit2 = 'e2000000-0000-4000-8000-0000000000e2';
const visit3 = 'e3000000-0000-4000-8000-0000000000e3';
const otherClinic = 'b0000000-0000-4000-8000-00000000000b';
const historyId = 'f4000000-0000-4000-8000-0000000000f4';

const toggle = 'Share my history from other clinics with this clinic';

Map<String, Object?> candidate(
  String id,
  String date, {
  String? title,
  bool selected = false,
  String timeZone = 'Europe/Sarajevo',
}) => {
  'date': date,
  'encounterId': id,
  'organizationId': otherClinic,
  'organizationName': 'Synthetic clinic Mostar',
  'timeZone': timeZone,
  'specialty': '394579002',
  'title': ?title,
  'selected': selected,
};

/// The three previous visits at another clinic, newest first, visit 1 shared (at [version]) when [shared].
Map<String, Object?> candidates({bool shared = false, String version = '$historyId.2'}) => {
  'versionId': ?(shared ? version : null),
  'candidates': [
    candidate(visit3, '2026-10-22T08:00:00Z', title: 'Placeholder findings'),
    candidate(visit2, '2026-10-21T08:00:00Z'),
    candidate(visit1, '2026-10-20T08:00:00Z', selected: shared),
  ],
};

Map<String, Object?> sharing(List<String> shared, {String version = '$historyId.2'}) => {
  'organizationId': clinicId,
  'consentId': historyId,
  'versionId': version,
  'shared': shared,
};

/// A booking that goes through at once: the care consent is there.
void answerBooking(FakeApi api) {
  api.answer('POST /appointments', 201, booked(appointmentId));
  answerAppointments(api);
}

Future<void> tick(WidgetTester tester, String date) => tapFound(tester, find.widgetWithText(CheckboxListTile, date));

Finder inVisits(String text) => find.descendant(of: find.byType(CheckboxListTile), matching: find.text(text));

bool ticked(WidgetTester tester, String date) =>
    tester.widget<CheckboxListTile>(find.widgetWithText(CheckboxListTile, date)).value ?? false;

void main() {
  testWidgets('sharing is off by default at booking when nothing is shared, and shares nothing', (tester) async {
    final h = await atBook(tester, sharing: candidates());
    answerBooking(h.api);

    expect(tester.widget<SwitchListTile>(find.widgetWithText(SwitchListTile, toggle)).value, isFalse);
    expect(find.byType(CheckboxListTile), findsNothing);
    await tapButton(tester, 'Book');

    expect(h.api.calls.where((c) => c.contains('history-sharing')).toList(), ['GET /history-sharing/$clinicId']);
    expect(find.text('Your appointment is booked.'), findsOneWidget);
  });

  testWidgets('an active sharing shows the toggle on with its visits ticked', (tester) async {
    final h = await atBook(tester, sharing: candidates(shared: true));
    h.api.answer('PUT /history-sharing/$clinicId', 200, sharing([visit1], version: '$historyId.2'));
    answerBooking(h.api);

    expect(tester.widget<SwitchListTile>(find.widgetWithText(SwitchListTile, toggle)).value, isTrue);
    expect(ticked(tester, 'Tuesday 20 October 2026'), isTrue);
    expect(ticked(tester, 'Wednesday 21 October 2026'), isFalse);
    await tick(tester, 'Wednesday 21 October 2026');
    await tapButton(tester, 'Book');

    final put = h.api.last('PUT /history-sharing/$clinicId');
    expect(jsonDecode(put.body), {
      'encounterIds': [visit2, visit1],
    });
    expect(put.headers['If-Match'], 'W/"$historyId.2"');
  });

  testWidgets('an active sharing turned off at booking is stopped', (tester) async {
    final h = await atBook(tester, sharing: candidates(shared: true));
    h.api.answer('PUT /history-sharing/$clinicId', 200, {'organizationId': clinicId, 'shared': <String>[]});
    answerBooking(h.api);
    await tapFound(tester, find.widgetWithText(SwitchListTile, toggle));
    await tapButton(tester, 'Book');

    final put = h.api.last('PUT /history-sharing/$clinicId');
    expect(jsonDecode(put.body), {'encounterIds': <String>[]});
    expect(put.headers['If-Match'], 'W/"$historyId.2"');
  });

  testWidgets('each visit is dated in its own clinic\'s time zone', (tester) async {
    final h = await atBook(
      tester,
      sharing: {
        'candidates': [candidate(visit1, '2026-10-21T02:00:00Z', timeZone: 'America/New_York')],
      },
    );
    await tapFound(tester, find.widgetWithText(SwitchListTile, toggle));

    expect(h.api.calls.where((c) => c.contains('history-sharing')).length, 1);
    expect(find.widgetWithText(CheckboxListTile, 'Tuesday 20 October 2026'), findsOneWidget);
  });

  testWidgets('turned on, it previews each visit as the clinic will see it, none ticked', (tester) async {
    final h = await atBook(tester, sharing: candidates());
    h.api.answer('GET /me/history-sharing', 200, {'sharing': <Object>[]});
    await tapFound(tester, find.widgetWithText(SwitchListTile, toggle));

    expect(find.byType(CheckboxListTile), findsNWidgets(3));
    for (final date in ['Thursday 22 October 2026', 'Wednesday 21 October 2026', 'Tuesday 20 October 2026']) {
      expect(ticked(tester, date), isFalse, reason: date);
    }
    expect(inVisits('Synthetic clinic Mostar'), findsNWidgets(3));
    expect(inVisits('Cardiology'), findsNWidgets(3));
    expect(inVisits('Placeholder findings'), findsOneWidget);
    expect(
      find.text(
        'Tick the previous visits $clinicName may see. For each visit you tick, the doctors at $clinicName you have '
        'an appointment with see what is shown here, with the visit\'s documents, questionnaire answers and files, '
        'the allergies and medications recorded in it, and anything added to it later. Visits you do not tick stay '
        'hidden from $clinicName.',
      ),
      findsOneWidget,
    );

    await tapLink(tester, 'Select all');
    expect(tester.widgetList<CheckboxListTile>(find.byType(CheckboxListTile)).every((c) => c.value!), isTrue);
    await tapLink(tester, 'Select none');
    expect(tester.widgetList<CheckboxListTile>(find.byType(CheckboxListTile)).any((c) => c.value!), isFalse);
  });

  testWidgets('booking shares exactly the visits ticked, after the booking', (tester) async {
    final h = await atBook(tester, sharing: candidates());
    h.api.answer('GET /me/history-sharing', 200, {'sharing': <Object>[]});
    h.api.answer('PUT /history-sharing/$clinicId', 200, sharing([visit3, visit1], version: '1'));
    answerBooking(h.api);
    await tapFound(tester, find.widgetWithText(SwitchListTile, toggle));
    await tick(tester, 'Thursday 22 October 2026');
    await tick(tester, 'Tuesday 20 October 2026');
    await tapButton(tester, 'Book');

    final put = h.api.last('PUT /history-sharing/$clinicId');
    expect(jsonDecode(put.body), {
      'encounterIds': [visit3, visit1],
    });
    expect(put.headers.keys.map((k) => k.toLowerCase()), isNot(contains('if-match')));
    expect(h.api.calls.where((c) => c.startsWith('POST') || c.startsWith('PUT')).toList(), [
      'POST /appointments',
      'PUT /history-sharing/$clinicId',
    ]);
    expect(find.text('Your appointment is booked.'), findsOneWidget);
    expectNoCsrfHeader(h.api);
  });

  testWidgets('a sharing that fails after the booking says the appointment is booked', (tester) async {
    final h = await atBook(tester, sharing: candidates());
    h.api.answer('GET /me/history-sharing', 200, {'sharing': <Object>[]});
    h.api.answer('PUT /history-sharing/$clinicId', 400, error('invalid_visit'));
    answerBooking(h.api);
    await tapFound(tester, find.widgetWithText(SwitchListTile, toggle));
    await tick(tester, 'Wednesday 21 October 2026');
    await tapButton(tester, 'Book');

    expect(
      find.text(
        'Your appointment is booked, but your history could not be shared. You can share it under Shared history.',
      ),
      findsOneWidget,
    );
  });

  testWidgets('without previous visits elsewhere there is nothing to tick', (tester) async {
    final h = await atBook(tester, sharing: {'candidates': <Object>[]});
    h.api.answer('GET /me/history-sharing', 200, {'sharing': <Object>[]});
    await tapFound(tester, find.widgetWithText(SwitchListTile, toggle));

    expect(find.text('You have no previous visits at other clinics to share.'), findsOneWidget);
    expect(find.byType(CheckboxListTile), findsNothing);
  });

  group('Shared history', () {
    Future<Harness> atSharedHistory(WidgetTester tester) async {
      final h = Harness(store: FakeTokenStore(token));
      h.api.answer('GET /auth/me', 200, account());
      answerDirectory(h.api);
      h.api.answer('GET /me/consents', 200, {
        'consents': [consent('c1'), consent('h1', kind: 'history')],
      });
      h.api.answer('GET /me/history-sharing', 200, {
        'sharing': [
          sharing([visit1], version: '$historyId.1'), // the list read earlier: an older version than the preview's
        ],
      });
      h.api.answer('GET /history-sharing/$clinicId', 200, candidates(shared: true));
      await h.pump(tester);
      await tapOutlined(tester, 'My consents');
      await tapOutlined(tester, 'Shared history');
      return h;
    }

    testWidgets('lists each consented clinic with what it is shared', (tester) async {
      await atSharedHistory(tester);

      expect(find.text(clinicName), findsOneWidget);
      expect(find.text('1 visit shared'), findsOneWidget);
    });

    testWidgets('changes the visits at the version read', (tester) async {
      final h = await atSharedHistory(tester);
      h.api.answer('PUT /history-sharing/$clinicId', 200, sharing([visit2], version: '$historyId.3'));
      await tapText(tester, clinicName);

      expect(ticked(tester, 'Tuesday 20 October 2026'), isTrue);
      expect(ticked(tester, 'Wednesday 21 October 2026'), isFalse);
      await tick(tester, 'Tuesday 20 October 2026');
      await tick(tester, 'Wednesday 21 October 2026');
      await tapButton(tester, 'Save');

      final put = h.api.last('PUT /history-sharing/$clinicId');
      expect(jsonDecode(put.body), {
        'encounterIds': [visit2],
      });
      expect(put.headers['If-Match'], 'W/"$historyId.2"', reason: 'the version read with the preview');
      expect(find.text('Your choice is saved and takes effect at once.'), findsOneWidget);
    });

    testWidgets('a change made elsewhere meanwhile is read again', (tester) async {
      final h = await atSharedHistory(tester);
      h.api.answer('PUT /history-sharing/$clinicId', 412, error('version_conflict'));
      await tapText(tester, clinicName);
      await tick(tester, 'Wednesday 21 October 2026');
      await tapButton(tester, 'Save');

      expect(find.text('Your shared visits changed in the meantime. Check them and save again.'), findsOneWidget);
      expect(count(h.api, 'GET /history-sharing/$clinicId'), 2);
    });

    testWidgets('stops sharing after the confirmation', (tester) async {
      final h = await atSharedHistory(tester);
      h.api.answer('DELETE /history-sharing/$clinicId', 204);
      await tapText(tester, clinicName);
      await tapOutlined(tester, 'Stop sharing');

      expect(find.text('Stop sharing your history?'), findsOneWidget);
      expect(
        find.text(
          '$clinicName will no longer see any of the visits from other clinics you shared with it. This takes effect '
          'at once.',
        ),
        findsOneWidget,
      );
      await tapLink(tester, 'Keep sharing');
      expect(count(h.api, 'DELETE /history-sharing/$clinicId'), 0);

      h.api.answer('GET /me/history-sharing', 200, {'sharing': <Object>[]});
      h.api.answer('GET /history-sharing/$clinicId', 200, candidates());
      await tapOutlined(tester, 'Stop sharing');
      await tapButton(tester, 'Stop sharing');

      expect(count(h.api, 'DELETE /history-sharing/$clinicId'), 1);
      expect(find.text('You no longer share your history with this clinic.'), findsOneWidget);
    });
  });

  testWidgets('My consents names a history consent and says what revoking it does', (tester) async {
    final h = Harness(store: FakeTokenStore(token));
    h.api.answer('GET /auth/me', 200, account());
    answerDirectory(h.api);
    h.api.answer('GET /me/consents', 200, {
      'consents': [consent('h1', kind: 'history')],
    });
    await h.pump(tester);
    await tapOutlined(tester, 'My consents');

    expect(find.text('Visits shared from other clinics'), findsOneWidget);
    await tapOutlined(tester, 'Revoke consent');
    expect(find.text('Stop sharing your history?'), findsOneWidget);
    expect(find.widgetWithText(TextButton, 'Keep sharing'), findsOneWidget);
  });
}
