import 'package:flutter/widgets.dart';

/// The privacy notice's address (PRIVACY_NOTICE_URL, set at build time) and how to open it on the device, for
/// every screen under the app: the consent screen links to it.
class PrivacyNotice extends InheritedWidget {
  const PrivacyNotice({super.key, required this.url, required this.open, required super.child});

  final Uri url;

  /// Opens a link outside the app; false when the device could not.
  final Future<bool> Function(Uri url) open;

  static PrivacyNotice of(BuildContext context) => context.dependOnInheritedWidgetOfExactType<PrivacyNotice>()!;

  @override
  bool updateShouldNotify(PrivacyNotice oldWidget) => url != oldWidget.url || open != oldWidget.open;
}
