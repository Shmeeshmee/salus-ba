import 'package:flutter/material.dart';

import 'booking.dart';
import 'call.dart';
import 'fhir.dart';
import 'files.dart';
import 'loading.dart';
import 'messages.dart';
import 'slots_screen.dart';
import 'ui.dart' as ui;

/// The clinics that chose to be listed.
class ClinicsScreen extends StatefulWidget {
  const ClinicsScreen({super.key, required this.booking, required this.now, required this.media, required this.files});

  final Booking booking;
  final DateTime Function() now;
  final Media media;
  final Files files;

  @override
  State<ClinicsScreen> createState() => _ClinicsScreenState();
}

class _ClinicsScreenState extends LoadingState<ClinicsScreen, List<Organization>> {
  @override
  Future<List<Organization>> load() => widget.booking.fhir.listedClinics();

  @override
  Widget build(BuildContext context) => ui.Page(
    title: context.t.chooseClinic,
    children: body(
      (clinics) => [
        if (clinics.isEmpty) Text(context.t.noClinics),
        for (final clinic in clinics)
          ui.Choice(
            clinic.name,
            onTap: () => Navigator.of(context).push(
              MaterialPageRoute<void>(
                builder: (_) => PractitionersScreen(
                  booking: widget.booking,
                  now: widget.now,
                  media: widget.media,
                  files: widget.files,
                  clinic: clinic,
                ),
              ),
            ),
          ),
      ],
    ),
  );
}

/// A listed clinic's practitioners, each with the specialty the API gives.
class PractitionersScreen extends StatefulWidget {
  const PractitionersScreen({
    super.key,
    required this.booking,
    required this.now,
    required this.media,
    required this.files,
    required this.clinic,
  });

  final Booking booking;
  final DateTime Function() now;
  final Media media;
  final Files files;
  final Organization clinic;

  @override
  State<PractitionersScreen> createState() => _PractitionersScreenState();
}

class _PractitionersScreenState extends LoadingState<PractitionersScreen, List<PractitionerRole>> {
  @override
  Future<List<PractitionerRole>> load() => widget.booking.fhir.practitioners(widget.clinic.id);

  @override
  Widget build(BuildContext context) => ui.Page(
    title: widget.clinic.name,
    children: body(
      (roles) => [
        Text(roles.isEmpty ? context.t.noPractitioners : context.t.choosePractitioner),
        for (final role in roles)
          ui.Choice(
            role.name,
            subtitle: specialtyName(context.t, role),
            onTap: () => Navigator.of(context).push(
              MaterialPageRoute<void>(
                builder: (_) => SlotsScreen(
                  booking: widget.booking,
                  now: widget.now,
                  media: widget.media,
                  files: widget.files,
                  clinic: widget.clinic,
                  role: role,
                ),
              ),
            ),
          ),
      ],
    ),
  );
}
