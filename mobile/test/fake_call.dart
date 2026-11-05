import 'dart:async';

import 'package:flutter/widgets.dart';
import 'package:salus/call.dart';

/// A camera or microphone opened for the check; a test sends the microphone's levels.
class FakePreview implements Preview {
  FakePreview(this.kind);

  final Kind kind;
  final _levels = StreamController<double>.broadcast();
  var stopped = false;

  void level(double value) => _levels.add(value);

  @override
  Widget view() => SizedBox(key: Key('preview-${kind.name}'));

  @override
  Stream<double> get levels => _levels.stream;

  @override
  Future<void> stop() async => stopped = true;
}

/// A connection to a room; a test sends LiveKit's events through [emit].
class FakeRoom implements Room {
  FakeRoom(this.url, this.token, this.settings, this._onEvent)
    : camera = settings.camera,
      microphone = settings.microphone;

  final String url;
  final String token;
  final Settings settings;
  final void Function(CallEvent event) _onEvent;
  bool camera;
  bool microphone;
  var left = false;

  void emit(CallEvent event) => _onEvent(event);

  @override
  Widget remoteView() => const SizedBox(key: Key('remote-video'));

  @override
  Widget localView() => const SizedBox(key: Key('own-video'));

  @override
  Future<bool> setCamera(bool on) async => camera = on;

  @override
  Future<bool> setMicrophone(bool on) async => microphone = on;

  @override
  Future<void> leave() async => left = true;
}

/// The device and LiveKit as the screens see them: what the app may use, what the system prompt answers, which
/// devices are missing, and every room connected.
class FakeMedia implements Media {
  /// Whether the app may use each device now.
  final state = {Kind.camera: Access.allowed, Kind.microphone: Access.allowed};

  /// What the system prompt answers for a device not asked yet.
  final answers = {Kind.camera: Access.allowed, Kind.microphone: Access.allowed};

  /// Holds the system prompt open until the test completes it.
  Completer<void>? prompt;
  final asked = <Kind>[];
  final missing = <Kind>{};
  final previews = <FakePreview>[];
  final rooms = <FakeRoom>[];
  var settingsOpened = 0;
  var connectFails = false;

  FakeRoom get room => rooms.last;

  FakePreview preview(Kind kind) => previews.lastWhere((p) => p.kind == kind);

  @override
  Future<Access> access(Kind kind) async => state[kind]!;

  @override
  Future<Access> ask(Kind kind) async {
    asked.add(kind);
    await prompt?.future;
    if (state[kind] == Access.unasked) state[kind] = answers[kind]!;
    return state[kind]!;
  }

  @override
  Future<void> openSettings() async => settingsOpened++;

  @override
  Future<Opened> open(Kind kind) async {
    if (missing.contains(kind)) return const Opened.problem(Problem.missing);
    final preview = FakePreview(kind);
    previews.add(preview);
    return Opened.preview(preview);
  }

  @override
  Future<Room> connect(String url, String token, Settings settings, void Function(CallEvent event) onEvent) async {
    if (connectFails) throw const CallFailed();
    final room = FakeRoom(url, token, settings, onEvent);
    rooms.add(room);
    return room;
  }
}
