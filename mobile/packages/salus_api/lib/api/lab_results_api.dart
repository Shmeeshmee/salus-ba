//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class LabResultsApi {
  LabResultsApi([ApiClient? apiClient])
      : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// Attach a lab result to a visit
  ///
  /// Values typed in (test, value, unit, reference range, a flag), the lab's PDF after it (`PUT …/pdf`), or both; `preliminary` until a practitioner releases it, and the patient does not see it before. For the visit's patient with the clinic's active care consent. By the clinic's admin or a practitioner who may see the visit, never a receptionist. Built in package LB1. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [SaveDiagnosticReportRequest] saveDiagnosticReportRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> attachDiagnosticReportWithHttpInfo(
    String id,
    SaveDiagnosticReportRequest saveDiagnosticReportRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path =
        r'/clinic/appointments/{id}/diagnostic-reports'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = saveDiagnosticReportRequest;

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

  /// Attach a lab result to a visit
  ///
  /// Values typed in (test, value, unit, reference range, a flag), the lab's PDF after it (`PUT …/pdf`), or both; `preliminary` until a practitioner releases it, and the patient does not see it before. For the visit's patient with the clinic's active care consent. By the clinic's admin or a practitioner who may see the visit, never a receptionist. Built in package LB1. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [SaveDiagnosticReportRequest] saveDiagnosticReportRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<DiagnosticReport?> attachDiagnosticReport(
    String id,
    SaveDiagnosticReportRequest saveDiagnosticReportRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await attachDiagnosticReportWithHttpInfo(
      id,
      saveDiagnosticReportRequest,
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
        'DiagnosticReport',
      ) as DiagnosticReport;
    }
    return null;
  }

  /// A lab result of the clinic
  ///
  /// Built in package LB1. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Response> getClinicDiagnosticReportWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/diagnostic-reports/{id}'.replaceAll('{id}', id);

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

  /// A lab result of the clinic
  ///
  /// Built in package LB1. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<DiagnosticReport?> getClinicDiagnosticReport(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getClinicDiagnosticReportWithHttpInfo(
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
        'DiagnosticReport',
      ) as DiagnosticReport;
    }
    return null;
  }

  /// A lab result's PDF
  ///
  /// The lab's PDF as attached, to the clinic's admin and the practitioners who may see the result. Built in package LB1. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Response> getClinicDiagnosticReportPdfWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/diagnostic-reports/{id}/pdf'.replaceAll('{id}', id);

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

  /// A lab result's PDF
  ///
  /// The lab's PDF as attached, to the clinic's admin and the practitioners who may see the result. Built in package LB1. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<String?> getClinicDiagnosticReportPdf(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getClinicDiagnosticReportPdfWithHttpInfo(
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

  /// One of the patient's own lab results
  ///
  /// Released only. Built in package LB1. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Response> getMyDiagnosticReportWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/me/diagnostic-reports/{id}'.replaceAll('{id}', id);

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

  /// One of the patient's own lab results
  ///
  /// Released only. Built in package LB1. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<DiagnosticReport?> getMyDiagnosticReport(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getMyDiagnosticReportWithHttpInfo(
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
        'DiagnosticReport',
      ) as DiagnosticReport;
    }
    return null;
  }

  /// The PDF of one of the patient's own lab results
  ///
  /// Released only. Built in package LB1. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Response> getMyDiagnosticReportPdfWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/me/diagnostic-reports/{id}/pdf'.replaceAll('{id}', id);

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

  /// The PDF of one of the patient's own lab results
  ///
  /// Released only. Built in package LB1. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<String?> getMyDiagnosticReportPdf(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getMyDiagnosticReportPdfWithHttpInfo(
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

  /// The patient's own lab results
  ///
  /// The released ones from every clinic, newest first. Built in package LB1. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> listMyDiagnosticReportsWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/me/diagnostic-reports';

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

  /// The patient's own lab results
  ///
  /// The released ones from every clinic, newest first. Built in package LB1. Audited.
  Future<List<DiagnosticReport>?> listMyDiagnosticReports({
    Future<void>? abortTrigger,
  }) async {
    final response = await listMyDiagnosticReportsWithHttpInfo(
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
              responseBody, 'List<DiagnosticReport>') as List)
          .cast<DiagnosticReport>()
          .toList(growable: false);
    }
    return null;
  }

  /// A visit's lab results
  ///
  /// Released or not, oldest first (U5 of docs/clinic-cms/WAVE6.md). For the clinic's admin and the practitioners who may see the visit; never a receptionist (no clinical content). Built in package LB1. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Response> listVisitDiagnosticReportsWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path =
        r'/clinic/appointments/{id}/diagnostic-reports'.replaceAll('{id}', id);

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

  /// A visit's lab results
  ///
  /// Released or not, oldest first (U5 of docs/clinic-cms/WAVE6.md). For the clinic's admin and the practitioners who may see the visit; never a receptionist (no clinical content). Built in package LB1. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<List<DiagnosticReport>?> listVisitDiagnosticReports(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await listVisitDiagnosticReportsWithHttpInfo(
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
      final responseBody = await _decodeBodyBytes(response);
      return (await apiClient.deserializeAsync(
              responseBody, 'List<DiagnosticReport>') as List)
          .cast<DiagnosticReport>()
          .toList(growable: false);
    }
    return null;
  }

  /// Attach the lab's PDF to a result
  ///
  /// Before its release, at the version of If-Match; replaces a PDF already there. Scanned and encrypted as every Binary (docs/attachments). Built in package LB1. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///
  /// * [int] contentLength (required):
  ///   The file's size in bytes.
  ///
  /// * [MultipartFile] body (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> putClinicDiagnosticReportPdfWithHttpInfo(
    String ifMatch,
    String id,
    int contentLength,
    MultipartFile body, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/diagnostic-reports/{id}/pdf'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = body;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    if (xSalusCSRF != null) {
      headerParams[r'X-Salus-CSRF'] = parameterToString(xSalusCSRF);
    }
    headerParams[r'If-Match'] = parameterToString(ifMatch);
    headerParams[r'Content-Length'] = parameterToString(contentLength);

    const contentTypes = <String>['application/pdf'];

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

  /// Attach the lab's PDF to a result
  ///
  /// Before its release, at the version of If-Match; replaces a PDF already there. Scanned and encrypted as every Binary (docs/attachments). Built in package LB1. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///
  /// * [int] contentLength (required):
  ///   The file's size in bytes.
  ///
  /// * [MultipartFile] body (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<DiagnosticReport?> putClinicDiagnosticReportPdf(
    String ifMatch,
    String id,
    int contentLength,
    MultipartFile body, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await putClinicDiagnosticReportPdfWithHttpInfo(
      ifMatch,
      id,
      contentLength,
      body,
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
        'DiagnosticReport',
      ) as DiagnosticReport;
    }
    return null;
  }

  /// Release a lab result to the patient
  ///
  /// The practitioner's review: the result becomes `final`, never changes after, and the patient sees it in the app. By a practitioner of the clinic who may see the visit, at the version of If-Match. Built in package LB1. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> releaseDiagnosticReportWithHttpInfo(
    String ifMatch,
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path =
        r'/clinic/diagnostic-reports/{id}/release'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    if (xSalusCSRF != null) {
      headerParams[r'X-Salus-CSRF'] = parameterToString(xSalusCSRF);
    }
    headerParams[r'If-Match'] = parameterToString(ifMatch);

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

  /// Release a lab result to the patient
  ///
  /// The practitioner's review: the result becomes `final`, never changes after, and the patient sees it in the app. By a practitioner of the clinic who may see the visit, at the version of If-Match. Built in package LB1. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<DiagnosticReport?> releaseDiagnosticReport(
    String ifMatch,
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await releaseDiagnosticReportWithHttpInfo(
      ifMatch,
      id,
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
        'DiagnosticReport',
      ) as DiagnosticReport;
    }
    return null;
  }

  /// Change a lab result before its release
  ///
  /// Its title, code, date and every value, at the version of If-Match. A released result never changes. Built in package LB1. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///
  /// * [SaveDiagnosticReportRequest] saveDiagnosticReportRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> updateClinicDiagnosticReportWithHttpInfo(
    String ifMatch,
    String id,
    SaveDiagnosticReportRequest saveDiagnosticReportRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/diagnostic-reports/{id}'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = saveDiagnosticReportRequest;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    if (xSalusCSRF != null) {
      headerParams[r'X-Salus-CSRF'] = parameterToString(xSalusCSRF);
    }
    headerParams[r'If-Match'] = parameterToString(ifMatch);

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

  /// Change a lab result before its release
  ///
  /// Its title, code, date and every value, at the version of If-Match. A released result never changes. Built in package LB1. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///
  /// * [SaveDiagnosticReportRequest] saveDiagnosticReportRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<DiagnosticReport?> updateClinicDiagnosticReport(
    String ifMatch,
    String id,
    SaveDiagnosticReportRequest saveDiagnosticReportRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await updateClinicDiagnosticReportWithHttpInfo(
      ifMatch,
      id,
      saveDiagnosticReportRequest,
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
        'DiagnosticReport',
      ) as DiagnosticReport;
    }
    return null;
  }
}
