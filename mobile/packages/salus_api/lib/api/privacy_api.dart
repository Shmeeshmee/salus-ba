//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class PrivacyApi {
  PrivacyApi([ApiClient? apiClient])
      : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// Answer a request targeting the clinic
  ///
  /// Fulfilled or refused, with a code told to the patient (never free text about health). For the clinic's admin, while the request is open. Built in package D1. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [DataRequestAnswer] dataRequestAnswer (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> answerClinicDataRequestWithHttpInfo(
    String id,
    String ifMatch,
    DataRequestAnswer dataRequestAnswer, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/data-requests/{id}/answer'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = dataRequestAnswer;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    headerParams[r'If-Match'] = parameterToString(ifMatch);
    if (xSalusCSRF != null) {
      headerParams[r'X-Salus-CSRF'] = parameterToString(xSalusCSRF);
    }

    const contentTypes = <String>['application/json'];

    return apiClient.invokeAPI(
      path,
      'POST',
      queryParams,
      postBody,
      headerParams,
      formParams,
      contentTypes.isEmpty ? null : contentTypes.first,
      abortTrigger: abortTrigger,
    );
  }

  /// Answer a request targeting the clinic
  ///
  /// Fulfilled or refused, with a code told to the patient (never free text about health). For the clinic's admin, while the request is open. Built in package D1. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [DataRequestAnswer] dataRequestAnswer (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<DataRequest?> answerClinicDataRequest(
    String id,
    String ifMatch,
    DataRequestAnswer dataRequestAnswer, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await answerClinicDataRequestWithHttpInfo(
      id,
      ifMatch,
      dataRequestAnswer,
      xSalusCSRF: xSalusCSRF,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
    // When a remote server returns no body with a status of 204, we shall not decode it.
    // At the time of writing this, `dart:convert` will throw an "Unexpected end of input"
    // FormatException when trying to decode an empty string.
    if (response.body.isNotEmpty &&
        response.statusCode != HttpStatus.noContent) {
      return await apiClient.deserializeAsync(
        await _decodeBodyBytes(response),
        'DataRequest',
      ) as DataRequest;
    }
    return null;
  }

  /// Close the patient's account
  ///
  /// At once: sign-in ends, sessions and device tokens are removed, future visits are cancelled with their notices, waitlist entries withdrawn and consents revoked (S5). Not reversible by the patient; the operator may reopen it within 30 days, after which what has no legal basis to stay is erased and the clinical record stays with each clinic, restricted (S6). Confirmed with the password. For a patient. Built in package D3. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [AccountClosure] accountClosure (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> closeMyAccountWithHttpInfo(
    AccountClosure accountClosure, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/me/account/close';

    // ignore: prefer_final_locals
    Object? postBody = accountClosure;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    if (xSalusCSRF != null) {
      headerParams[r'X-Salus-CSRF'] = parameterToString(xSalusCSRF);
    }

    const contentTypes = <String>['application/json'];

    return apiClient.invokeAPI(
      path,
      'POST',
      queryParams,
      postBody,
      headerParams,
      formParams,
      contentTypes.isEmpty ? null : contentTypes.first,
      abortTrigger: abortTrigger,
    );
  }

  /// Close the patient's account
  ///
  /// At once: sign-in ends, sessions and device tokens are removed, future visits are cancelled with their notices, waitlist entries withdrawn and consents revoked (S5). Not reversible by the patient; the operator may reopen it within 30 days, after which what has no legal basis to stay is erased and the clinical record stays with each clinic, restricted (S6). Confirmed with the password. For a patient. Built in package D3. Audited.
  ///
  /// Parameters:
  ///
  /// * [AccountClosure] accountClosure (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<void> closeMyAccount(
    AccountClosure accountClosure, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await closeMyAccountWithHttpInfo(
      accountClosure,
      xSalusCSRF: xSalusCSRF,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
  }

  /// Download a ready export
  ///
  /// A zip archive of FHIR Bundles. For the patient who filed the request, until it expires. Built in package D2. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Response> downloadMyDataRequestExportWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path =
        r'/me/data-requests/{id}/export/archive'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    const contentTypes = <String>[];

    return apiClient.invokeAPI(
      path,
      'GET',
      queryParams,
      postBody,
      headerParams,
      formParams,
      contentTypes.isEmpty ? null : contentTypes.first,
      abortTrigger: abortTrigger,
    );
  }

  /// Download a ready export
  ///
  /// A zip archive of FHIR Bundles. For the patient who filed the request, until it expires. Built in package D2. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<String?> downloadMyDataRequestExport(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await downloadMyDataRequestExportWithHttpInfo(
      id,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
    // When a remote server returns no body with a status of 204, we shall not decode it.
    // At the time of writing this, `dart:convert` will throw an "Unexpected end of input"
    // FormatException when trying to decode an empty string.
    if (response.body.isNotEmpty &&
        response.statusCode != HttpStatus.noContent) {
      return await apiClient.deserializeAsync(
        await _decodeBodyBytes(response),
        'String',
      ) as String;
    }
    return null;
  }

  /// Extend a request's deadline once
  ///
  /// By two months, with a reason told to the patient (GDPR Art. 12(3); S3). For the clinic's admin, once, while the request is open. Built in package D1. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [DataRequestExtension] dataRequestExtension (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> extendClinicDataRequestWithHttpInfo(
    String id,
    String ifMatch,
    DataRequestExtension dataRequestExtension, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/data-requests/{id}/extend'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = dataRequestExtension;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    headerParams[r'If-Match'] = parameterToString(ifMatch);
    if (xSalusCSRF != null) {
      headerParams[r'X-Salus-CSRF'] = parameterToString(xSalusCSRF);
    }

    const contentTypes = <String>['application/json'];

    return apiClient.invokeAPI(
      path,
      'POST',
      queryParams,
      postBody,
      headerParams,
      formParams,
      contentTypes.isEmpty ? null : contentTypes.first,
      abortTrigger: abortTrigger,
    );
  }

  /// Extend a request's deadline once
  ///
  /// By two months, with a reason told to the patient (GDPR Art. 12(3); S3). For the clinic's admin, once, while the request is open. Built in package D1. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [DataRequestExtension] dataRequestExtension (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<DataRequest?> extendClinicDataRequest(
    String id,
    String ifMatch,
    DataRequestExtension dataRequestExtension, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await extendClinicDataRequestWithHttpInfo(
      id,
      ifMatch,
      dataRequestExtension,
      xSalusCSRF: xSalusCSRF,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
    // When a remote server returns no body with a status of 204, we shall not decode it.
    // At the time of writing this, `dart:convert` will throw an "Unexpected end of input"
    // FormatException when trying to decode an empty string.
    if (response.body.isNotEmpty &&
        response.statusCode != HttpStatus.noContent) {
      return await apiClient.deserializeAsync(
        await _decodeBodyBytes(response),
        'DataRequest',
      ) as DataRequest;
    }
    return null;
  }

  /// File a request on behalf of a registered patient
  ///
  /// At the desk, after checking the patient's identity, for a patient the clinic registered (S10); the request targets the clinic and is handled as any other. For the clinic's admin and its receptionists. Built in package D1. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [FileClinicPatientDataRequestRequest] fileClinicPatientDataRequestRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> fileClinicPatientDataRequestWithHttpInfo(
    String id,
    FileClinicPatientDataRequestRequest fileClinicPatientDataRequestRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/patients/{id}/data-requests'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = fileClinicPatientDataRequestRequest;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    if (xSalusCSRF != null) {
      headerParams[r'X-Salus-CSRF'] = parameterToString(xSalusCSRF);
    }

    const contentTypes = <String>['application/json'];

    return apiClient.invokeAPI(
      path,
      'POST',
      queryParams,
      postBody,
      headerParams,
      formParams,
      contentTypes.isEmpty ? null : contentTypes.first,
      abortTrigger: abortTrigger,
    );
  }

  /// File a request on behalf of a registered patient
  ///
  /// At the desk, after checking the patient's identity, for a patient the clinic registered (S10); the request targets the clinic and is handled as any other. For the clinic's admin and its receptionists. Built in package D1. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [FileClinicPatientDataRequestRequest] fileClinicPatientDataRequestRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<DataRequest?> fileClinicPatientDataRequest(
    String id,
    FileClinicPatientDataRequestRequest fileClinicPatientDataRequestRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await fileClinicPatientDataRequestWithHttpInfo(
      id,
      fileClinicPatientDataRequestRequest,
      xSalusCSRF: xSalusCSRF,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
    // When a remote server returns no body with a status of 204, we shall not decode it.
    // At the time of writing this, `dart:convert` will throw an "Unexpected end of input"
    // FormatException when trying to decode an empty string.
    if (response.body.isNotEmpty &&
        response.statusCode != HttpStatus.noContent) {
      return await apiClient.deserializeAsync(
        await _decodeBodyBytes(response),
        'DataRequest',
      ) as DataRequest;
    }
    return null;
  }

  /// File a data subject request
  ///
  /// A request of a kind (S2) to the patient's Salus account, which Salus answers, or to their record at one clinic they have a record with, which that clinic's admin answers (S1). Due one month after receipt (S3). For a patient. Built in package D1. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [DataRequestFiling] dataRequestFiling (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> fileMyDataRequestWithHttpInfo(
    DataRequestFiling dataRequestFiling, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/me/data-requests';

    // ignore: prefer_final_locals
    Object? postBody = dataRequestFiling;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    if (xSalusCSRF != null) {
      headerParams[r'X-Salus-CSRF'] = parameterToString(xSalusCSRF);
    }

    const contentTypes = <String>['application/json'];

    return apiClient.invokeAPI(
      path,
      'POST',
      queryParams,
      postBody,
      headerParams,
      formParams,
      contentTypes.isEmpty ? null : contentTypes.first,
      abortTrigger: abortTrigger,
    );
  }

  /// File a data subject request
  ///
  /// A request of a kind (S2) to the patient's Salus account, which Salus answers, or to their record at one clinic they have a record with, which that clinic's admin answers (S1). Due one month after receipt (S3). For a patient. Built in package D1. Audited.
  ///
  /// Parameters:
  ///
  /// * [DataRequestFiling] dataRequestFiling (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<DataRequest?> fileMyDataRequest(
    DataRequestFiling dataRequestFiling, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await fileMyDataRequestWithHttpInfo(
      dataRequestFiling,
      xSalusCSRF: xSalusCSRF,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
    // When a remote server returns no body with a status of 204, we shall not decode it.
    // At the time of writing this, `dart:convert` will throw an "Unexpected end of input"
    // FormatException when trying to decode an empty string.
    if (response.body.isNotEmpty &&
        response.statusCode != HttpStatus.noContent) {
      return await apiClient.deserializeAsync(
        await _decodeBodyBytes(response),
        'DataRequest',
      ) as DataRequest;
    }
    return null;
  }

  /// One request targeting the clinic
  ///
  /// With its history of status changes. For the clinic's admin. Built in package D1. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Response> getClinicDataRequestWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/data-requests/{id}'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    const contentTypes = <String>[];

    return apiClient.invokeAPI(
      path,
      'GET',
      queryParams,
      postBody,
      headerParams,
      formParams,
      contentTypes.isEmpty ? null : contentTypes.first,
      abortTrigger: abortTrigger,
    );
  }

  /// One request targeting the clinic
  ///
  /// With its history of status changes. For the clinic's admin. Built in package D1. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<DataRequest?> getClinicDataRequest(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getClinicDataRequestWithHttpInfo(
      id,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
    // When a remote server returns no body with a status of 204, we shall not decode it.
    // At the time of writing this, `dart:convert` will throw an "Unexpected end of input"
    // FormatException when trying to decode an empty string.
    if (response.body.isNotEmpty &&
        response.statusCode != HttpStatus.noContent) {
      return await apiClient.deserializeAsync(
        await _decodeBodyBytes(response),
        'DataRequest',
      ) as DataRequest;
    }
    return null;
  }

  /// Where the export of an access or portability request stands
  ///
  /// Everything the patient can read in the app, from every clinic of the target, plus their account data for the account target, as FHIR Bundles in one encrypted archive built by a job (S4). Downloadable for 7 days once ready. For the patient who filed the request. Built in package D2. Audited, as a read of the patient's request.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Response> getMyDataRequestExportWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/me/data-requests/{id}/export'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    const contentTypes = <String>[];

    return apiClient.invokeAPI(
      path,
      'GET',
      queryParams,
      postBody,
      headerParams,
      formParams,
      contentTypes.isEmpty ? null : contentTypes.first,
      abortTrigger: abortTrigger,
    );
  }

  /// Where the export of an access or portability request stands
  ///
  /// Everything the patient can read in the app, from every clinic of the target, plus their account data for the account target, as FHIR Bundles in one encrypted archive built by a job (S4). Downloadable for 7 days once ready. For the patient who filed the request. Built in package D2. Audited, as a read of the patient's request.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<DataRequestExport?> getMyDataRequestExport(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getMyDataRequestExportWithHttpInfo(
      id,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
    // When a remote server returns no body with a status of 204, we shall not decode it.
    // At the time of writing this, `dart:convert` will throw an "Unexpected end of input"
    // FormatException when trying to decode an empty string.
    if (response.body.isNotEmpty &&
        response.statusCode != HttpStatus.noContent) {
      return await apiClient.deserializeAsync(
        await _decodeBodyBytes(response),
        'DataRequestExport',
      ) as DataRequestExport;
    }
    return null;
  }

  /// Lift a restriction
  ///
  /// Lifting a record that is not restricted changes nothing. For the clinic's admin. Built in package D4. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> liftClinicPatientRestrictionWithHttpInfo(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/patients/{id}/restriction'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    if (xSalusCSRF != null) {
      headerParams[r'X-Salus-CSRF'] = parameterToString(xSalusCSRF);
    }

    const contentTypes = <String>[];

    return apiClient.invokeAPI(
      path,
      'DELETE',
      queryParams,
      postBody,
      headerParams,
      formParams,
      contentTypes.isEmpty ? null : contentTypes.first,
      abortTrigger: abortTrigger,
    );
  }

  /// Lift a restriction
  ///
  /// Lifting a record that is not restricted changes nothing. For the clinic's admin. Built in package D4. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<void> liftClinicPatientRestriction(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await liftClinicPatientRestrictionWithHttpInfo(
      id,
      xSalusCSRF: xSalusCSRF,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
  }

  /// The requests targeting the clinic
  ///
  /// The open ones by due date first, then the answered and withdrawn ones (S1, S3). For the clinic's admin. Built in package D1. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> listClinicDataRequestsWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/data-requests';

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    const contentTypes = <String>[];

    return apiClient.invokeAPI(
      path,
      'GET',
      queryParams,
      postBody,
      headerParams,
      formParams,
      contentTypes.isEmpty ? null : contentTypes.first,
      abortTrigger: abortTrigger,
    );
  }

  /// The requests targeting the clinic
  ///
  /// The open ones by due date first, then the answered and withdrawn ones (S1, S3). For the clinic's admin. Built in package D1. Audited.
  Future<List<DataRequest>?> listClinicDataRequests({
    Future<void>? abortTrigger,
  }) async {
    final response = await listClinicDataRequestsWithHttpInfo(
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
    // When a remote server returns no body with a status of 204, we shall not decode it.
    // At the time of writing this, `dart:convert` will throw an "Unexpected end of input"
    // FormatException when trying to decode an empty string.
    if (response.body.isNotEmpty &&
        response.statusCode != HttpStatus.noContent) {
      final responseBody = await _decodeBodyBytes(response);
      return (await apiClient.deserializeAsync(
              responseBody, 'List<DataRequest>') as List)
          .cast<DataRequest>()
          .toList(growable: false);
    }
    return null;
  }

  /// The patient's own data subject requests
  ///
  /// Every request the patient filed, to their account or to one of their clinics, newest first, with its state and due date (docs/clinic-cms WAVE5.md, S1 to S3). For a patient. Built in package D1. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> listMyDataRequestsWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/me/data-requests';

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    const contentTypes = <String>[];

    return apiClient.invokeAPI(
      path,
      'GET',
      queryParams,
      postBody,
      headerParams,
      formParams,
      contentTypes.isEmpty ? null : contentTypes.first,
      abortTrigger: abortTrigger,
    );
  }

  /// The patient's own data subject requests
  ///
  /// Every request the patient filed, to their account or to one of their clinics, newest first, with its state and due date (docs/clinic-cms WAVE5.md, S1 to S3). For a patient. Built in package D1. Audited.
  Future<List<DataRequest>?> listMyDataRequests({
    Future<void>? abortTrigger,
  }) async {
    final response = await listMyDataRequestsWithHttpInfo(
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
    // When a remote server returns no body with a status of 204, we shall not decode it.
    // At the time of writing this, `dart:convert` will throw an "Unexpected end of input"
    // FormatException when trying to decode an empty string.
    if (response.body.isNotEmpty &&
        response.statusCode != HttpStatus.noContent) {
      final responseBody = await _decodeBodyBytes(response);
      return (await apiClient.deserializeAsync(
              responseBody, 'List<DataRequest>') as List)
          .cast<DataRequest>()
          .toList(growable: false);
    }
    return null;
  }

  /// Restrict a patient's record at the clinic
  ///
  /// Hidden from the clinic's everyday screens and searches, and never used for analytics, reminders or messages; the admin opens it only with a stated reason (S7). Restricting a restricted record changes nothing. For the clinic's admin. Built in package D4. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  ///
  /// * [RestrictClinicPatientRequest] restrictClinicPatientRequest:
  Future<Response> restrictClinicPatientWithHttpInfo(
    String id, {
    String? xSalusCSRF,
    RestrictClinicPatientRequest? restrictClinicPatientRequest,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/patients/{id}/restriction'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = restrictClinicPatientRequest;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    if (xSalusCSRF != null) {
      headerParams[r'X-Salus-CSRF'] = parameterToString(xSalusCSRF);
    }

    const contentTypes = <String>['application/json'];

    return apiClient.invokeAPI(
      path,
      'PUT',
      queryParams,
      postBody,
      headerParams,
      formParams,
      contentTypes.isEmpty ? null : contentTypes.first,
      abortTrigger: abortTrigger,
    );
  }

  /// Restrict a patient's record at the clinic
  ///
  /// Hidden from the clinic's everyday screens and searches, and never used for analytics, reminders or messages; the admin opens it only with a stated reason (S7). Restricting a restricted record changes nothing. For the clinic's admin. Built in package D4. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  ///
  /// * [RestrictClinicPatientRequest] restrictClinicPatientRequest:
  Future<void> restrictClinicPatient(
    String id, {
    String? xSalusCSRF,
    RestrictClinicPatientRequest? restrictClinicPatientRequest,
    Future<void>? abortTrigger,
  }) async {
    final response = await restrictClinicPatientWithHttpInfo(
      id,
      xSalusCSRF: xSalusCSRF,
      restrictClinicPatientRequest: restrictClinicPatientRequest,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
  }

  /// Withdraw an open request
  ///
  /// The request is withdrawn and keeps its history; it is never deleted. For the patient who filed it, while it is received or in progress. Built in package D1. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> withdrawMyDataRequestWithHttpInfo(
    String id,
    String ifMatch, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/me/data-requests/{id}/withdraw'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    headerParams[r'If-Match'] = parameterToString(ifMatch);
    if (xSalusCSRF != null) {
      headerParams[r'X-Salus-CSRF'] = parameterToString(xSalusCSRF);
    }

    const contentTypes = <String>[];

    return apiClient.invokeAPI(
      path,
      'POST',
      queryParams,
      postBody,
      headerParams,
      formParams,
      contentTypes.isEmpty ? null : contentTypes.first,
      abortTrigger: abortTrigger,
    );
  }

  /// Withdraw an open request
  ///
  /// The request is withdrawn and keeps its history; it is never deleted. For the patient who filed it, while it is received or in progress. Built in package D1. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<DataRequest?> withdrawMyDataRequest(
    String id,
    String ifMatch, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await withdrawMyDataRequestWithHttpInfo(
      id,
      ifMatch,
      xSalusCSRF: xSalusCSRF,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
    // When a remote server returns no body with a status of 204, we shall not decode it.
    // At the time of writing this, `dart:convert` will throw an "Unexpected end of input"
    // FormatException when trying to decode an empty string.
    if (response.body.isNotEmpty &&
        response.statusCode != HttpStatus.noContent) {
      return await apiClient.deserializeAsync(
        await _decodeBodyBytes(response),
        'DataRequest',
      ) as DataRequest;
    }
    return null;
  }
}
