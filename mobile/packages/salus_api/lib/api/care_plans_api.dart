//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class CarePlansApi {
  CarePlansApi([ApiClient? apiClient])
      : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// Book every step of a treatment plan
  ///
  /// One free slot per step, in the steps' order, in one transaction: every visit is booked or none is. The plan becomes `active` and the patient sees it. By the clinic's admin or a practitioner, at the version of If-Match. Built in package TP1. Audited, also when refused.
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
  /// * [BookCarePlanRequest] bookCarePlanRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> bookCarePlanWithHttpInfo(
    String ifMatch,
    String id,
    BookCarePlanRequest bookCarePlanRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/care-plans/{id}/book'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = bookCarePlanRequest;

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
      'POST',
      queryParams,
      postBody,
      headerParams,
      formParams,
      contentTypes.isEmpty ? null : contentTypes.first,
      abortTrigger: abortTrigger,
    );
  }

  /// Book every step of a treatment plan
  ///
  /// One free slot per step, in the steps' order, in one transaction: every visit is booked or none is. The plan becomes `active` and the patient sees it. By the clinic's admin or a practitioner, at the version of If-Match. Built in package TP1. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///
  /// * [BookCarePlanRequest] bookCarePlanRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<CarePlan?> bookCarePlan(
    String ifMatch,
    String id,
    BookCarePlanRequest bookCarePlanRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await bookCarePlanWithHttpInfo(
      ifMatch,
      id,
      bookCarePlanRequest,
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
        'CarePlan',
      ) as CarePlan;
    }
    return null;
  }

  /// Cancel a treatment plan
  ///
  /// The plan becomes `revoked`; its steps' future visits are cancelled (by staff, `prov`), its past ones kept. By the clinic's admin or a practitioner, at the version of If-Match. Built in package TP1. Audited, also when refused.
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
  Future<Response> cancelCarePlanWithHttpInfo(
    String ifMatch,
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/care-plans/{id}/cancel'.replaceAll('{id}', id);

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

  /// Cancel a treatment plan
  ///
  /// The plan becomes `revoked`; its steps' future visits are cancelled (by staff, `prov`), its past ones kept. By the clinic's admin or a practitioner, at the version of If-Match. Built in package TP1. Audited, also when refused.
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
  Future<CarePlan?> cancelCarePlan(
    String ifMatch,
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await cancelCarePlanWithHttpInfo(
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
        'CarePlan',
      ) as CarePlan;
    }
    return null;
  }

  /// A treatment plan of the clinic
  ///
  /// Its steps in the order of their visits once booked (a moved step shows in its new place). Built in package TP1. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Response> getClinicCarePlanWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/care-plans/{id}'.replaceAll('{id}', id);

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

  /// A treatment plan of the clinic
  ///
  /// Its steps in the order of their visits once booked (a moved step shows in its new place). Built in package TP1. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<CarePlan?> getClinicCarePlan(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getClinicCarePlanWithHttpInfo(
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
        'CarePlan',
      ) as CarePlan;
    }
    return null;
  }

  /// A patient's treatment plans at the clinic
  ///
  /// Newest first, drafts included. For the clinic's admin and its practitioners; never a receptionist. Built in package TP1. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Response> listClinicPatientCarePlansWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/patients/{id}/care-plans'.replaceAll('{id}', id);

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

  /// A patient's treatment plans at the clinic
  ///
  /// Newest first, drafts included. For the clinic's admin and its practitioners; never a receptionist. Built in package TP1. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<List<CarePlan>?> listClinicPatientCarePlans(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await listClinicPatientCarePlansWithHttpInfo(
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
      return (await apiClient.deserializeAsync(responseBody, 'List<CarePlan>')
              as List)
          .cast<CarePlan>()
          .toList(growable: false);
    }
    return null;
  }

  /// The patient's own treatment plans
  ///
  /// Booked ones from every clinic, newest first, never a draft. Built in package TP1. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> listMyCarePlansWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/me/care-plans';

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

  /// The patient's own treatment plans
  ///
  /// Booked ones from every clinic, newest first, never a draft. Built in package TP1. Audited.
  Future<List<CarePlan>?> listMyCarePlans({
    Future<void>? abortTrigger,
  }) async {
    final response = await listMyCarePlansWithHttpInfo(
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
      return (await apiClient.deserializeAsync(responseBody, 'List<CarePlan>')
              as List)
          .cast<CarePlan>()
          .toList(growable: false);
    }
    return null;
  }

  /// Change a draft treatment plan
  ///
  /// Its title, quote and every step, at the version of If-Match. Once booked, a step is moved by moving its visit (`POST /clinic/appointments/{id}/reschedule`) and the plan shows the new order. Built in package TP1. Audited, also when refused.
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
  /// * [SaveCarePlanRequest] saveCarePlanRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> updateClinicCarePlanWithHttpInfo(
    String ifMatch,
    String id,
    SaveCarePlanRequest saveCarePlanRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/care-plans/{id}'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = saveCarePlanRequest;

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

  /// Change a draft treatment plan
  ///
  /// Its title, quote and every step, at the version of If-Match. Once booked, a step is moved by moving its visit (`POST /clinic/appointments/{id}/reschedule`) and the plan shows the new order. Built in package TP1. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///
  /// * [SaveCarePlanRequest] saveCarePlanRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<CarePlan?> updateClinicCarePlan(
    String ifMatch,
    String id,
    SaveCarePlanRequest saveCarePlanRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await updateClinicCarePlanWithHttpInfo(
      ifMatch,
      id,
      saveCarePlanRequest,
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
        'CarePlan',
      ) as CarePlan;
    }
    return null;
  }

  /// Lay out a treatment plan
  ///
  /// A draft for a patient with the clinic's active care consent: ordered steps, each a service of the price list, a day offset and a visit mode, and optionally its quote (U6 of docs/clinic-cms/WAVE6.md). By the clinic's admin or a practitioner. The patient does not see a draft. Built in package TP1. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [SaveCarePlanRequest] saveCarePlanRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> writeCarePlanWithHttpInfo(
    SaveCarePlanRequest saveCarePlanRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/care-plans';

    // ignore: prefer_final_locals
    Object? postBody = saveCarePlanRequest;

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

  /// Lay out a treatment plan
  ///
  /// A draft for a patient with the clinic's active care consent: ordered steps, each a service of the price list, a day offset and a visit mode, and optionally its quote (U6 of docs/clinic-cms/WAVE6.md). By the clinic's admin or a practitioner. The patient does not see a draft. Built in package TP1. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [SaveCarePlanRequest] saveCarePlanRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<CarePlan?> writeCarePlan(
    SaveCarePlanRequest saveCarePlanRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await writeCarePlanWithHttpInfo(
      saveCarePlanRequest,
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
        'CarePlan',
      ) as CarePlan;
    }
    return null;
  }
}
