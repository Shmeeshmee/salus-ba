import 'package:flutter_test/flutter_test.dart';

import 'support.dart';

const booked = 'd2000000-0000-4000-8000-000000000001';
const revoked = 'd2000000-0000-4000-8000-000000000002';

// A synthetic treatment plan (clinic-cms U6): an examination at the clinic on its first day, a video check-up a week
// later.
Map<String, Object?> plan(String id, String status, {String clinic = clinicId}) => {
  'id': id,
  'organizationId': clinic,
  'patientId': 'd1000000-0000-4000-8000-0000000000d1',
  'status': status,
  'title': 'Placeholder knee plan',
  'startsOn': '2026-11-30',
  'createdAt': '2026-11-20T09:00:00Z',
  'bookedAt': '2026-11-21T10:00:00Z',
  'versionId': 3,
  'steps': [
    {'id': 'd3000000-0000-4000-8000-000000000001', 'position': 1, 'serviceId': 'c1000000-0000-4000-8000-000000000001', 'dayOffset': 0, 'visitMode': 'in-person', 'appointmentId': 'e1000000-0000-4000-8000-000000000001'},
    {'id': 'd3000000-0000-4000-8000-000000000002', 'position': 2, 'serviceId': 'c1000000-0000-4000-8000-000000000002', 'dayOffset': 7, 'visitMode': 'video', 'appointmentId': 'e1000000-0000-4000-8000-000000000002'},
  ],
};

Future<Harness> atPlans(WidgetTester tester, List<Map<String, Object?>> plans) async {
  final h = Harness(store: FakeTokenStore(token));
  h.api.answer('GET /auth/me', 200, account());
  answerDirectory(h.api);
  h.api.answer('GET /me/care-plans', 200, plans);
  await h.pump(tester);
  await tapOutlined(tester, 'My treatment plans');
  return h;
}

void main() {
  testWidgets('lists the plans with their clinic, state, first day and each step as a visit on its day', (tester) async {
    await atPlans(tester, [plan(booked, 'active'), plan(revoked, 'revoked', clinic: 'a1000000-0000-4000-8000-0000000000a1')]);

    expect(find.textContaining('Each step of a plan is a visit'), findsOneWidget);
    expect(find.text('Placeholder knee plan'), findsNWidgets(2));
    expect(find.text(clinicName), findsOneWidget);
    expect(find.text('A clinic no longer listed'), findsOneWidget);
    expect(find.text('Booked'), findsOneWidget);
    expect(find.text('Cancelled'), findsOneWidget);
    expect(find.text('Starts on Monday 30 November 2026'), findsNWidgets(2));
    expect(find.text('1. Monday 30 November 2026 · At the clinic'), findsNWidgets(2));
    expect(find.text('2. Monday 7 December 2026 · By video'), findsNWidgets(2));
  });

  testWidgets('names each step\'s service and shows its visit\'s day after a move', (tester) async {
    final moved = plan(booked, 'active');
    moved['steps'] = [
      {'id': 'd3000000-0000-4000-8000-000000000002', 'position': 1, 'serviceId': 'c1000000-0000-4000-8000-000000000002', 'serviceName': 'Placeholder video check-up', 'dayOffset': 7, 'visitMode': 'video', 'appointmentId': 'e1000000-0000-4000-8000-000000000002', 'visitStart': '2026-12-07T09:00:00Z'},
      // The examination's visit was moved after the check-up: the step shows the visit's day, not its offset's.
      {'id': 'd3000000-0000-4000-8000-000000000001', 'position': 2, 'serviceId': 'c1000000-0000-4000-8000-000000000001', 'serviceName': 'Placeholder examination', 'dayOffset': 0, 'visitMode': 'in-person', 'appointmentId': 'e1000000-0000-4000-8000-000000000001', 'visitStart': '2026-12-09T23:30:00Z'},
    ];
    await atPlans(tester, [moved]);

    expect(find.text('Placeholder video check-up'), findsOneWidget);
    expect(find.text('Placeholder examination'), findsOneWidget);
    expect(find.text('1. Monday 7 December 2026 · By video'), findsOneWidget);
    expect(find.text('2. Thursday 10 December 2026 · At the clinic'), findsOneWidget); // in the clinic's zone
  });

  testWidgets('says when there are no plans', (tester) async {
    await atPlans(tester, const []);

    expect(find.text('You have no treatment plans yet.'), findsOneWidget);
  });

  testWidgets('says so when the plans cannot be read', (tester) async {
    final h = Harness(store: FakeTokenStore(token));
    h.api.answer('GET /auth/me', 200, account());
    answerDirectory(h.api);
    h.api.answer('GET /me/care-plans', 500, error('internal'));
    await h.pump(tester);
    await tapOutlined(tester, 'My treatment plans');

    expect(find.text('Something went wrong. Try again.'), findsOneWidget);
  });
}
