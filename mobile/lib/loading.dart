import 'package:flutter/widgets.dart';

import 'messages.dart';
import 'session.dart';
import 'ui.dart' as ui;

/// Loads once on open, shows a spinner, then the data or a fixed sentence with "Try again".
abstract class LoadingState<W extends StatefulWidget, T> extends State<W> {
  T? data;
  String? error;

  Future<T> load();

  @override
  void initState() {
    super.initState();
    reload();
  }

  Future<void> reload() async {
    setState(() {
      data = null;
      error = null;
    });
    try {
      final loaded = await load();
      if (mounted) setState(() => data = loaded);
    } on ApiFailure catch (failure) {
      if (mounted) setState(() => error = bookingMessage(failure));
    }
  }

  /// The page's content: [ready] once loaded.
  List<Widget> body(List<Widget> Function(T data) ready) {
    if (error != null) return [ui.LoadFailed(error!, onRetry: reload)];
    final loaded = data;
    return loaded == null ? const [ui.Loading()] : ready(loaded);
  }
}
