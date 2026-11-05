import 'dart:async';

import 'package:flutter/material.dart';
import 'package:salus_api/api.dart' as api;

import 'booking.dart';
import 'call.dart';
import 'device_check.dart';
import 'messages.dart';
import 'session.dart';
import 'ui.dart' as ui;

enum _Phase { check, joining, call, reconnecting, lost, ended }

const _nobody = Peer(present: false, camera: false, microphone: false);
const _ink = Color(0xFF0E1512);

/// The patient's video consultation (docs/consultation/SPEC.md "Client states"): the device check, joining,
/// waiting for the practitioner, the call, reconnecting, and the end. The token comes from our API on every join
/// and rejoin; the server decides whether one is issued. Why the patient's connection ended comes from LiveKit's
/// reason with our API's answer: removed from the room, the join endpoint says whether the consent was withdrawn
/// or the appointment cancelled; the room deleted, the consultation's status says how it ended. Events of a room
/// this screen has left are ignored (the generation), so a late event of an old connection changes nothing.
class CallScreen extends StatefulWidget {
  const CallScreen({super.key, required this.booking, required this.media, required this.appointmentId});

  final Booking booking;
  final Media media;
  final String appointmentId;

  @override
  State<CallScreen> createState() => _CallScreenState();
}

class _CallScreenState extends State<CallScreen> {
  var _phase = _Phase.check;
  String? _error;
  ({String? title, String line})? _end;
  Room? _room;
  var _peer = _nobody;
  var _peerName = '';
  var _seen = false;
  var _settings = const Settings(camera: false, microphone: false);
  var _cameraUsable = false;
  var _reconnected = false;
  Timer? _reconnectedTimer;
  var _generation = 0;

  @override
  void dispose() {
    _generation++;
    _room?.leave();
    _reconnectedTimer?.cancel();
    super.dispose();
  }

  bool _current(int generation) => mounted && generation == _generation;

  void _join(Settings settings, bool cameraUsable) {
    setState(() {
      _phase = _Phase.joining;
      _error = null;
      _settings = settings;
      _cameraUsable = cameraUsable;
    });
    _connect(again: false);
  }

  /// Asks for a token and joins the room; [again] is a rejoin of a connection that ended, [removed] after LiveKit
  /// removed the patient.
  Future<void> _connect({required bool again, bool removed = false}) async {
    final gen = ++_generation;
    final api.ConsultationJoin join;
    try {
      join = await widget.booking.join(widget.appointmentId);
    } on ApiFailure catch (failure) {
      if (_current(gen)) _refused(failure, again: again, removed: removed);
      return;
    }
    if (!_current(gen)) return;
    final Room room;
    try {
      room = await widget.media.connect(join.url, join.token, _settings, (e) {
        if (_current(gen)) _onEvent(e);
      });
    } on CallFailed {
      if (!_current(gen)) return;
      return setState(() {
        _phase = again ? _Phase.lost : _Phase.check;
        _error = again ? null : cannotConnect;
      });
    }
    if (!_current(gen)) {
      unawaited(room.leave());
      return;
    }
    setState(() {
      _room = room;
      _peerName = join.peerName;
      _phase = _Phase.call;
    });
    if (again) _flashReconnected();
  }

  void _refused(ApiFailure failure, {required bool again, required bool removed}) {
    switch (failure.code) {
      case 'practitioner_did_not_join':
        return _ended(practitionerAbsent, timePassed);
      case 'consent_required' when again:
        return _ended(null, withdrewConsent);
      case 'consultation_ended' when removed:
        return _ended(null, appointmentWasCancelled);
      case 'consultation_ended' when again:
        _finish(rejoin: false);
        return;
    }
    if (again && failure.unreachable) return setState(() => _phase = _Phase.lost);
    setState(() {
      _phase = _Phase.check;
      _error = joinMessage(failure);
    });
  }

  /// Shows how the consultation ended once its room is gone; a room gone while the consultation is still open is
  /// joined again ([rejoin]).
  Future<void> _finish({bool rejoin = true}) async {
    final gen = ++_generation;
    api.ConsultationStatus? status;
    try {
      status = await widget.booking.consultation(widget.appointmentId);
    } on ApiFailure {
      status = null;
    }
    if (!_current(gen)) return;
    if (status?.state != api.ConsultationStatusStateEnum.ended) {
      return rejoin ? _connect(again: true) : _ended(consultationOver, closeScreen);
    }
    switch (status?.endReason) {
      case api.ConsultationStatusEndReasonEnum.practitionerAbsent:
        _ended(practitionerAbsent, timePassed);
      case api.ConsultationStatusEndReasonEnum.timeLimit:
        _ended(consultationOver, timeLimit);
      case api.ConsultationStatusEndReasonEnum.cancelled:
        _ended(null, appointmentWasCancelled);
      default:
        _ended(consultationOver, closeScreen);
    }
  }

  void _ended(String? title, String line) {
    _generation++;
    setState(() {
      _room = null;
      _phase = _Phase.ended;
      _end = (title: title, line: line);
    });
  }

  void _onEvent(CallEvent event) {
    switch (event) {
      case PeerChanged(:final peer):
        setState(() {
          _peer = peer;
          if (peer.present) _seen = true;
        });
      case Reconnecting():
        setState(() {
          _phase = _Phase.reconnecting;
          _reconnected = false;
        });
      case Reconnected():
        setState(() => _phase = _Phase.call);
        _flashReconnected();
      case Disconnected(:final ending):
        _disconnected(ending);
    }
  }

