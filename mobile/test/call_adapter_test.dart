import 'package:flutter_test/flutter_test.dart';
import 'package:livekit_client/livekit_client.dart' as lk;
import 'package:permission_handler/permission_handler.dart' as ph;
import 'package:salus/call.dart';

// The mappings in call.dart from LiveKit's and the system's values to what the screens act on.
void main() {
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
}
