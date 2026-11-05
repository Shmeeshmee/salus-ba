import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

import 'call.dart';
import 'messages.dart';
import 'ui.dart' as ui;

/// Below this level the microphone counts as silent.
const _silence = 0.02;

/// The step before joining (docs/consultation/SPEC.md "Client states"): the camera's preview, the microphone's
/// level, and what the device does not allow, with a way to its settings. On Android the app says why it needs
/// the camera and microphone before the system asks (M35); on iOS the system prompt carries Info.plist's text.
/// Join waits until both devices have opened or failed, then the patient joins with what remains; [onJoin] gets
/// the settings and whether there is a camera to turn on later. The previews stop on Join, as the room opens the
/// devices itself, and open again when the join was refused.
class DeviceCheck extends StatefulWidget {
  const DeviceCheck({super.key, required this.media, required this.busy, this.error, required this.onJoin});

  final Media media;
  final bool busy;
  final String? error;
  final void Function(Settings settings, bool cameraUsable) onJoin;

  @override
  State<DeviceCheck> createState() => _DeviceCheckState();
}

class _DeviceCheckState extends State<DeviceCheck> with WidgetsBindingObserver {
  final _opened = <Kind, Opened>{};
  var _explain = false;
  var _cameraOff = false;
  var _microphoneOff = false;
  var _level = 0.0;
  var _silent = false;
  Timer? _quiet;
  StreamSubscription<double>? _levels;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _start();
  }

  @override
  void didUpdateWidget(DeviceCheck old) {
    super.didUpdateWidget(old);
    if (old.busy && !widget.busy) _start();
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _stop();
    super.dispose();
  }

  /// Back from the device settings: a device allowed there opens now.
  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state != AppLifecycleState.resumed) return;
    for (final kind in Kind.values) {
      if (_opened[kind]?.problem == Problem.denied) _reopen(kind);
    }
  }

  Future<void> _reopen(Kind kind) async {
    final access = await widget.media.access(kind);
    if (mounted && access == Access.allowed) await _open(kind, access);
  }

  Future<void> _start() async {
    final access = {for (final kind in Kind.values) kind: await widget.media.access(kind)};
    if (!mounted) return;
    if (access.containsValue(Access.unasked)) {
      if (defaultTargetPlatform == TargetPlatform.android) {
        setState(() => _explain = true);
        await WidgetsBinding.instance.endOfFrame;
      }
      for (final kind in Kind.values) {
        if (access[kind] == Access.unasked) access[kind] = await widget.media.ask(kind);
        if (!mounted) return;
      }
      setState(() => _explain = false);
    }
    for (final kind in Kind.values) {
      await _open(kind, access[kind]!);
    }
  }

  Future<void> _open(Kind kind, Access access) async {
    final opened = access == Access.allowed ? await widget.media.open(kind) : const Opened.problem(Problem.denied);
    if (!mounted || widget.busy) {
      await opened.preview?.stop();
      return;
    }
    await _opened[kind]?.preview?.stop();
    setState(() => _opened[kind] = opened);
    final preview = opened.preview;
    if (kind == Kind.microphone && preview != null) _listen(preview);
  }

  /// The level, and silence once no reading has reached [_silence] for 5 seconds. Until a first reading arrives
  /// the level is not measured, which is neither sound nor silence.
  void _listen(Preview microphone) {
    _levels?.cancel();
    _levels = microphone.levels.listen((level) {
      if (!mounted) return;
      setState(() => _level = level);
      if (level < _silence) {
        _quiet ??= Timer(const Duration(seconds: 5), _silenced);
        return;
      }
      _quiet?.cancel();
      _quiet = Timer(const Duration(seconds: 5), _silenced);
      setState(() => _silent = false);
    });
  }

  void _silenced() {
    if (mounted) setState(() => _silent = true);
  }

  /// Stops the previews; what was opened stays on screen until the check opens them again.
  void _stop() {
    _quiet?.cancel();
    _quiet = null;
    _levels?.cancel();
    _levels = null;
    for (final opened in _opened.values) {
      opened.preview?.stop();
    }
    _silent = false;
    _level = 0;
  }

  bool _usable(Kind kind) => _opened[kind]?.preview != null;

  void _join() {
    final camera = _usable(Kind.camera);
    final microphone = _usable(Kind.microphone);
    setState(_stop);
    widget.onJoin(Settings(camera: camera && !_cameraOff, microphone: microphone && !_microphoneOff), camera);
  }

  @override
  Widget build(BuildContext context) {
    final camera = _opened[Kind.camera];
    final microphone = _opened[Kind.microphone];
    final settled = camera != null && microphone != null;
    final denied = camera?.problem == Problem.denied || microphone?.problem == Problem.denied;
    return ui.Page(
      title: checkTitle,
      children: [
        const Text(checkIntro),
        if (_explain) const ui.Message(whyCameraAndMicrophone),
        if (camera?.preview case final preview?)
          ClipRRect(
            borderRadius: BorderRadius.circular(12),
            child: SizedBox(height: 240, child: widget.busy ? null : preview.view()),
          ),
        if (camera?.problem case final problem?)
          ui.Message(problem == Problem.denied ? cameraDenied : noCamera, error: true),
        if (microphone?.preview != null) ...[
          LinearProgressIndicator(value: _level, minHeight: 8, borderRadius: BorderRadius.circular(4)),
          if (_silent) const ui.Message(noSound),
        ],
        if (microphone?.problem case final problem?)
          ui.Message(problem == Problem.denied ? microphoneDenied : noMicrophone, error: true),
        if (denied) OutlinedButton(onPressed: widget.media.openSettings, child: const Text(openSettings)),
        _option(joinCameraOff, Kind.camera, _cameraOff, (off) => _cameraOff = off),
        _option(joinMicrophoneOff, Kind.microphone, _microphoneOff, (off) => _microphoneOff = off),
        if (widget.error != null) ui.Message(widget.error!, error: true),
        FilledButton(onPressed: widget.busy || !settled ? null : _join, child: Text(widget.busy ? joining : joinNow)),
      ],
    );
  }

  /// "Join with … off": ticked and fixed when the device cannot be used.
  Widget _option(String label, Kind kind, bool off, void Function(bool off) set) {
    final usable = _usable(kind);
    return CheckboxListTile(
      title: Text(label),
      value: off || !usable,
      contentPadding: EdgeInsets.zero,
      controlAffinity: ListTileControlAffinity.leading,
      onChanged: usable ? (v) => setState(() => set(v ?? false)) : null,
    );
  }
}
