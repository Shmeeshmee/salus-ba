import 'package:flutter/material.dart';
import 'package:intl/intl.dart' show DateFormat;

import 'package:salus_api/api.dart' show ChatInvitation;

import 'booking.dart' show Places;
import 'chat.dart';
import 'messages.dart';
import 'ui.dart' as ui;

// The patient's consent prompt when the clinic invites a doctor from another clinic (SAL-28, SAL-31), the
// revocation of a granted one, and the clinic's chat hours (SAL-29). The prompt's and the revocation's words carry
// the legal tag in the catalogs.

/// The consent prompt: who asks and why, which doctor of which clinic, what the doctor would see, that the patient
/// may decline or later revoke. Access cannot be granted before the doctor and their clinic are shown: until then
/// the card says so and offers to load them again ([onRetry]).
class InvitationCard extends StatelessWidget {
  const InvitationCard({
    super.key,
    required this.invitation,
    required this.history,
    required this.names,
    required this.busy,
    required this.onDecide,
    required this.onRetry,
  });

  final ChatInvitation invitation;

  /// What the doctor would read: `full`, `from`, or `unknown` when the invited participant could not be read.
  final String history;
  final Places names;
  final bool busy;
  final void Function(bool grant) onDecide;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final t = context.t;
    final named = names.roles[invitation.practitionerRoleId];
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            ui.Heading(t.chatInviteTitle),
            const SizedBox(height: 8),
            Text(t.chatInviteBody(invitation.reason.toString())),
            const SizedBox(height: 8),
            if (named != null)
              Text(t.chatInviteWho(named.$2.name, named.$1.name))
            else ...[
              Text(t.chatInviteUnresolved),
              Align(alignment: AlignmentDirectional.centerStart, child: TextButton(onPressed: onRetry, child: Text(t.tryAgain))),
            ],
            const SizedBox(height: 8),
            Text(t.chatInviteSees(history)),
            const SizedBox(height: 8),
            Text(t.chatInviteChoice),
            const SizedBox(height: 16),
            Wrap(
              alignment: WrapAlignment.end,
              spacing: 8,
              runSpacing: 8,
              children: [
                OutlinedButton(onPressed: busy ? null : () => onDecide(false), child: Text(t.chatDecline)),
                FilledButton(onPressed: busy || named == null ? null : () => onDecide(true), child: Text(t.chatGrant)),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

/// Asks before revoking a granted invitation; true to revoke.
Future<bool> confirmRevoke(BuildContext context) async {
  final t = context.t;
  final yes = await showDialog<bool>(
    context: context,
    builder: (context) => AlertDialog(
      title: Text(t.chatRevoke),
      content: Text(t.chatRevokeBody),
      actions: [
        TextButton(onPressed: () => Navigator.of(context).pop(false), child: Text(t.cancel)),
        TextButton(onPressed: () => Navigator.of(context).pop(true), child: Text(t.chatRevokeConfirm)),
      ],
    ),
  );
  return yes ?? false;
}

/// The ekavian day names intl's Serbian data has, and the ijekavian ones Serbian in Bosnia and Herzegovina writes,
/// as zones.dart writes them in dates.
const _ijekavian = {'ponedeljak': 'ponedjeljak', 'sreda': 'srijeda', 'nedelja': 'nedjelja', 'понедељак': 'понедјељак', 'среда': 'сриједа', 'недеља': 'недјеља'};

/// The name of ISO weekday [weekday] (Monday 1) in gen-l10n's locale [locale].
String weekdayName(int weekday, String locale) {
  final name = DateFormat.EEEE(locale == 'sr_Cyrl' ? 'sr' : locale).format(DateTime.utc(2026, 11, 8 + weekday));
  return _ijekavian[name] ?? name;
}

/// The clinic's chat hours per weekday and its usual response time.
Future<void> showChatHours(BuildContext context, ChatHours hours) {
  final t = context.t;
  return showDialog<void>(
    context: context,
    builder: (context) => AlertDialog(
      title: Text(t.chatHours),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (hours.hours.isEmpty) Text(t.chatHoursNone),
          for (final d in hours.hours) Text(t.chatHoursDay(weekdayName(d.weekday, t.localeName), d.start, d.end)),
          const SizedBox(height: 12),
          Text(t.chatResponseTime(hours.responseTimeHours)),
        ],
      ),
      actions: [TextButton(onPressed: () => Navigator.of(context).pop(), child: Text(t.termClose))],
    ),
  );
}
