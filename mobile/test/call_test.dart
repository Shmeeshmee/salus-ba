import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:salus/call.dart';

import 'support.dart';

// Now is Wednesday 4 November 2026, 09:00 UTC (support.dart); the appointment starts at 09:05.
const joinRoute = 'POST /appointments/now/consultation/join';
const statusRoute = 'GET /appointments/now/consultation';

// Synthetic: no real LiveKit, token or person.
Map<String, Object?> joined({String peerName = 'Cy Example'}) => {
  'url': 'ws://livekit.test',
  'token': 'synthetic-livekit-token',
  'expiresAt': '2026-11-04T09:05:00Z',
  'peerName': peerName,
  'role': 'patient',
};

Map<String, Object?> status(String state, {String? endReason, String? reason}) => {
  'state': state,
  'patientJoinedAt': '2026-11-04T09:00:00Z',
  'practitionerJoinedAt': ?(endReason == 'practitioner_absent' ? null : '2026-11-04T09:00:00Z'),
  'endedAt': ?(state == 'ended' ? '2026-11-04T09:30:00Z' : null),
  'endReason': ?endReason,
  'reason': ?reason,
};

const practitioner = Peer(present: true, camera: true, microphone: true);
const nobody = Peer(present: false, camera: false, microphone: false);

/// Signed in, on my appointments, Join consultation tapped: the device check.
Future<Harness> atCheck(WidgetTester tester, {void Function(Harness h)? before}) async {
  final h = Harness(store: FakeTokenStore(token));
  h.api.answer('GET /auth/me', 200, account());
  answerDirectory(h.api);
  h.api.on(
    'GET /fhir/R4/Appointment',
    (_) async => fhirResponse(200, bundle([appointment('now', '2026-11-04T09:05:00Z', '2026-11-04T09:35:00Z')])),
  );
  h.api.answer('GET /me/consents', 200, {
    'consents': [consent('c1')],
  });
  h.api.answer(joinRoute, 200, joined());
  h.api.answer(statusRoute, 200, status('open'));
  before?.call(h);
  await h.pump(tester);
  await tapOutlined(tester, 'My appointments');
  await tapButton(tester, 'Join consultation');
  return h;
}

/// The device check passed with Join: in the room.
Future<Harness> inCall(WidgetTester tester, {void Function(Harness h)? before}) async {
  final h = await atCheck(tester, before: before);
  await tapButton(tester, 'Join');
  return h;
}

Future<void> emit(WidgetTester tester, Harness h, CallEvent event) async {
  h.media.room.emit(event);
  await tester.pumpAndSettle();
}

int joins(Harness h) => h.api.calls.where((c) => c == joinRoute).length;