  void _disconnected(Ending ending) {
    if (ending == Ending.left) return;
    setState(() {
      _room = null;
      _peer = _nobody;
    });
    switch (ending) {
      case Ending.replaced:
        _generation++;
        setState(() => _phase = _Phase.check);
      case Ending.removed:
        _connect(again: true, removed: true);
      case Ending.ended:
        _finish();
      case Ending.lost || Ending.left:
        setState(() => _phase = _Phase.reconnecting);
        _connect(again: true);
    }
  }

  void _flashReconnected() {
    _reconnectedTimer?.cancel();
    setState(() => _reconnected = true);
    _reconnectedTimer = Timer(const Duration(seconds: 4), () {
      if (mounted) setState(() => _reconnected = false);
    });
  }

  Future<void> _toggle(Kind kind) async {
    final room = _room;
    if (room == null) return;
    final camera = kind == Kind.camera;
    final on = camera ? await room.setCamera(!_settings.camera) : await room.setMicrophone(!_settings.microphone);
    if (!mounted) return;
    setState(
      () =>
          _settings = Settings(camera: camera ? on : _settings.camera, microphone: camera ? _settings.microphone : on),
    );
  }

  @override
  Widget build(BuildContext context) {
    switch (_phase) {
      case _Phase.check || _Phase.joining:
        return DeviceCheck(media: widget.media, busy: _phase == _Phase.joining, error: _error, onJoin: _join);
      case _Phase.lost:
        return ui.Page(
          title: null,
          children: [
            const ui.Message(notRestored, error: true),
            ui.SubmitButton(
              tryAgain,
              busy: false,
              onPressed: () {
                setState(() => _phase = _Phase.reconnecting);
                _connect(again: true);
              },
            ),
          ],
        );
      case _Phase.ended:
        return ui.Page(title: _end?.title, children: [ui.Message(_end?.line ?? closeScreen)]);
      case _Phase.call || _Phase.reconnecting:
        return _call(context);
    }
  }

  Widget _call(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Align(alignment: Alignment.centerLeft, child: BackButton()),
              Expanded(child: _stage(context)),
              const SizedBox(height: 12),
              if (_reconnected) const Center(child: ui.Message(reconnected)),
              _controls(),
            ],
          ),
        ),
      ),
    );
  }

  /// The practitioner's video, large, with the patient's own camera small in a corner. The lines that need the
  /// practitioner's name are left out without one.
  Widget _stage(BuildContext context) {
    final room = _room;
    // Between connections (reconnecting, or asking why the room was left) there is nothing to say about the peer.
    final between = _phase == _Phase.reconnecting || room == null;
    final name = _peerName;
    final white = Theme.of(context).textTheme.bodyLarge?.copyWith(color: Colors.white);
    final lines = <Widget>[
      if (!_peer.present && !between && !_seen) ...[
        Semantics(
          header: true,
          child: Text(waiting, style: Theme.of(context).textTheme.titleLarge?.copyWith(color: Colors.white)),
        ),
        if (name.isNotEmpty) Text(joinsYouHere(name), style: white, textAlign: TextAlign.center),
      ],
      if (!_peer.present && !between && _seen && name.isNotEmpty)
        Text(peerLeft(name), style: white, textAlign: TextAlign.center),
      if (_peer.present && !_peer.camera && name.isNotEmpty) Text(peerCameraOff(name), style: white),
    ];
    return ClipRRect(
      borderRadius: BorderRadius.circular(12),
      child: ColoredBox(
        color: _ink,
        child: Stack(
          fit: StackFit.expand,
          children: [
            if (room != null && _peer.present && _peer.camera) room.remoteView(),
            Center(
              child: Padding(
                padding: const EdgeInsets.all(24),
                child: Column(mainAxisSize: MainAxisSize.min, spacing: 8, children: lines),
              ),
            ),
            if (_peer.present && !_peer.microphone && name.isNotEmpty)
              Positioned(left: 12, top: 12, child: _badge(peerMuted(name))),
            if (room != null && _settings.camera)
              Positioned(
                right: 12,
                bottom: 12,
                width: 96,
                height: 128,
                child: ClipRRect(borderRadius: BorderRadius.circular(8), child: room.localView()),
              ),
            if (_phase == _Phase.reconnecting)
              Positioned(left: 12, right: 12, top: 12, child: Semantics(liveRegion: true, child: _badge(reconnecting))),
          ],
        ),
      ),
    );
  }

  static Widget _badge(String text) => DecoratedBox(
    decoration: BoxDecoration(color: Colors.white, borderRadius: BorderRadius.circular(8)),
    child: Padding(padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6), child: Text(text)),
  );

  Widget _controls() {
    final room = _room;
    final microphone = _settings.microphone;
    final camera = _settings.camera;
    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      spacing: 16,
      children: [
        IconButton.filledTonal(
          tooltip: microphone ? mute : unmute,
          onPressed: room == null ? null : () => _toggle(Kind.microphone),
          icon: Icon(microphone ? Icons.mic : Icons.mic_off),
        ),
        IconButton.filledTonal(
          tooltip: camera ? cameraOff : cameraOn,
          onPressed: room == null || !_cameraUsable ? null : () => _toggle(Kind.camera),
          icon: Icon(camera ? Icons.videocam : Icons.videocam_off),
        ),
        FilledButton(
          style: FilledButton.styleFrom(backgroundColor: Theme.of(context).colorScheme.error),
          onPressed: () => Navigator.of(context).pop(),
          child: const Text(leave),
        ),
      ],
    );
  }
}
