import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:http/http.dart' as http;
import 'package:salus_api/api.dart' as api;

import 'fhir.dart';
import 'file_prep.dart' show Prepared;
import 'health_profile.dart';
import 'session.dart';

/// How long an upload may go without sending a byte or hearing its answer before it is stopped (COPY M5).
const uploadIdle = Duration(seconds: 60);

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
    : _client = client,
      fhir = Fhir(client),
      health = HealthProfile(client),
      notifications = api.NotificationsApi(client),
      _patient = api.PatientApi(client),
      _consents = api.ConsentsApi(client),
      _appointments = api.AppointmentsApi(client),
      _consultation = api.ConsultationApi(client),
      _questionnaires = api.QuestionnairesApi(client),
      _history = api.HistoryApi(client);

  final api.ApiClient _client;
  final Fhir fhir;

  /// The allergies and medications of the profile (SAL-41).
  final HealthProfile health;

  /// The patient's reminder channels, quiet hours and zone (SAL-33).
  final api.NotificationsApi notifications;
  final api.PatientApi _patient;
  final api.ConsentsApi _consents;
  final api.AppointmentsApi _appointments;
  final api.ConsultationApi _consultation;
  final api.QuestionnairesApi _questionnaires;
  final api.HistoryApi _history;

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

  Future<void> grant(String clinicId) => apiCall(
    () => _consents.grantConsent(api.GrantConsentRequest(organizationId: clinicId, kind: api.ConsentKind.care)),
  );

  /// The transcription consent with the clinic (SAL-12): transcription and translation of the consultations, with
  /// AI assistance for the practitioner. An active one already is `consent_active`.
  Future<void> grantTranscription(String clinicId) => apiCall(
    () =>
        _consents.grantConsent(api.GrantConsentRequest(organizationId: clinicId, kind: api.ConsentKind.transcription)),
  );

  Future<void> revoke(String consentId) => apiCall(() => _consents.revokeConsent(consentId));

  /// The visits the patient may share with the clinic (SAL-42), newest first, each as the clinic would see it, marked
  /// when shared now.
  Future<api.SharingCandidateList> sharingPreview(String clinicId) async {
    final preview = await apiCall(() => _history.previewHistorySharing(clinicId));
    return preview ?? api.SharingCandidateList();
  }

  /// What the patient shares with each clinic now.
  Future<List<api.HistorySharing>> mySharing() async =>
      (await apiCall(_history.listMyHistorySharing))?.sharing ?? const [];

  /// Shares exactly [encounterIds] with the clinic, changing the sharing at [versionId] (null for a first grant);
  /// none stops it. A change made meanwhile is `version_conflict`.
  Future<void> setSharing(String clinicId, List<String> encounterIds, String? versionId) => apiCall(
    () => _history.setHistorySharing(
      clinicId,
      api.SetHistorySharingRequest(encounterIds: encounterIds.toSet()),
      ifMatch: versionId == null ? null : 'W/"$versionId"',
    ),
  );

  /// Stops sharing history with the clinic; nothing when nothing is shared.
  Future<void> stopSharing(String clinicId) => apiCall(() => _history.revokeHistorySharing(clinicId));

  Future<void> book(String slotId) =>
      apiCall(() => _appointments.bookAppointment(api.BookAppointmentRequest(slotId: slotId)));

  Future<void> cancel(String appointmentId) => apiCall(() => _appointments.cancelAppointment(appointmentId));

  /// A token for the appointment's video consultation (docs/consultation/SPEC.md "Join response"). The token is a
  /// secret: it goes to LiveKit and nowhere else.
  Future<api.ConsultationJoin> join(String appointmentId) async {
    final join = await apiCall(() => _consultation.joinConsultation(appointmentId));
    if (join == null) throw const ApiFailure(status: HttpStatus.badGateway);
    return join;
  }

  /// Where the appointment's video consultation stands.
  Future<api.ConsultationStatus> consultation(String appointmentId) async {
    final status = await apiCall(() => _consultation.getConsultation(appointmentId));
    if (status == null) throw const ApiFailure(status: HttpStatus.badGateway);
    return status;
  }

  /// The PDF of the signed document whose DocumentReference is [id] (docs/consultation/SPEC.md "Patient documents").
  Future<Uint8List> pdf(String id) async {
    final response = await apiCall(
      () => _client.invokeAPI(
        '/document-references/${Uri.encodeComponent(id)}/content',
        'GET',
        const [],
        null,
        {'Accept': 'application/pdf'},
        {},
        null,
      ),
    );
    if (response.statusCode >= HttpStatus.badRequest) throw ApiFailure.fromBody(response.statusCode, response.body);
    return response.bodyBytes;
  }

  /// One page of the patient's history across every clinic (SAL-39), newest first, after [cursor] (the `next` of the
  /// page before; null for the first), narrowed to one clinic, some types and the instants in [from, to) when given.
  Future<api.HistoryPage> history({
    String? cursor,
    String? clinicId,
    List<api.HistoryItemType>? types,
    DateTime? from,
    DateTime? to,
  }) async {
    final page = await apiCall(
      () => _history.listMyHistory(
        cursor: cursor,
        organization: clinicId,
        type: types == null || types.isEmpty ? null : types,
        from: from?.toUtc(),
        to: to?.toUtc(),
      ),
    );
    if (page == null) throw const ApiFailure(status: HttpStatus.badGateway);
    return page;
  }

  /// The bytes of the patient's clean file [binaryId] and its content type, to hand to the device (SAL-39).
  Future<(Uint8List, String)> file(String binaryId) async {
    final response = await apiCall(
      () => _client.invokeAPI('/binaries/${Uri.encodeComponent(binaryId)}/content', 'GET', const [], null, {}, {}, null),
    );
    if (response.statusCode >= HttpStatus.badRequest) throw ApiFailure.fromBody(response.statusCode, response.body);
    return (response.bodyBytes, response.headers['content-type'] ?? 'application/octet-stream');
  }

  /// The plain-language explanations of the medical terms in [language], else in English (SAL-13), by term id.
  Future<Map<String, api.ExplainedTerm>> explanations(String language) async {
    final list = await apiCall(
      () => api.TerminologyApi(_client).getTermExplanations(api.TermLanguage.fromJson(language)!),
    );
    return {for (final e in list?.explanations ?? const <api.ExplainedTerm>[]) e.termId: e};
  }

  /// The explained terms of the signed document whose Composition is [id], per field and line (SAL-13).
  Future<api.DocumentTerms?> documentTerms(String id) => apiCall(() => api.DocumentsApi(_client).getDocumentTerms(id));

  /// The patient's copy of the consultation's stored transcript, or null while the clinic does not share it (403).
  Future<List<api.TranscriptEvent>?> transcript(String encounterId) async {
    try {
      return (await apiCall(() => _consultation.exportTranscript(encounterId)))?.segments ?? const [];
    } on ApiFailure catch (f) {
      if (f.status == HttpStatus.forbidden) return null;
      rethrow;
    }
  }

  /// Starts the response [id] in [language], one of its questionnaire's languages (docs/i18n/SPEC.md §3 "The
  /// patient's language"); again with another until an answer or a file is saved. The answer carries the new version.
  Future<api.QuestionnaireResponse> start(String id, String language) => _answered(
    () => _questionnaires.startQuestionnaire(
      id,
      api.StartQuestionnaireRequest(language: api.Language.fromJson(language)!),
    ),
  );

  /// Replaces the saved answers of the response at [versionId] (docs/questionnaire/SPEC.md §2 "Patient flow");
  /// the answer carries the new version.
  Future<api.QuestionnaireResponse> saveAnswers(String id, int versionId, List<api.Answer> answers) => _answered(
    () => _questionnaires.saveQuestionnaireAnswers(_etag(versionId), id, api.SaveAnswersRequest(answers: answers)),
  );

  /// Submits the saved answers of the response at [versionId]; final.
  Future<api.QuestionnaireResponse> submitAnswers(String id, int versionId) =>
      _answered(() => _questionnaires.submitQuestionnaireAnswers(_etag(versionId), id));

  static String _etag(int versionId) => 'W/"$versionId"';

  static Future<api.QuestionnaireResponse> _answered(Future<api.QuestionnaireResponse?> Function() call) async {
    final r = await apiCall(call);
    if (r == null) throw const ApiFailure(status: HttpStatus.badGateway);
    return r;
  }

  /// Attaches [file] to the attachment item [linkId] of the response [id] (docs/attachments/SPEC.md §2 "Upload
  /// pipeline"): the bytes streamed as the body with `Content-Length`, the name in `Content-Disposition` and never in
  /// the URL, through the client's base and bearer token. The generated client would send a multipart form, which
  /// the API does not take. [onSent] hears the bytes sent so far; [abort] stops the upload, which then throws
  /// [http.RequestAbortedException]. An upload that makes no progress for [uploadIdle], its bytes or its answer, is
  /// stopped and fails as unreachable ([ApiFailure]), as a lost connection does.
  Future<api.Binary> upload(
    String id,
    String linkId,
    Prepared file, {
    required void Function(int sent) onSent,
    Future<void>? abort,
  }) async {
    final headers = <String, String>{};
    await _client.authentication?.applyToParams(<api.QueryParam>[], headers);
    final stalled = Completer<void>();
    Timer? idle;
    void waiting() {
      idle?.cancel();
      if (!stalled.isCompleted) idle = Timer(uploadIdle, stalled.complete);
    }

    var sent = 0;
    final body = file.read().map((chunk) {
      sent += chunk.length;
      if (!stalled.isCompleted) onSent(sent);
      waiting();
      return chunk;
    });
    final path = '/questionnaire-responses/${Uri.encodeComponent(id)}/items/${Uri.encodeComponent(linkId)}/attachments';
    final request = _Upload(Uri.parse('${_client.basePath}$path'), body, Future.any([?abort, stalled.future]))
      ..contentLength = file.size
      ..headers.addAll({
        ...headers,
        'Content-Type': 'application/octet-stream',
        'Content-Disposition': contentDisposition(file.name),
      });
    waiting();
    unawaited(abort?.then((_) => idle?.cancel()));
    try {
      final (status, text) = await Future.any([
        _send(request),
        stalled.future.then<(int, String)>((_) => throw const ApiFailure()),
      ]);
      if (status >= HttpStatus.badRequest) throw ApiFailure.fromBody(status, text);
      return _binary(text);
    } finally {
      idle?.cancel();
    }
  }

  /// A file's status (`quarantined`, `clean`, `rejected`).
  Future<api.Binary> fileStatus(String binaryId) async {
    final b = await apiCall(() => _attachments.getBinary(binaryId));
    if (b == null) throw const ApiFailure(status: HttpStatus.badGateway);
    return b;
  }

  /// Removes a file from the patient's answers; the server deletes it.
  Future<void> removeFile(String binaryId) => apiCall(() => _attachments.deleteBinary(binaryId));

  api.AttachmentsApi get _attachments => api.AttachmentsApi(_client);

  Future<(int, String)> _send(http.BaseRequest request) async {
    try {
      final response = await _client.client.send(request);
      return (response.statusCode, await response.stream.bytesToString());
    } on http.RequestAbortedException {
      rethrow;
    } on Exception {
      throw const ApiFailure();
    }
  }

  static api.Binary _binary(String text) {
    try {
      final b = api.Binary.fromJson(jsonDecode(text));
      if (b != null) return b;
    } on FormatException {
      // Not the API's JSON.
    }
    throw const ApiFailure(status: HttpStatus.badGateway);
  }

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

/// A streamed body of known length: read from the file as the connection takes it, so the memory used is a chunk
/// whatever the file's size.
final class _Upload extends http.BaseRequest implements http.Abortable {
  _Upload(Uri url, this._body, this.abortTrigger) : super('POST', url);

  final Stream<List<int>> _body;

  @override
  final Future<void>? abortTrigger;

  @override
  http.ByteStream finalize() {
    super.finalize();
    return http.ByteStream(_body);
  }
}

/// `attachment; filename*=UTF-8''<name>` (RFC 6266, RFC 8187): the name percent-encoded, with the characters
/// RFC 8187 does not allow bare (`'`, `(`, `)`, `*`, `!`) encoded too.
String contentDisposition(String name) {
  final encoded = Uri.encodeComponent(name)
      .replaceAllMapped(RegExp(r"[!'()*]"), (m) => '%${m[0]!.codeUnitAt(0).toRadixString(16).toUpperCase()}');
  return "attachment; filename*=UTF-8''$encoded";
}