void main() {
  group('device check', () {
    testWidgets('shows the camera and the microphone, then joins with both', (tester) async {
      final h = await atCheck(tester);

      expect(find.text('Check your camera and microphone'), findsOneWidget);
      expect(find.text('The practitioner will see and hear you once you join.'), findsOneWidget);
      expect(find.byKey(const Key('preview-camera')), findsOneWidget);
      expect(
        find.text(
          'To join the video consultation, Salus needs your camera and microphone. The consultation is not recorded.',
        ),
        findsNothing,
        reason: 'both were allowed before',
      );
      expect(h.media.asked, isEmpty);
      expect(h.media.rooms, isEmpty, reason: 'no token is asked for before Join');

      await tapButton(tester, 'Join');

      expect(h.api.calls.where((c) => c.startsWith('POST')), [joinRoute]);
      expect(h.media.room.url, 'ws://livekit.test');
      expect(h.media.room.token, 'synthetic-livekit-token');
      expect((h.media.room.settings.camera, h.media.room.settings.microphone), (true, true));
      expect(h.media.previews.every((p) => p.stopped), isTrue, reason: 'the room opens the devices itself');
      expectNoCsrfHeader(h.api);
    });

    testWidgets('on Android, explains why before the system prompt', (tester) async {
      final h = Harness(store: FakeTokenStore(token));
      h.media.state.updateAll((_, _) => Access.unasked);
      h.media.prompt = Completer<void>();
      h.api.answer('GET /auth/me', 200, account());
      answerDirectory(h.api);
      h.api.on(
        'GET /fhir/R4/Appointment',
        (_) async => fhirResponse(200, bundle([appointment('now', '2026-11-04T09:05:00Z', '2026-11-04T09:35:00Z')])),
      );
      h.api.answer('GET /me/consents', 200, {
        'consents': [consent('c1')],
      });
      await h.pump(tester);
      await tapOutlined(tester, 'My appointments');
      await tester.tap(find.widgetWithText(FilledButton, 'Join consultation'));
      await tester.pump();
      await tester.pump();

      expect(
        find.text(
          'To join the video consultation, Salus needs your camera and microphone. The consultation is not recorded.',
        ),
        findsOneWidget,
      );
      expect(h.media.asked, [Kind.camera], reason: 'the prompt comes once the explanation is on screen');

      h.media.prompt!.complete();
      await tester.pumpAndSettle();

      expect(h.media.asked, [Kind.camera, Kind.microphone]);
      expect(find.byKey(const Key('preview-camera')), findsOneWidget);
      expect(h.media.previews.map((p) => p.kind), containsAll(Kind.values));
    });

    testWidgets('on iOS, the system prompt carries the explanation', (tester) async {
      debugDefaultTargetPlatformOverride = TargetPlatform.iOS;
      final h = await atCheck(tester, before: (h) => h.media.state.updateAll((_, _) => Access.unasked));

      expect(
        find.text(
          'To join the video consultation, Salus needs your camera and microphone. The consultation is not recorded.',
        ),
        findsNothing,
      );
      expect(h.media.asked, [Kind.camera, Kind.microphone]);
      expect(find.byKey(const Key('preview-camera')), findsOneWidget);
      debugDefaultTargetPlatformOverride = null;
    });

    testWidgets('a denied camera says how to allow it, and joins with the microphone only', (tester) async {
      final h = await atCheck(tester, before: (h) => h.media.state[Kind.camera] = Access.refused);

      expect(
        find.text(
          'Salus does not have access to your camera. You can allow it in your device settings, or join without video.',
        ),
        findsOneWidget,
      );
      expect(find.byKey(const Key('preview-camera')), findsNothing);
      await tapOutlined(tester, 'Open settings');
      expect(h.media.settingsOpened, 1);
      final cameraOff = tester.widget<CheckboxListTile>(find.widgetWithText(CheckboxListTile, 'Join with camera off'));
      expect(cameraOff.value, isTrue);
      expect(cameraOff.onChanged, isNull);

      await tapButton(tester, 'Join');

      expect((h.media.room.settings.camera, h.media.room.settings.microphone), (false, true));
      final camera = tester.widget<IconButton>(find.widgetWithIcon(IconButton, Icons.videocam_off));
      expect(camera.onPressed, isNull, reason: 'there is no camera to turn on');
      expect(find.byTooltip('Turn camera on'), findsOneWidget);
    });

    testWidgets('a camera allowed in the settings opens when the app comes back', (tester) async {
      final h = await atCheck(tester, before: (h) => h.media.state[Kind.camera] = Access.refused);
      expect(find.byKey(const Key('preview-camera')), findsNothing);

      h.media.state[Kind.camera] = Access.allowed;
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
      await tester.pumpAndSettle();

      expect(find.byKey(const Key('preview-camera')), findsOneWidget);
      expect(find.textContaining('does not have access to your camera'), findsNothing);
      expect(h.media.asked, isEmpty, reason: 'coming back asks nothing');
    });

    testWidgets('a denied microphone says the practitioner cannot hear', (tester) async {
      final h = await atCheck(tester, before: (h) => h.media.state[Kind.microphone] = Access.refused);

      expect(
        find.text(
          'Salus does not have access to your microphone. You can allow it in your device settings. Without it, the '
          'practitioner cannot hear you.',
        ),
        findsOneWidget,
      );
      expect(find.widgetWithText(OutlinedButton, 'Open settings'), findsOneWidget);
      await tapButton(tester, 'Join');
      expect((h.media.room.settings.camera, h.media.room.settings.microphone), (true, false));
    });

    testWidgets('with neither allowed, still joins to see and hear the practitioner', (tester) async {
      final h = await atCheck(tester, before: (h) => h.media.state.updateAll((_, _) => Access.refused));

      expect(find.textContaining('does not have access to your camera'), findsOneWidget);
      expect(find.textContaining('does not have access to your microphone'), findsOneWidget);
      expect(find.widgetWithText(OutlinedButton, 'Open settings'), findsOneWidget);
      await tapButton(tester, 'Join');

      expect((h.media.room.settings.camera, h.media.room.settings.microphone), (false, false));
      expect(find.text('Waiting for the practitioner'), findsOneWidget);
    });

    testWidgets('names a missing camera and microphone', (tester) async {
      final h = await atCheck(tester, before: (h) => h.media.missing.addAll(Kind.values));

      expect(find.text('No camera found.'), findsOneWidget);
      expect(find.text('No microphone found.'), findsOneWidget);
      expect(find.widgetWithText(OutlinedButton, 'Open settings'), findsNothing);
      await tapButton(tester, 'Join');
      expect((h.media.room.settings.camera, h.media.room.settings.microphone), (false, false));
    });

    testWidgets('joins with the camera or the microphone off when asked', (tester) async {
      final h = await atCheck(tester);
      await tapFound(tester, find.widgetWithText(CheckboxListTile, 'Join with camera off'));
      await tapFound(tester, find.widgetWithText(CheckboxListTile, 'Join with microphone off'));
      await tapButton(tester, 'Join');

      expect((h.media.room.settings.camera, h.media.room.settings.microphone), (false, false));
      expect(find.byTooltip('Turn camera on'), findsOneWidget);
      expect(find.byTooltip('Unmute microphone'), findsOneWidget);
    });

    testWidgets('says when no sound has come from the microphone for 5 seconds', (tester) async {
      final h = await atCheck(tester);
      const noSound = 'No sound detected. Check that your microphone is on and not muted.';
      final microphone = h.media.preview(Kind.microphone);

      microphone.level(0.01);
      await tester.pump(const Duration(seconds: 4));
      expect(find.text(noSound), findsNothing);
      microphone.level(0.01);
      await tester.pump(const Duration(seconds: 1));
      expect(find.text(noSound), findsOneWidget);

      microphone.level(0.3);
      await tester.pump();
      expect(find.text(noSound), findsNothing);
      await tester.pump(const Duration(seconds: 4));
      expect(find.text(noSound), findsNothing);
      await tester.pump(const Duration(seconds: 1));
      expect(find.text(noSound), findsOneWidget, reason: 'silent again for 5 seconds');
    });

    testWidgets('does not take a level it cannot measure for silence', (tester) async {
      await atCheck(tester);
      await tester.pump(const Duration(seconds: 10));
      expect(find.text('No sound detected. Check that your microphone is on and not muted.'), findsNothing);
    });

    testWidgets('going back stops the camera and the microphone', (tester) async {
      final h = await atCheck(tester);
      await tapFound(tester, find.byType(BackButton));

      expect(h.media.previews.every((p) => p.stopped), isTrue);
      expect(find.text('My appointments'), findsOneWidget);
    });
  });

  group('joining', () {
    for (final (status, code, message) in [
      (409, 'join_window_not_open', 'This consultation opens 10 minutes before the start.'),
      (409, 'consultation_ended', 'This consultation has ended.'),
      (409, 'consent_required', 'Joining needs your consent for this clinic.'),
      (404, 'not_found', 'This consultation is not available.'),
      (500, 'internal error', 'Something went wrong. Try again.'),
    ]) {
      testWidgets('a refused join ($code) says why and connects nothing', (tester) async {
        final h = await atCheck(tester, before: (h) => h.api.answer(joinRoute, status, error(code)));
        await tapButton(tester, 'Join');

        expect(find.text(message), findsOneWidget);
        expect(find.text('Check your camera and microphone'), findsOneWidget);
        expect(h.media.rooms, isEmpty);
      });
    }

    testWidgets('a join after the practitioner did not come says so', (tester) async {
      await atCheck(tester, before: (h) => h.api.answer(joinRoute, 409, error('practitioner_did_not_join')));
      await tapButton(tester, 'Join');

      expect(find.text('The practitioner did not join'), findsOneWidget);
      expect(
        find.text(
          'The time to join this consultation has passed. The clinic can see that you joined. The clinic may contact '
          'you, or you can book a new appointment.',
        ),
        findsOneWidget,
      );
    });

    testWidgets('a connection to LiveKit that fails says so', (tester) async {
      final h = await atCheck(tester, before: (h) => h.media.connectFails = true);
      await tapButton(tester, 'Join');

      expect(
        find.text('The video connection could not be started. Check your connection and try again.'),
        findsOneWidget,
      );
      expect(find.widgetWithText(FilledButton, 'Join'), findsOneWidget);
      expect(joins(h), 1);
    });

    testWidgets('shows Joining… while the token is asked for', (tester) async {
      final answer = Completer<http.Response>();
      await atCheck(tester, before: (h) => h.api.on(joinRoute, (_) => answer.future));
      await tester.ensureVisible(find.widgetWithText(FilledButton, 'Join'));
      await tester.tap(find.widgetWithText(FilledButton, 'Join'));
      await tester.pump();

      final button = tester.widget<FilledButton>(find.widgetWithText(FilledButton, 'Joining…'));
      expect(button.onPressed, isNull);
      answer.complete(jsonResponse(200, joined()));
      await tester.pumpAndSettle();
      expect(find.text('Waiting for the practitioner'), findsOneWidget);
    });
  });

  group('call', () {
    testWidgets('waits for the practitioner until they are in the room', (tester) async {
      final h = await inCall(tester);

      expect(find.text('Waiting for the practitioner'), findsOneWidget);
      expect(find.text('Cy Example will join you here. Keep this screen open.'), findsOneWidget);
      expect(find.byKey(const Key('own-video')), findsOneWidget);

      await emit(tester, h, const PeerChanged(practitioner));

      expect(find.text('Waiting for the practitioner'), findsNothing);
      expect(find.byKey(const Key('remote-video')), findsOneWidget);
    });

    testWidgets('leaves out the lines that need a name when there is none', (tester) async {
      final h = await inCall(tester, before: (h) => h.api.answer(joinRoute, 200, joined(peerName: '')));

      expect(find.text('Waiting for the practitioner'), findsOneWidget);
      expect(find.textContaining('will join you here'), findsNothing);
      await emit(tester, h, const PeerChanged(Peer(present: true, camera: false, microphone: false)));
      expect(find.textContaining('camera is off'), findsNothing);
      expect(find.textContaining('is muted'), findsNothing);
    });

    testWidgets("shows the practitioner's camera off and muted", (tester) async {
      final h = await inCall(tester);
      await emit(tester, h, const PeerChanged(Peer(present: true, camera: false, microphone: false)));

      expect(find.text("Cy Example's camera is off"), findsOneWidget);
      expect(find.text('Cy Example is muted'), findsOneWidget);
      expect(find.byKey(const Key('remote-video')), findsNothing);

      await emit(tester, h, const PeerChanged(practitioner));
      expect(find.text("Cy Example's camera is off"), findsNothing);
      expect(find.text('Cy Example is muted'), findsNothing);
    });

    testWidgets('says when the practitioner has left', (tester) async {
      final h = await inCall(tester);
      await emit(tester, h, const PeerChanged(practitioner));
      await emit(tester, h, const PeerChanged(nobody));

      expect(
        find.text('Cy Example has left the consultation. They can join again until the consultation ends.'),
        findsOneWidget,
      );
      expect(find.text('Waiting for the practitioner'), findsNothing);
    });

    testWidgets('mutes, unmutes and turns the camera off and on', (tester) async {
      final h = await inCall(tester);

      await tapFound(tester, find.byTooltip('Mute microphone'));
      expect(h.media.room.microphone, isFalse);
      await tapFound(tester, find.byTooltip('Unmute microphone'));
      expect(h.media.room.microphone, isTrue);
      await tapFound(tester, find.byTooltip('Turn camera off'));
      expect(h.media.room.camera, isFalse);
      expect(find.byKey(const Key('own-video')), findsNothing);
      await tapFound(tester, find.byTooltip('Turn camera on'));
      expect(h.media.room.camera, isTrue);
    });

    testWidgets('leaves the room and reads the appointments again', (tester) async {
      final h = await inCall(tester);
      final searches = h.api.calls.where((c) => c == 'GET /fhir/R4/Appointment').length;

      await tapButton(tester, 'Leave');

      expect(h.media.room.left, isTrue);
      expect(find.text('My appointments'), findsOneWidget);
      expect(h.api.calls.where((c) => c == 'GET /fhir/R4/Appointment').length, searches + 1);

      h.media.room.emit(const Disconnected(Ending.ended));
      await tester.pumpAndSettle();
      expect(h.api.calls, isNot(contains(statusRoute)), reason: 'a left room is not followed');
    });

    testWidgets('going back leaves the room', (tester) async {
      final h = await inCall(tester);
      await tapFound(tester, find.byType(BackButton));
      expect(h.media.room.left, isTrue);
    });

    testWidgets('shows reconnecting, then reconnected for a few seconds', (tester) async {
      final h = await inCall(tester);
      await emit(tester, h, const PeerChanged(practitioner));
      await emit(tester, h, const Reconnecting());

      expect(find.text('Connection lost. Reconnecting…'), findsOneWidget);

      await emit(tester, h, const Reconnected());
      expect(find.text('Connection lost. Reconnecting…'), findsNothing);
      expect(find.text('Reconnected'), findsOneWidget);
      await tester.pump(const Duration(seconds: 4));
      expect(find.text('Reconnected'), findsNothing);
    });

    testWidgets('a lost connection asks for a new token and rejoins with the same settings', (tester) async {
      final h = await inCall(tester);
      await tapFound(tester, find.byTooltip('Mute microphone'));

      await emit(tester, h, const Disconnected(Ending.lost));

      expect(joins(h), 2);
      expect(h.media.rooms, hasLength(2));
      expect((h.media.room.settings.camera, h.media.room.settings.microphone), (true, false));
      expect(find.text('Reconnected'), findsOneWidget);
    });

    testWidgets('a rejoin that fails offers to try again', (tester) async {
      final h = await inCall(tester);
      h.api.on(joinRoute, (_) => throw http.ClientException('offline'));

      await emit(tester, h, const Disconnected(Ending.lost));

      expect(find.text('The connection could not be restored.'), findsOneWidget);
      h.api.answer(joinRoute, 200, joined());
      await tapButton(tester, 'Try again');

      expect(h.media.rooms, hasLength(2));
      expect(find.text('Waiting for the practitioner'), findsOneWidget);
    });

    testWidgets('joining from another device goes back to the device check', (tester) async {
      final h = await inCall(tester);
      await emit(tester, h, const Disconnected(Ending.replaced));

      expect(find.text('Check your camera and microphone'), findsOneWidget);
      expect(joins(h), 1, reason: 'the two devices do not take turns');
    });
  });

  group('end', () {
    for (final (reason, title, line) in [
      ('practitioner', 'The consultation has ended', 'You can close this screen.'),
      ('window_closed', 'The consultation has ended', 'You can close this screen.'),
      ('time_limit', 'The consultation has ended', 'The consultation reached its time limit and has ended.'),
      ('practitioner_absent', 'The practitioner did not join', 'The time to join this consultation has passed.'),
      ('cancelled', null, 'This appointment was cancelled, so the consultation has ended.'),
    ]) {
      testWidgets('a deleted room ended $reason says so', (tester) async {
        final h = await inCall(tester);
        h.api.answer(statusRoute, 200, status('ended', endReason: reason));

        await emit(tester, h, const Disconnected(Ending.ended));

        if (title != null) expect(find.text(title), findsOneWidget);
        expect(find.textContaining(line), findsOneWidget);
        expect(joins(h), 1);
        expect(find.byTooltip('Mute microphone'), findsNothing);
      });
    }

    testWidgets('a room deleted while the consultation is open is joined again', (tester) async {
      final h = await inCall(tester);
      await emit(tester, h, const Disconnected(Ending.ended));

      expect(h.media.rooms, hasLength(2));
      expect(find.text('Waiting for the practitioner'), findsOneWidget);
    });

    // Why the patient was removed comes from the status's reason alone; the join endpoint is not asked.
    testWidgets('removed after withdrawing consent says so', (tester) async {
      final h = await inCall(tester);
      await emit(tester, h, const PeerChanged(practitioner));
      h.api.answer(statusRoute, 200, status('open', reason: 'consent_withdrawn'));

      await emit(tester, h, const Disconnected(Ending.removed));

      expect(
        find.text('You withdrew your consent for this clinic, so you have left the consultation.'),
        findsOneWidget,
      );
      expect(h.media.rooms, hasLength(1));
      expect(joins(h), 1, reason: 'no join request to learn why');
    });

    testWidgets('says nothing about the practitioner while it asks why it was removed', (tester) async {
      final h = await inCall(tester);
      await emit(tester, h, const PeerChanged(practitioner));
      final answer = Completer<http.Response>();
      h.api.on(statusRoute, (_) => answer.future);

      await emit(tester, h, const Disconnected(Ending.removed));

      expect(find.textContaining('has left the consultation'), findsNothing);
      expect(find.text('Waiting for the practitioner'), findsNothing);
      answer.complete(jsonResponse(200, status('open', reason: 'consent_withdrawn')));
      await tester.pumpAndSettle();
      expect(find.textContaining('You withdrew your consent'), findsOneWidget);
    });

    testWidgets('removed after a cancellation says so', (tester) async {
      final h = await inCall(tester);
      h.api.answer(statusRoute, 200, status('ended', endReason: 'cancelled', reason: 'cancelled'));

      await emit(tester, h, const Disconnected(Ending.removed));

      expect(find.text('This appointment was cancelled, so the consultation has ended.'), findsOneWidget);
      expect(joins(h), 1, reason: 'no join request to learn why');
    });

    testWidgets('a deleted room says the consent was withdrawn when a revocation before the start cancelled it', (
      tester,
    ) async {
      final h = await inCall(tester);
      h.api.answer(statusRoute, 200, status('ended', endReason: 'cancelled', reason: 'consent_withdrawn'));

      await emit(tester, h, const Disconnected(Ending.ended));

      expect(find.textContaining('You withdrew your consent'), findsOneWidget);
      expect(joins(h), 1);
    });

    testWidgets('removed while access is still there joins again', (tester) async {
      final h = await inCall(tester);
      h.api.answer(statusRoute, 200, status('open', reason: 'left'));

      await emit(tester, h, const Disconnected(Ending.removed));

      expect(h.media.rooms, hasLength(2));
    });

    testWidgets('a rejoin refused as ended shows the end', (tester) async {
      final h = await inCall(tester);
      h.api.answer(joinRoute, 409, error('consultation_ended'));
      h.api.answer(statusRoute, 200, status('ended', endReason: 'practitioner'));

      await emit(tester, h, const Disconnected(Ending.lost));

      expect(find.text('The consultation has ended'), findsOneWidget);
      expect(find.text('You can close this screen.'), findsOneWidget);
    });

    testWidgets('a rejoin refused as ended while the status is still open does not loop', (tester) async {
      final h = await inCall(tester);
      h.api.answer(joinRoute, 409, error('consultation_ended'));

      await emit(tester, h, const Disconnected(Ending.lost));

      expect(find.text('The consultation has ended'), findsOneWidget);
      expect(joins(h), 2);
    });

    testWidgets('the window closing while waiting says the practitioner did not join', (tester) async {
      final h = await inCall(tester);
      h.api.answer(statusRoute, 200, status('ended', endReason: 'practitioner_absent'));

      await emit(tester, h, const Disconnected(Ending.ended));

      expect(find.text('The practitioner did not join'), findsOneWidget);
    });
  });

  testWidgets('every consultation screen fits a phone 390 px wide without overflowing', (tester) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final h = await atCheck(
      tester,
      before: (h) => h.media.state
        ..[Kind.camera] = Access.refused
        ..[Kind.microphone] = Access.refused,
    );
    expect(tester.takeException(), isNull, reason: 'device check');

    await tapButton(tester, 'Join');
    await emit(tester, h, const PeerChanged(Peer(present: true, camera: false, microphone: false)));
    await emit(tester, h, const Reconnecting());
    await emit(tester, h, const Reconnected());
    expect(tester.takeException(), isNull, reason: 'call');

    h.api.answer(statusRoute, 200, status('ended', endReason: 'practitioner_absent'));
    await emit(tester, h, const Disconnected(Ending.ended));
    expect(find.text('The practitioner did not join'), findsOneWidget);
    expect(tester.takeException(), isNull, reason: 'end');
  });

  testWidgets('prints nothing through a consultation', (tester) async {
    final printed = <String?>[];
    final original = debugPrint;
    debugPrint = (String? message, {int? wrapWidth}) => printed.add(message);
    try {
      final h = await inCall(tester);
      await emit(tester, h, const PeerChanged(practitioner));
      await emit(tester, h, const Disconnected(Ending.lost));
      h.api.answer(statusRoute, 200, status('ended', endReason: 'practitioner'));
      await emit(tester, h, const Disconnected(Ending.ended));
      expect(find.text('The consultation has ended'), findsOneWidget);
    } finally {
      debugPrint = original;
    }
    expect(printed, isEmpty);
  });
}
