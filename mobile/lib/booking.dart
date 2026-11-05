import 'dart:io';

import 'package:salus_api/api.dart' as api;

import 'fhir.dart';
import 'session.dart';

/// Where an appointment or a consent belongs: the listed clinics by id, and their practitioners by role id.
class Places {
  const Places(this.clinics, this.roles);

  final Map<String, Organization> clinics;
  final Map<String, (Organization, PractitionerRole)> roles;
}

/// The patient's booking calls: the JSON actions through the generated client (docs/booking/SPEC.md "JSON API")
/// and the FHIR reads in [fhir]. Every failure is an [ApiFailure].
class Booking {
  Booking(api.ApiClient client)
    : fhir = Fhir(client),
      _patient = api.PatientApi(client),
      _consents = api.ConsentsApi(client),
      _appointments = api.AppointmentsApi(client);

  final Fhir fhir;
  final api.PatientApi _patient;
  final api.ConsentsApi _consents;
  final api.AppointmentsApi _appointments;

  /// The signed-in patient's profile, or null before it is first saved.
  Future<api.Patient?> me() async {
    try {
      return await apiCall(_patient.getMyPatient);
    } on ApiFailure catch (f) {
      if (f.status == HttpStatus.notFound) return null;
      rethrow;
    }
  }

  /// Creates the profile ([versionId] null) or updates the version given.
  Future<api.Patient> saveMe({
    required String givenName,
    required String familyName,
    required DateTime birthDate,
    String? phone,
    int? versionId,
  }) async {
    final saved = await apiCall(
      () => _patient.saveMyPatient(
        api.SavePatientRequest(
          givenName: givenName,
          familyName: familyName,
          birthDate: birthDate,
          phone: phone,
          versionId: versionId,
        ),
      ),
    );
    if (saved == null) throw const ApiFailure(status: HttpStatus.badGateway);
    return saved;
  }

  Future<List<api.Consent>> consents() async => (await apiCall(_consents.listMyConsents))?.consents ?? const [];

  Future<void> grant(String clinicId) =>
      apiCall(() => _consents.grantConsent(api.GrantConsentRequest(organizationId: clinicId)));

  Future<void> revoke(String consentId) => apiCall(() => _consents.revokeConsent(consentId));

  Future<void> book(String slotId) =>
      apiCall(() => _appointments.bookAppointment(api.BookAppointmentRequest(slotId: slotId)));

  Future<void> cancel(String appointmentId) => apiCall(() => _appointments.cancelAppointment(appointmentId));

  /// The listed clinics, with the practitioners of those among [clinicIds]. A clinic that is no longer listed
  /// is in neither map.
  Future<Places> places(Iterable<String> clinicIds) async {
    final clinics = {for (final c in await fhir.listedClinics()) c.id: c};
    final roles = <String, (Organization, PractitionerRole)>{};
    for (final id in clinicIds.toSet()) {
      final clinic = clinics[id];
      if (clinic == null) continue;
      for (final role in await fhir.practitioners(id)) {
        roles[role.id] = (clinic, role);
      }
    }
    return Places(clinics, roles);
  }
}
