import 'dart:async';

import 'package:flutter/widgets.dart';
import 'package:livekit_client/livekit_client.dart' as lk;
import 'package:permission_handler/permission_handler.dart' as ph;

// call.dart is the one file that reaches LiveKit (livekit_client) and the device's camera, microphone and
// permissions (permission_handler). The screens see only the types below, so they are tested with a fake
// (test/fake_call.dart). Nothing here logs, and nothing about the people in the call goes to LiveKit: the token
// from our API is all it gets (docs/consultation/SPEC.md "Opaque to LiveKit").

enum Kind { camera, microphone }

/// Whether the app may use a device: allowed, not asked yet (the system prompt can still be shown), or refused
/// (only the device settings can change it).
enum Access { allowed, unasked, refused }

/// Why a device the app may not or cannot use is off: access refused, or no such device.
enum Problem { denied, missing }

/// A camera or microphone opened for the check before joining: the camera's picture, the microphone's levels
/// from 0 to 1 (none while they cannot be measured).
abstract interface class Preview {
  Widget view();
  Stream<double> get levels;
  Future<void> stop();
}

/// A device opened for the check, or why it is off.
class Opened {
  const Opened.preview(Preview this.preview) : problem = null;
  const Opened.problem(Problem this.problem) : preview = null;

  final Preview? preview;
  final Problem? problem;
}

/// The other person in the room as LiveKit shows them: present, and whether their camera and microphone are on.
class Peer {
  const Peer({required this.present, required this.camera, required this.microphone});

  final bool present;
  final bool camera;
  final bool microphone;
}

/// Why the connection to the room ended: this side left, this side was removed from the room, the room was
/// deleted or closed, the same person joined from another device, or the network.
enum Ending { left, removed, ended, replaced, lost }

sealed class CallEvent {
  const CallEvent();
}

final class PeerChanged extends CallEvent {
  const PeerChanged(this.peer);
  final Peer peer;
}

final class Reconnecting extends CallEvent {
  const Reconnecting();
}

final class Reconnected extends CallEvent {
  const Reconnected();
}

final class Disconnected extends CallEvent {
  const Disconnected(this.ending);
  final Ending ending;
}

/// The connection to LiveKit could not be made. It carries nothing from LiveKit's error.
class CallFailed implements Exception {
  const CallFailed();
}

/// What to publish on joining.
class Settings {
  const Settings({required this.camera, required this.microphone});

  final bool camera;
  final bool microphone;
}

/// A joined room.
abstract interface class Room {
  /// The other person's camera, and this side's own; read again on every [PeerChanged].
  Widget remoteView();
  Widget localView();

  /// Turn this side's camera or microphone on or off; the answer is whether it is on.
  Future<bool> setCamera(bool on);
  Future<bool> setMicrophone(bool on);
  Future<void> leave();
}

abstract interface class Media {
  /// Whether the app may use [kind], without asking.
  Future<Access> access(Kind kind);

  /// Shows the system prompt for [kind] when it has not been asked; answers what the app may now do.
  Future<Access> ask(Kind kind);

  /// The app's page in the device settings, where a refused device can be allowed.
  Future<void> openSettings();

  Future<Opened> open(Kind kind);

  /// Joins the room at [url] with [token]; throws [CallFailed].
  Future<Room> connect(String url, String token, Settings settings, void Function(CallEvent event) onEvent);
}

/// Media on livekit_client and permission_handler. No LiveKit UI components; the screens draw their own.
class LiveKitMedia implements Media {
  LiveKitMedia() {
    // LiveKit's own log could carry the server URL with the token; it stays off.
    lk.disableLogging();
  }

  static ph.Permission _permission(Kind kind) => kind == Kind.camera ? ph.Permission.camera : ph.Permission.microphone;

  @override
  Future<Access> access(Kind kind) async => accessOf(await _permission(kind).status);

  @override
  Future<Access> ask(Kind kind) async {
    final status = await _permission(kind).request();
    return accessOf(status) == Access.allowed ? Access.allowed : Access.refused;
  }

  @override
  Future<void> openSettings() => ph.openAppSettings();

  @override
  Future<Opened> open(Kind kind) async {
    try {
      if (kind == Kind.camera) return Opened.preview(_CameraPreview(await lk.LocalVideoTrack.createCameraTrack()));
      final track = await lk.LocalAudioTrack.create();
      return Opened.preview(await _MicrophonePreview.start(track));
    } on Object {
      return const Opened.problem(Problem.missing);
    }
  }

