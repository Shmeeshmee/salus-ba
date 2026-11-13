import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'support.dart';

const entryId = 'e1000000-0000-4000-8000-000000000001';
const offerId = '0f000000-0000-4000-8000-000000000001';

/// Patient A's entry for Cy Example until 18 November; with [offer], a time offered on Thursday 5 November at 09:00
/// (Europe/Sarajevo), held until 10:15.
Map<String, Object?> entry({bool offer = false}) => {
  'id': entryId,
  'organization_id': clinicId,
  'practitioner_role_id': roleId,
  'earliest': '2026-11-04',
  'latest': '2026-11-18',
  'status': 'active',
  'created_at': '2026-11-04T09:00:00Z',
  if (offer)
    'offer': {
      'id': offerId,
      'slot_start': '2026-11-05T08:00:00Z',
      'slot_end': '2026-11-05T08:30:00Z',
      'expires_at': '2026-11-04T09:15:00Z',
    },
};

Future<Harness> signedIn(WidgetTester tester) async {
  final h = Harness(store: FakeTokenStore(token));
  h.api.answer('GET /auth/me', 200, account());
  answerDirectory(h.api);
  h.api.on('GET /fhir/R4/Schedule', (_) async => fhirResponse(200, bundle([schedule()])));
  h.api.on('GET /fhir/R4/Slot', (_) async => fhirResponse(200, bundle(const [])));
  await h.pump(tester);
  return h;
}

void main() {
  testWidgets('a patient joins a practitioner waitlist from their free times', (tester) async {
    final h = await signedIn(tester);
    h.api.answer('POST /waitlist', 201, entry());
    h.api.answer('GET /me/waitlist', 200, {
      'entries': [entry()],
    });
    await tapButton(tester, 'Book an appointment');
    await tapText(tester, clinicName);
    await tapText(tester, 'Cy Example');
    await tapLink(tester, 'No suitable time? Join the waitlist');
    expect(find.textContaining('one at a time'), findsOneWidget);
    await tapFound(tester, find.widgetWithText(ChoiceChip, 'Mornings, 08:00 to 12:00'));
    await tapButton(tester, 'Join the waitlist');
    expect(sent(h.api.last('POST /waitlist').body), {
      'organization_id': clinicId,
      'practitioner_role_id': roleId,
      'earliest': '2026-11-04',
      'latest': '2026-11-18',
      'times': {'start': '08:00', 'end': '12:00'},
    });
    expect(find.text('You are on the waitlist.'), findsOneWidget);
    expect(find.text('Cy Example'), findsOneWidget);
  });

  testWidgets('a refused join says why', (tester) async {
    final h = await signedIn(tester);
    h.api.answer('POST /waitlist', 409, error('waitlist_off'));
    await tapButton(tester, 'Book an appointment');
    await tapText(tester, clinicName);
    await tapText(tester, 'Cy Example');
    await tapLink(tester, 'No suitable time? Join the waitlist');
    await tapButton(tester, 'Join the waitlist');
    expect(find.text('This clinic has no waitlist.'), findsOneWidget);
  });

  testWidgets('an offered time is accepted in the app, after saying that it books it', (tester) async {
    final h = await signedIn(tester);
    var offered = true;
    h.api.on('GET /me/waitlist', (_) async => jsonResponse(200, {
      'entries': [entry(offer: offered)],
    }));
    h.api.on('POST /waitlist/offers/$offerId/accept', (_) async {
      offered = false;
      return jsonResponse(201, booked('e9000000-0000-4000-8000-000000000001'));
    });
    await tapOutlined(tester, 'My waitlist');
    expect(find.text('A time is free for you'), findsOneWidget);
    expect(find.text('09:00 to 09:30 (Europe/Sarajevo)'), findsOneWidget);
    expect(find.text('Held for you until 10:15'), findsOneWidget);
    expect(find.textContaining('Accepting books this time for you.'), findsOneWidget);
    await tapButton(tester, 'Accept and book');
    expect(h.api.calls, contains('POST /waitlist/offers/$offerId/accept'));
    expect(find.text('Your appointment is booked. You find it under My appointments.'), findsOneWidget);
    expect(find.text('A time is free for you'), findsNothing);
  });

  testWidgets('a patient declines an offer, or leaves the waitlist', (tester) async {
    final h = await signedIn(tester);
    h.api.answer('GET /me/waitlist', 200, {
      'entries': [entry(offer: true)],
    });
    h.api.answer('POST /waitlist/offers/$offerId/decline', 204);
    h.api.answer('DELETE /waitlist/$entryId', 204);
    await tapOutlined(tester, 'My waitlist');
    await tapOutlined(tester, 'Decline');
    expect(find.text('You declined the time. You stay on the waitlist.'), findsOneWidget);
    await tapLink(tester, 'Leave the waitlist');
    expect(h.api.calls, contains('DELETE /waitlist/$entryId'));
    expect(find.text('You left the waitlist.'), findsOneWidget);
    expect(jsonEncode(h.api.calls), isNot(contains('accept')));
  });
}
