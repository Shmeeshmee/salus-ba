import 'dart:io';
import 'dart:typed_data';

import 'package:salus_api/api.dart' as api;

import 'session.dart';

/// The patient's data subject requests and the closing of their account (docs/clinic-cms/WAVE5.md, S1 to S6),
/// through the generated client. Every failure is an [ApiFailure].
class DataRequests {
  DataRequests(api.ApiClient client) : _client = client, _api = api.PrivacyApi(client);

  final api.ApiClient _client;
  final api.PrivacyApi _api;

  /// Every request the patient filed, newest first.
  Future<List<api.DataRequest>> mine() async {
    final all = await apiCall(() => _api.listMyDataRequests());
    return all ?? const [];
  }

  /// Files a request of [kind] to the patient's Salus account, or to their record at [clinicId] when given.
  Future<api.DataRequest> file(api.DataRequestKind kind, {String? clinicId}) async {
    final filed = await apiCall(
      () => _api.fileMyDataRequest(
        api.DataRequestFiling(
          kind: kind,
          target: clinicId == null ? api.DataRequestTarget.account : api.DataRequestTarget.clinic,
          organizationId: clinicId,
        ),
      ),
    );
    if (filed == null) throw const ApiFailure(status: HttpStatus.badGateway);
    return filed;
  }

  /// Withdraws [request] as it was read (its version in `If-Match`).
  Future<void> withdraw(api.DataRequest request) =>
      apiCall(() => _api.withdrawMyDataRequest(request.id, 'W/"${request.version}"'));

  /// Where the export of the access or portability request [id] stands, or null when there is none yet or it cannot
  /// be read now (the request's card then shows no export; the rest of the list stays).
  Future<api.DataRequestExport?> export(String id) async {
    try {
      return await apiCall(() => _api.getMyDataRequestExport(id));
    } on ApiFailure catch (f) {
      if (f.status == HttpStatus.notFound || f.status >= HttpStatus.internalServerError || f.unreachable) return null;
      rethrow;
    }
  }

  /// The bytes of the ready export of the request [id]: a zip archive of FHIR Bundles.
  Future<Uint8List> archive(String id) async {
    final response = await apiCall(
      () => _client.invokeAPI(
        '/me/data-requests/${Uri.encodeComponent(id)}/export/archive',
        'GET',
        const [],
        null,
        {'Accept': 'application/zip'},
        {},
        null,
      ),
    );
    if (response.statusCode >= HttpStatus.badRequest) throw ApiFailure.fromBody(response.statusCode, response.body);
    return response.bodyBytes;
  }

  /// Closes the patient's account, confirmed with their [password]; the session ends with it.
  Future<void> closeAccount(String password) =>
      apiCall(() => _api.closeMyAccount(api.AccountClosure(password: password)));
}