  @override
  Future<Room> connect(String url, String token, Settings settings, void Function(CallEvent event) onEvent) async {
    final room = lk.Room(roomOptions: const lk.RoomOptions(adaptiveStream: true, dynacast: true));
    final listener = room.createListener();
    var connected = false;
    void peer() => onEvent(PeerChanged(peerOf(room)));
    listener
      ..on<lk.ParticipantConnectedEvent>((_) => peer())
      ..on<lk.ParticipantDisconnectedEvent>((_) => peer())
      ..on<lk.TrackSubscribedEvent>((_) => peer())
      ..on<lk.TrackUnsubscribedEvent>((_) => peer())
      ..on<lk.TrackMutedEvent>((_) => peer())
      ..on<lk.TrackUnmutedEvent>((_) => peer())
      ..on<lk.LocalTrackPublishedEvent>((_) => peer())
      ..on<lk.RoomReconnectingEvent>((_) => onEvent(const Reconnecting()))
      ..on<lk.RoomReconnectedEvent>((_) => onEvent(const Reconnected()))
      ..on<lk.RoomDisconnectedEvent>((e) {
        if (connected) onEvent(Disconnected(endingOf(e.reason)));
        unawaited(listener.dispose());
        unawaited(room.dispose());
      });
    try {
      await room.connect(url, token);
    } on Object {
      await listener.dispose();
      await room.dispose();
      throw const CallFailed();
    }
    connected = true;
    // A device that fails now stays off; the call goes on with what remains.
    if (settings.microphone) await _quietly(() => room.localParticipant?.setMicrophoneEnabled(true));
    if (settings.camera) await _quietly(() => room.localParticipant?.setCameraEnabled(true));
    peer();
    return _LiveKitRoom(room);
  }
}

class _LiveKitRoom implements Room {
  _LiveKitRoom(this._room);

  final lk.Room _room;

  @override
  Widget remoteView() => _view(_room.remoteParticipants.values.firstOrNull);

  @override
  Widget localView() => _view(_room.localParticipant);

  static Widget _view(lk.Participant? p) {
    final track = p?.getTrackPublicationBySource(lk.TrackSource.camera)?.track;
    return track is lk.VideoTrack ? lk.VideoTrackRenderer(track, fit: lk.VideoViewFit.cover) : const SizedBox.shrink();
  }

  @override
  Future<bool> setCamera(bool on) async {
    await _quietly(() => _room.localParticipant?.setCameraEnabled(on));
    return _room.localParticipant?.isCameraEnabled() ?? false;
  }

  @override
  Future<bool> setMicrophone(bool on) async {
    await _quietly(() => _room.localParticipant?.setMicrophoneEnabled(on));
    return _room.localParticipant?.isMicrophoneEnabled() ?? false;
  }

  @override
  Future<void> leave() => _room.disconnect();
}

class _CameraPreview implements Preview {
  _CameraPreview(this._track);

  final lk.LocalVideoTrack _track;

  @override
  Widget view() => lk.VideoTrackRenderer(_track, fit: lk.VideoViewFit.cover);

  @override
  Stream<double> get levels => const Stream.empty();

  @override
  Future<void> stop() => _track.stop();
}

/// The microphone and LiveKit's audio visualizer: the loudest band of each reading is the level.
class _MicrophonePreview implements Preview {
  _MicrophonePreview(this._track, this._visualizer);

  final lk.LocalAudioTrack _track;
  final lk.AudioVisualizer _visualizer;
  final _levels = StreamController<double>.broadcast();

  static Future<_MicrophonePreview> start(lk.LocalAudioTrack track) async {
    final visualizer = lk.createVisualizer(track);
    final preview = _MicrophonePreview(track, visualizer);
    visualizer.events.on<lk.AudioVisualizerEvent>((e) => preview._levels.add(levelOf(e.event)));
    await _quietly(visualizer.start);
    return preview;
  }

  @override
  Widget view() => const SizedBox.shrink();

  @override
  Stream<double> get levels => _levels.stream;

  @override
  Future<void> stop() async {
    if (_levels.isClosed) return;
    await _levels.close();
    await _quietly(_visualizer.stop);
    await _visualizer.dispose();
    await _track.stop();
  }
}

Future<void> _quietly(Future<Object?>? Function() call) async {
  try {
    await call();
  } on Object {
    // The device stays as it was; the screen shows what it is.
  }
}

/// The other participant: the room holds at most two (SPEC §2 "Rooms").
Peer peerOf(lk.Room room) {
  final p = room.remoteParticipants.values.firstOrNull;
  if (p == null) return const Peer(present: false, camera: false, microphone: false);
  bool on(lk.TrackSource source) {
    final publication = p.getTrackPublicationBySource(source);
    return publication != null && !publication.muted;
  }

  return Peer(present: true, camera: on(lk.TrackSource.camera), microphone: on(lk.TrackSource.microphone));
}

/// What the app may do with a device, from the system's answer. On Android a single refusal can still be asked
/// again, so it is [Access.unasked]; on iOS a refusal is final.
Access accessOf(ph.PermissionStatus status) => switch (status) {
  ph.PermissionStatus.granted || ph.PermissionStatus.limited || ph.PermissionStatus.provisional => Access.allowed,
  ph.PermissionStatus.denied => Access.unasked,
  ph.PermissionStatus.permanentlyDenied || ph.PermissionStatus.restricted => Access.refused,
};

/// What the screen does about LiveKit's reason for a disconnection.
Ending endingOf(lk.DisconnectReason? reason) => switch (reason) {
  lk.DisconnectReason.clientInitiated => Ending.left,
  lk.DisconnectReason.duplicateIdentity => Ending.replaced,
  lk.DisconnectReason.participantRemoved => Ending.removed,
  lk.DisconnectReason.roomDeleted || lk.DisconnectReason.roomClosed => Ending.ended,
  _ => Ending.lost,
};

/// The loudest band of an audio visualizer reading, from 0 to 1.
double levelOf(List<Object?> bands) {
  var level = 0.0;
  for (final b in bands) {
    if (b is num && b > level) level = b.toDouble();
  }
  return level.clamp(0, 1).toDouble();
}
