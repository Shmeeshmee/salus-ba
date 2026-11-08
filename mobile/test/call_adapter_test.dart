import 'package:flutter_test/flutter_test.dart';
import 'package:livekit_client/livekit_client.dart' as lk;
import 'package:logging/logging.dart';
import 'package:permission_handler/permission_handler.dart' as ph;
import 'package:salus/call.dart';

// The mappings in call.dart from LiveKit's and the system's values to what the screens act on.
void main() {
  // livekit_client logs its connection steps and errors, the signalling URL among them, which carries the token
  // (SPEC.md "Changed after package 6"); LiveKitMedia turns its log off, so nothing reaches a listener.
  test("LiveKit's own log is off", () {
    final records = <LogRecord>[];
    final listening = Logger.root.onRecord.listen(records.add);
    addTearDown(listening.cancel);

    LiveKitMedia();
    Logger('livekit').warning('could not connect to ws://livekit.test/rtc?access_token=synthetic-token');

    expect(lk.getLoggingLevel(), Level.OFF);
    expect(records, isEmpty);
  });

  test("LiveKit's disconnect reasons", () {
    expect(endingOf(lk.DisconnectReason.clientInitiated), Ending.left);
    expect(endingOf(lk.DisconnectReason.duplicateIdentity), Ending.replaced);
    expect(endingOf(lk.DisconnectReason.participantRemoved), Ending.removed);
    expect(endingOf(lk.DisconnectReason.roomDeleted), Ending.ended);
    expect(endingOf(lk.DisconnectReason.roomClosed), Ending.ended);
    for (final lost in [
      null,
      lk.DisconnectReason.unknown,
      lk.DisconnectReason.serverShutdown,
      lk.DisconnectReason.signalClose,
      lk.DisconnectReason.reconnectAttemptsExceeded,
      lk.DisconnectReason.joinFailure,
    ]) {
      expect(endingOf(lost), Ending.lost, reason: '$lost');
    }
  });

  test("the system's permission states", () {
    expect(accessOf(ph.PermissionStatus.granted), Access.allowed);
    expect(accessOf(ph.PermissionStatus.limited), Access.allowed);
    expect(accessOf(ph.PermissionStatus.denied), Access.unasked, reason: 'iOS not asked, or Android asked once');
    expect(accessOf(ph.PermissionStatus.permanentlyDenied), Access.refused);
    expect(accessOf(ph.PermissionStatus.restricted), Access.refused);
  });

  test("the microphone's level is the loudest band, from 0 to 1", () {
    expect(levelOf(const []), 0);
    expect(levelOf(const [0.1, 0.4, 0.2]), 0.4);
    expect(levelOf(const [1.5]), 1);
    expect(levelOf(const [-0.3, 'x', null]), 0);
  });

  // The other person is never the agent, whatever the order LiveKit lists them in (SPEC.md "Changed for SAL-3").
  test('the peer is the other person, never the agent', () {
    const agent = Remote(agent: true, camera: false, microphone: true);
    const practitioner = Remote(agent: false, camera: true, microphone: false);

    for (final remotes in [
      [agent, practitioner],
      [practitioner, agent],
    ]) {
      final peer = peerOf(remotes);
      expect((peer.present, peer.camera, peer.microphone), (true, true, false));
    }
    expect(peerOf([agent]).present, isFalse);
    expect(peerOf(<Remote>[]).present, isFalse);
  });

  test('whether the agent is in the room', () {
    expect(
      agentIn([
        const Remote(agent: false, camera: true, microphone: true),
        const Remote(agent: true, camera: false, microphone: false),
      ]),
      isTrue,
    );
    expect(agentIn([const Remote(agent: false, camera: true, microphone: true)]), isFalse);
    expect(agentIn(<Remote>[]), isFalse);
  });
}
