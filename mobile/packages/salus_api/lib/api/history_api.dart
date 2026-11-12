//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class HistoryApi {
  HistoryApi([ApiClient? apiClient])
      : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// Add a clinic's allergy to the profile
  ///
  /// SAL-41. The patient adds an entry a clinic recorded in one of their visits to their profile, so every clinic with their care consent sees it (without the clinic that recorded it). Send If-Match. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///   The entry's id (FHIR AllergyIntolerance.id or MedicationStatement.id).
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> addAllergyToProfileWithHttpInfo(
    String ifMatch,
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/allergies/{id}/profile'.replaceAll('{id}', id);

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

  /// Add a clinic's allergy to the profile
  ///
  /// SAL-41. The patient adds an entry a clinic recorded in one of their visits to their profile, so every clinic with their care consent sees it (without the clinic that recorded it). Send If-Match. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///   The entry's id (FHIR AllergyIntolerance.id or MedicationStatement.id).
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<EntryWritten?> addAllergyToProfile(
    String ifMatch,
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await addAllergyToProfileWithHttpInfo(
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
        'EntryWritten',
      ) as EntryWritten;
    }
    return null;
  }

  /// Add a clinic's medication to the profile
  ///
  /// SAL-41. As `POST /allergies/{id}/profile`, for a medication.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///   The entry's id (FHIR AllergyIntolerance.id or MedicationStatement.id).
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> addMedicationToProfileWithHttpInfo(
    String ifMatch,
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/medications/{id}/profile'.replaceAll('{id}', id);

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

  /// Add a clinic's medication to the profile
  ///
  /// SAL-41. As `POST /allergies/{id}/profile`, for a medication.
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///   The entry's id (FHIR AllergyIntolerance.id or MedicationStatement.id).
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<EntryWritten?> addMedicationToProfile(
    String ifMatch,
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await addMedicationToProfileWithHttpInfo(
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
        'EntryWritten',
      ) as EntryWritten;
    }
    return null;
  }

  /// Add an allergy to the signed-in patient's profile
  ///
  /// SAL-41. Stored as an unconfirmed FHIR AllergyIntolerance in the patient's profile, seen by every clinic with the patient's active care consent and an appointment. Read it through the FHIR API (`GET /fhir/R4/AllergyIntolerance?patient=…`). Patients only. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [AllergyIntoleranceInput] allergyIntoleranceInput (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> addMyAllergyWithHttpInfo(
    AllergyIntoleranceInput allergyIntoleranceInput, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/me/allergies';

    // ignore: prefer_final_locals
    Object? postBody = allergyIntoleranceInput;

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

  /// Add an allergy to the signed-in patient's profile
  ///
  /// SAL-41. Stored as an unconfirmed FHIR AllergyIntolerance in the patient's profile, seen by every clinic with the patient's active care consent and an appointment. Read it through the FHIR API (`GET /fhir/R4/AllergyIntolerance?patient=…`). Patients only. Audited.
  ///
  /// Parameters:
  ///
  /// * [AllergyIntoleranceInput] allergyIntoleranceInput (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<EntryWritten?> addMyAllergy(
    AllergyIntoleranceInput allergyIntoleranceInput, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await addMyAllergyWithHttpInfo(
      allergyIntoleranceInput,
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
        'EntryWritten',
      ) as EntryWritten;
    }
    return null;
  }

  /// Add a medication to the signed-in patient's profile
  ///
  /// SAL-41. As `POST /me/allergies`, for a FHIR MedicationStatement (`GET /fhir/R4/MedicationStatement?patient=…`).
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [MedicationStatementInput] medicationStatementInput (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> addMyMedicationWithHttpInfo(
    MedicationStatementInput medicationStatementInput, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/me/medications';

    // ignore: prefer_final_locals
    Object? postBody = medicationStatementInput;

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

  /// Add a medication to the signed-in patient's profile
  ///
  /// SAL-41. As `POST /me/allergies`, for a FHIR MedicationStatement (`GET /fhir/R4/MedicationStatement?patient=…`).
  ///
  /// Parameters:
  ///
  /// * [MedicationStatementInput] medicationStatementInput (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<EntryWritten?> addMyMedication(
    MedicationStatementInput medicationStatementInput, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await addMyMedicationWithHttpInfo(
      medicationStatementInput,
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
        'EntryWritten',
      ) as EntryWritten;
    }
    return null;
  }

  /// Confirm an allergy
  ///
  /// SAL-41. A practitioner who may see the entry confirms it during a visit. Other clinics see \"confirmed by a clinician\" without the clinic or the practitioner. Send If-Match. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///   The entry's id (FHIR AllergyIntolerance.id or MedicationStatement.id).
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> confirmAllergyWithHttpInfo(
    String ifMatch,
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/allergies/{id}/confirm'.replaceAll('{id}', id);

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

  /// Confirm an allergy
  ///
  /// SAL-41. A practitioner who may see the entry confirms it during a visit. Other clinics see \"confirmed by a clinician\" without the clinic or the practitioner. Send If-Match. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///   The entry's id (FHIR AllergyIntolerance.id or MedicationStatement.id).
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<EntryWritten?> confirmAllergy(
    String ifMatch,
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await confirmAllergyWithHttpInfo(
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
        'EntryWritten',
      ) as EntryWritten;
    }
    return null;
  }

  /// Confirm a medication
  ///
  /// SAL-41. As `POST /allergies/{id}/confirm`, for a medication.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///   The entry's id (FHIR AllergyIntolerance.id or MedicationStatement.id).
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> confirmMedicationWithHttpInfo(
    String ifMatch,
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/medications/{id}/confirm'.replaceAll('{id}', id);

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

  /// Confirm a medication
  ///
  /// SAL-41. As `POST /allergies/{id}/confirm`, for a medication.
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///   The entry's id (FHIR AllergyIntolerance.id or MedicationStatement.id).
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<EntryWritten?> confirmMedication(
    String ifMatch,
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await confirmMedicationWithHttpInfo(
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
        'EntryWritten',
      ) as EntryWritten;
    }
    return null;
  }

  /// The pre-visit summary of a patient
  ///
  /// SAL-40. The allergies and current medications the practitioner may see, the latest visit they may see, and what is open for the appointment (a questionnaire not completed, a document not signed). Same access as the history. Not cached. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] appointment (required):
  ///   The appointment the summary is for, one of the practitioner's with the patient.
  ///
  /// * [String] id (required):
  ///   The patient's id (FHIR Patient.id).
  Future<Response> getPatientHistorySummaryWithHttpInfo(
    String appointment,
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/patients/{id}/history/summary'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    queryParams.addAll(_queryParams('', 'appointment', appointment));

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

  /// The pre-visit summary of a patient
  ///
  /// SAL-40. The allergies and current medications the practitioner may see, the latest visit they may see, and what is open for the appointment (a questionnaire not completed, a document not signed). Same access as the history. Not cached. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] appointment (required):
  ///   The appointment the summary is for, one of the practitioner's with the patient.
  ///
  /// * [String] id (required):
  ///   The patient's id (FHIR Patient.id).
  Future<HistorySummary?> getPatientHistorySummary(
    String appointment,
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getPatientHistorySummaryWithHttpInfo(
      appointment,
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
        'HistorySummary',
      ) as HistorySummary;
    }
    return null;
  }

  /// The signed-in patient's history across every clinic
  ///
  /// SAL-39. The patient's own visits, signed documents, completed questionnaire answers and their files, from every clinic, newest first, one page at a time. Never a draft, a response in progress or a file not yet scanned. Chats are not listed yet (item type `communication` is reserved for them). Patients only. Not cached. Each item returned is audited by its owning domain.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] cursor:
  ///   The `next` of the previous page; absent for the first.
  ///
  /// * [int] limit:
  ///   Items per page.
  ///
  /// * [String] organization:
  ///   Only this clinic's items.
  ///
  /// * [List<HistoryItemType>] type:
  ///   Only items of these types (repeat the parameter for several).
  ///
  /// * [DateTime] from:
  ///   Only items on or after this instant.
  ///
  /// * [DateTime] to:
  ///   Only items before this instant.
  Future<Response> listMyHistoryWithHttpInfo({
    String? cursor,
    int? limit,
    String? organization,
    List<HistoryItemType>? type,
    DateTime? from,
    DateTime? to,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/me/history';

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    if (cursor != null) {
      queryParams.addAll(_queryParams('', 'cursor', cursor));
    }
    if (limit != null) {
      queryParams.addAll(_queryParams('', 'limit', limit));
    }
    if (organization != null) {
      queryParams.addAll(_queryParams('', 'organization', organization));
    }
    if (type != null) {
      queryParams.addAll(_queryParams('multi', 'type', type));
    }
    if (from != null) {
      queryParams.addAll(_queryParams('', 'from', from));
    }
    if (to != null) {
      queryParams.addAll(_queryParams('', 'to', to));
    }

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

  /// The signed-in patient's history across every clinic
  ///
  /// SAL-39. The patient's own visits, signed documents, completed questionnaire answers and their files, from every clinic, newest first, one page at a time. Never a draft, a response in progress or a file not yet scanned. Chats are not listed yet (item type `communication` is reserved for them). Patients only. Not cached. Each item returned is audited by its owning domain.
  ///
  /// Parameters:
  ///
  /// * [String] cursor:
  ///   The `next` of the previous page; absent for the first.
  ///
  /// * [int] limit:
  ///   Items per page.
  ///
  /// * [String] organization:
  ///   Only this clinic's items.
  ///
  /// * [List<HistoryItemType>] type:
  ///   Only items of these types (repeat the parameter for several).
  ///
  /// * [DateTime] from:
  ///   Only items on or after this instant.
  ///
  /// * [DateTime] to:
  ///   Only items before this instant.
  Future<HistoryPage?> listMyHistory({
    String? cursor,
    int? limit,
    String? organization,
    List<HistoryItemType>? type,
    DateTime? from,
    DateTime? to,
    Future<void>? abortTrigger,
  }) async {
    final response = await listMyHistoryWithHttpInfo(
      cursor: cursor,
      limit: limit,
      organization: organization,
      type: type,
      from: from,
      to: to,
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
        'HistoryPage',
      ) as HistoryPage;
    }
    return null;
  }

  /// The clinics the signed-in patient shares history with
  ///
  /// SAL-42. Every clinic with an active history consent of the patient's, with the visits it lists now. Patients only. Not cached. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> listMyHistorySharingWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/me/history-sharing';

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

  /// The clinics the signed-in patient shares history with
  ///
  /// SAL-42. Every clinic with an active history consent of the patient's, with the visits it lists now. Patients only. Not cached. Audited.
  Future<HistorySharingList?> listMyHistorySharing({
    Future<void>? abortTrigger,
  }) async {
    final response = await listMyHistorySharingWithHttpInfo(
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
        'HistorySharingList',
      ) as HistorySharingList;
    }
    return null;
  }

  /// A patient's history as the practitioner may see it
  ///
  /// SAL-40. For a practitioner of the caller's clinic with an appointment with the patient and the patient's active care consent (ADR 0013): the visits at this clinic the practitioner held themselves, and the visits at other clinics the patient shared with this clinic (`shared`, marked with their clinic), with what belongs to each, newest first. Nothing else exists for the caller: an unshared visit is not listed, counted or hinted at. 404 to anyone else, the clinic's admin included. Not cached. Every view is audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The patient's id (FHIR Patient.id).
  ///
  /// * [String] cursor:
  ///   The `next` of the previous page; absent for the first.
  ///
  /// * [int] limit:
  ///   Items per page.
  Future<Response> listPatientHistoryWithHttpInfo(
    String id, {
    String? cursor,
    int? limit,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/patients/{id}/history'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    if (cursor != null) {
      queryParams.addAll(_queryParams('', 'cursor', cursor));
    }
    if (limit != null) {
      queryParams.addAll(_queryParams('', 'limit', limit));
    }

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

  /// A patient's history as the practitioner may see it
  ///
  /// SAL-40. For a practitioner of the caller's clinic with an appointment with the patient and the patient's active care consent (ADR 0013): the visits at this clinic the practitioner held themselves, and the visits at other clinics the patient shared with this clinic (`shared`, marked with their clinic), with what belongs to each, newest first. Nothing else exists for the caller: an unshared visit is not listed, counted or hinted at. 404 to anyone else, the clinic's admin included. Not cached. Every view is audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The patient's id (FHIR Patient.id).
  ///
  /// * [String] cursor:
  ///   The `next` of the previous page; absent for the first.
  ///
  /// * [int] limit:
  ///   Items per page.
  Future<HistoryPage?> listPatientHistory(
    String id, {
    String? cursor,
    int? limit,
    Future<void>? abortTrigger,
  }) async {
    final response = await listPatientHistoryWithHttpInfo(
      id,
      cursor: cursor,
      limit: limit,
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
        'HistoryPage',
      ) as HistoryPage;
    }
    return null;
  }

  /// The visits the patient may share with a clinic
  ///
  /// SAL-42. The preview at booking and in settings: every finished visit of the patient's at other clinics, each exactly as the clinic would see it marked (date, clinic, specialty, title), with whether it is shared now. Patients only. Not cached. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] organizationId (required):
  ///   The receiving clinic (FHIR Organization.id).
  Future<Response> previewHistorySharingWithHttpInfo(
    String organizationId, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/history-sharing/{organizationId}'
        .replaceAll('{organizationId}', organizationId);

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

  /// The visits the patient may share with a clinic
  ///
  /// SAL-42. The preview at booking and in settings: every finished visit of the patient's at other clinics, each exactly as the clinic would see it marked (date, clinic, specialty, title), with whether it is shared now. Patients only. Not cached. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] organizationId (required):
  ///   The receiving clinic (FHIR Organization.id).
  Future<SharingCandidateList?> previewHistorySharing(
    String organizationId, {
    Future<void>? abortTrigger,
  }) async {
    final response = await previewHistorySharingWithHttpInfo(
      organizationId,
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
        'SharingCandidateList',
      ) as SharingCandidateList;
    }
    return null;
  }

  /// Record an allergy in a visit
  ///
  /// SAL-41. The visit's practitioner records a confirmed allergy in it. It belongs to the visit (ADR 0013): seen where the visit is seen, until the patient adds it to their profile. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The encounter's id (FHIR Encounter.id).
  ///
  /// * [AllergyIntoleranceInput] allergyIntoleranceInput (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> recordAllergyWithHttpInfo(
    String id,
    AllergyIntoleranceInput allergyIntoleranceInput, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/encounters/{id}/allergies'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = allergyIntoleranceInput;

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

  /// Record an allergy in a visit
  ///
  /// SAL-41. The visit's practitioner records a confirmed allergy in it. It belongs to the visit (ADR 0013): seen where the visit is seen, until the patient adds it to their profile. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The encounter's id (FHIR Encounter.id).
  ///
  /// * [AllergyIntoleranceInput] allergyIntoleranceInput (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<EntryWritten?> recordAllergy(
    String id,
    AllergyIntoleranceInput allergyIntoleranceInput, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await recordAllergyWithHttpInfo(
      id,
      allergyIntoleranceInput,
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
        'EntryWritten',
      ) as EntryWritten;
    }
    return null;
  }

  /// Record a medication in a visit
  ///
  /// SAL-41. As `POST /encounters/{id}/allergies`, for a medication.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The encounter's id (FHIR Encounter.id).
  ///
  /// * [MedicationStatementInput] medicationStatementInput (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> recordMedicationWithHttpInfo(
    String id,
    MedicationStatementInput medicationStatementInput, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/encounters/{id}/medications'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = medicationStatementInput;

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

  /// Record a medication in a visit
  ///
  /// SAL-41. As `POST /encounters/{id}/allergies`, for a medication.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The encounter's id (FHIR Encounter.id).
  ///
  /// * [MedicationStatementInput] medicationStatementInput (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<EntryWritten?> recordMedication(
    String id,
    MedicationStatementInput medicationStatementInput, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await recordMedicationWithHttpInfo(
      id,
      medicationStatementInput,
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
        'EntryWritten',
      ) as EntryWritten;
    }
    return null;
  }

  /// Remove an allergy
  ///
  /// SAL-41. The patient removes an entry from what anyone sees: their own unconfirmed entry is deleted; a confirmed one or one a clinic recorded is kept with the record as entered-in-error. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The entry's id (FHIR AllergyIntolerance.id or MedicationStatement.id).
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> removeAllergyWithHttpInfo(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/allergies/{id}'.replaceAll('{id}', id);

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

  /// Remove an allergy
  ///
  /// SAL-41. The patient removes an entry from what anyone sees: their own unconfirmed entry is deleted; a confirmed one or one a clinic recorded is kept with the record as entered-in-error. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The entry's id (FHIR AllergyIntolerance.id or MedicationStatement.id).
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<void> removeAllergy(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await removeAllergyWithHttpInfo(
      id,
      xSalusCSRF: xSalusCSRF,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
  }

  /// Remove a medication
  ///
  /// SAL-41. As `DELETE /allergies/{id}`, for a medication.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The entry's id (FHIR AllergyIntolerance.id or MedicationStatement.id).
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> removeMedicationWithHttpInfo(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/medications/{id}'.replaceAll('{id}', id);

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

  /// Remove a medication
  ///
  /// SAL-41. As `DELETE /allergies/{id}`, for a medication.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The entry's id (FHIR AllergyIntolerance.id or MedicationStatement.id).
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<void> removeMedication(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await removeMedicationWithHttpInfo(
      id,
      xSalusCSRF: xSalusCSRF,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
  }

  /// Stop sharing history with a clinic
  ///
  /// SAL-42. Revokes the history consent with the clinic; takes effect when it commits. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] organizationId (required):
  ///   The receiving clinic (FHIR Organization.id).
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> revokeHistorySharingWithHttpInfo(
    String organizationId, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/history-sharing/{organizationId}'
        .replaceAll('{organizationId}', organizationId);

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

  /// Stop sharing history with a clinic
  ///
  /// SAL-42. Revokes the history consent with the clinic; takes effect when it commits. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] organizationId (required):
  ///   The receiving clinic (FHIR Organization.id).
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<void> revokeHistorySharing(
    String organizationId, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await revokeHistorySharingWithHttpInfo(
      organizationId,
      xSalusCSRF: xSalusCSRF,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
  }

  /// Share exactly these visits with a clinic
  ///
  /// SAL-42. Grants the history consent with the clinic, or changes the visits it lists, to exactly the ones sent: each a finished visit of the patient's at another clinic. An empty list revokes it. Takes effect when it commits. Needs the care consent with the clinic. To change an existing sharing send If-Match with its `versionId` (412 when it changed meanwhile, or when it is missing and a sharing exists); a first grant sends none. Every grant, change and revocation is audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] organizationId (required):
  ///   The receiving clinic (FHIR Organization.id).
  ///
  /// * [SetHistorySharingRequest] setHistorySharingRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  ///
  /// * [String] ifMatch:
  ///   The sharing's version read, as a weak ETag, `W/\"<versionId>\"`; absent for a first grant.
  Future<Response> setHistorySharingWithHttpInfo(
    String organizationId,
    SetHistorySharingRequest setHistorySharingRequest, {
    String? xSalusCSRF,
    String? ifMatch,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/history-sharing/{organizationId}'
        .replaceAll('{organizationId}', organizationId);

    // ignore: prefer_final_locals
    Object? postBody = setHistorySharingRequest;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    if (xSalusCSRF != null) {
      headerParams[r'X-Salus-CSRF'] = parameterToString(xSalusCSRF);
    }
    if (ifMatch != null) {
      headerParams[r'If-Match'] = parameterToString(ifMatch);
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

  /// Share exactly these visits with a clinic
  ///
  /// SAL-42. Grants the history consent with the clinic, or changes the visits it lists, to exactly the ones sent: each a finished visit of the patient's at another clinic. An empty list revokes it. Takes effect when it commits. Needs the care consent with the clinic. To change an existing sharing send If-Match with its `versionId` (412 when it changed meanwhile, or when it is missing and a sharing exists); a first grant sends none. Every grant, change and revocation is audited.
  ///
  /// Parameters:
  ///
  /// * [String] organizationId (required):
  ///   The receiving clinic (FHIR Organization.id).
  ///
  /// * [SetHistorySharingRequest] setHistorySharingRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  ///
  /// * [String] ifMatch:
  ///   The sharing's version read, as a weak ETag, `W/\"<versionId>\"`; absent for a first grant.
  Future<HistorySharing?> setHistorySharing(
    String organizationId,
    SetHistorySharingRequest setHistorySharingRequest, {
    String? xSalusCSRF,
    String? ifMatch,
    Future<void>? abortTrigger,
  }) async {
    final response = await setHistorySharingWithHttpInfo(
      organizationId,
      setHistorySharingRequest,
      xSalusCSRF: xSalusCSRF,
      ifMatch: ifMatch,
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
        'HistorySharing',
      ) as HistorySharing;
    }
    return null;
  }

  /// Change an allergy
  ///
  /// SAL-41. The patient changes their own entries; a practitioner the entries recorded in their visits. Send If-Match with the version read. A patient's change to a confirmed entry makes it unconfirmed again. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///   The entry's id (FHIR AllergyIntolerance.id or MedicationStatement.id).
  ///
  /// * [AllergyIntoleranceInput] allergyIntoleranceInput (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> updateAllergyWithHttpInfo(
    String ifMatch,
    String id,
    AllergyIntoleranceInput allergyIntoleranceInput, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/allergies/{id}'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = allergyIntoleranceInput;

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

  /// Change an allergy
  ///
  /// SAL-41. The patient changes their own entries; a practitioner the entries recorded in their visits. Send If-Match with the version read. A patient's change to a confirmed entry makes it unconfirmed again. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///   The entry's id (FHIR AllergyIntolerance.id or MedicationStatement.id).
  ///
  /// * [AllergyIntoleranceInput] allergyIntoleranceInput (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<EntryWritten?> updateAllergy(
    String ifMatch,
    String id,
    AllergyIntoleranceInput allergyIntoleranceInput, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await updateAllergyWithHttpInfo(
      ifMatch,
      id,
      allergyIntoleranceInput,
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
        'EntryWritten',
      ) as EntryWritten;
    }
    return null;
  }

  /// Change a medication
  ///
  /// SAL-41. As `PUT /allergies/{id}`, for a medication.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///   The entry's id (FHIR AllergyIntolerance.id or MedicationStatement.id).
  ///
  /// * [MedicationStatementInput] medicationStatementInput (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> updateMedicationWithHttpInfo(
    String ifMatch,
    String id,
    MedicationStatementInput medicationStatementInput, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/medications/{id}'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = medicationStatementInput;

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

  /// Change a medication
  ///
  /// SAL-41. As `PUT /allergies/{id}`, for a medication.
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///   The entry's id (FHIR AllergyIntolerance.id or MedicationStatement.id).
  ///
  /// * [MedicationStatementInput] medicationStatementInput (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<EntryWritten?> updateMedication(
    String ifMatch,
    String id,
    MedicationStatementInput medicationStatementInput, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await updateMedicationWithHttpInfo(
      ifMatch,
      id,
      medicationStatementInput,
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
        'EntryWritten',
      ) as EntryWritten;
    }
    return null;
  }
}
