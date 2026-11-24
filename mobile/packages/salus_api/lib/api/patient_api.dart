//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class PatientApi {
  PatientApi([ApiClient? apiClient])
      : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// Correct a profile the clinic registered
  ///
  /// The names, birth date and phone of a patient without an account whom the caller's clinic registered (V2 of docs/clinic-cms WAVE4.md: the patient's rectification, through the clinic). For the clinic's admin and its receptionists. Versioned (`If-Match`). Built in package R2. Audited, also when refused. Another clinic's profile, or a patient's own, is not found.
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
  /// * [CorrectPatientRequest] correctPatientRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> correctClinicPatientWithHttpInfo(
    String id,
    String ifMatch,
    CorrectPatientRequest correctPatientRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/patients/{id}'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = correctPatientRequest;

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

  /// Correct a profile the clinic registered
  ///
  /// The names, birth date and phone of a patient without an account whom the caller's clinic registered (V2 of docs/clinic-cms WAVE4.md: the patient's rectification, through the clinic). For the clinic's admin and its receptionists. Versioned (`If-Match`). Built in package R2. Audited, also when refused. Another clinic's profile, or a patient's own, is not found.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [CorrectPatientRequest] correctPatientRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<ClinicPatient?> correctClinicPatient(
    String id,
    String ifMatch,
    CorrectPatientRequest correctPatientRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await correctClinicPatientWithHttpInfo(
      id,
      ifMatch,
      correctPatientRequest,
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
        'ClinicPatient',
      ) as ClinicPatient;
    }
    return null;
  }

  /// Download a registered patient's record to hand over
  ///
  /// A FHIR R4 Bundle of type collection with the Patient and the Consents the patient gave the clinic, saved as a file the admin hands to the patient (their access and portability, through the clinic). For the clinic's admin, on a patient the clinic registered. Built in package R2. Audited with the patient's request as the purpose, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Response> getClinicPatientRecordWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/patients/{id}/record'.replaceAll('{id}', id);

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

  /// Download a registered patient's record to hand over
  ///
  /// A FHIR R4 Bundle of type collection with the Patient and the Consents the patient gave the clinic, saved as a file the admin hands to the patient (their access and portability, through the clinic). For the clinic's admin, on a patient the clinic registered. Built in package R2. Audited with the patient's request as the purpose, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Object?> getClinicPatientRecord(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getClinicPatientRecordWithHttpInfo(
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
        'Object',
      ) as Object;
    }
    return null;
  }

  /// The signed-in patient's profile
  ///
  /// The profile of the signed-in patient account, or 404 before it is first saved (a profile exists only complete). Patients only. Not cached. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> getMyPatientWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/me/patient';

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

  /// The signed-in patient's profile
  ///
  /// The profile of the signed-in patient account, or 404 before it is first saved (a profile exists only complete). Patients only. Not cached. Audited, also when refused.
  Future<Patient?> getMyPatient({
    Future<void>? abortTrigger,
  }) async {
    final response = await getMyPatientWithHttpInfo(
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
        'Patient',
      ) as Patient;
    }
    return null;
  }

  /// Import the clinic's patients from a file
  ///
  /// A CSV (UTF-8, or Windows-1250 as Excel saves it) or XLSX file of patients (given name, family name, birth date, an optional phone; headers in English, Bosnian, Croatian or Serbian), at most 5 000 rows. With `save` false (the default) nothing is stored: every row comes back with its errors and whether it is a likely duplicate (a patient of the same name and birth date is the clinic's already, or on an earlier row). With `save` true all rows are registered as the clinic's profiles without a consent (the desk records it at the first booking) or, when any row has an error, none (`saved` false). A likely duplicate is saved all the same. By the clinic's admin. Built in package M3. The clinic's patients a preview finds are audited as a search, each saved profile as a create, and the import with its count.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [MultipartFile] file (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  ///
  /// * [bool] save:
  Future<Response> importClinicPatientsWithHttpInfo(
    MultipartFile file, {
    String? xSalusCSRF,
    bool? save,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/patients/import';

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    if (save != null) {
      queryParams.addAll(_queryParams('', 'save', save));
    }

    if (xSalusCSRF != null) {
      headerParams[r'X-Salus-CSRF'] = parameterToString(xSalusCSRF);
    }

    const contentTypes = <String>['multipart/form-data'];

    bool hasFields = false;
    final mp = MultipartRequest('POST', Uri.parse(path));
    if (file != null) {
      hasFields = true;
      mp.fields[r'file'] = file.field;
      mp.files.add(file);
    }
    if (hasFields) {
      postBody = mp;
    }

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

  /// Import the clinic's patients from a file
  ///
  /// A CSV (UTF-8, or Windows-1250 as Excel saves it) or XLSX file of patients (given name, family name, birth date, an optional phone; headers in English, Bosnian, Croatian or Serbian), at most 5 000 rows. With `save` false (the default) nothing is stored: every row comes back with its errors and whether it is a likely duplicate (a patient of the same name and birth date is the clinic's already, or on an earlier row). With `save` true all rows are registered as the clinic's profiles without a consent (the desk records it at the first booking) or, when any row has an error, none (`saved` false). A likely duplicate is saved all the same. By the clinic's admin. Built in package M3. The clinic's patients a preview finds are audited as a search, each saved profile as a create, and the import with its count.
  ///
  /// Parameters:
  ///
  /// * [MultipartFile] file (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  ///
  /// * [bool] save:
  Future<PatientImport?> importClinicPatients(
    MultipartFile file, {
    String? xSalusCSRF,
    bool? save,
    Future<void>? abortTrigger,
  }) async {
    final response = await importClinicPatientsWithHttpInfo(
      file,
      xSalusCSRF: xSalusCSRF,
      save: save,
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
        'PatientImport',
      ) as PatientImport;
    }
    return null;
  }

  /// Register a patient who has no Salus account
  ///
  /// The profile is the clinic's and seen by it only; the care consent is recorded as given in person, by the caller (W1 of docs/clinic-cms). For the clinic's admin and its receptionists. Built in package F2. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [RegisterPatientRequest] registerPatientRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> registerClinicPatientWithHttpInfo(
    RegisterPatientRequest registerPatientRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/patients';

    // ignore: prefer_final_locals
    Object? postBody = registerPatientRequest;

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

  /// Register a patient who has no Salus account
  ///
  /// The profile is the clinic's and seen by it only; the care consent is recorded as given in person, by the caller (W1 of docs/clinic-cms). For the clinic's admin and its receptionists. Built in package F2. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [RegisterPatientRequest] registerPatientRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<ClinicPatient?> registerClinicPatient(
    RegisterPatientRequest registerPatientRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await registerClinicPatientWithHttpInfo(
      registerPatientRequest,
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
        'ClinicPatient',
      ) as ClinicPatient;
    }
    return null;
  }

  /// Save the signed-in patient's profile
  ///
  /// Creates the profile when `versionId` is absent, with every required field; updates it when `versionId` is the current version. Names are trimmed. Patients only. Not cached. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [SavePatientRequest] savePatientRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> saveMyPatientWithHttpInfo(
    SavePatientRequest savePatientRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/me/patient';

    // ignore: prefer_final_locals
    Object? postBody = savePatientRequest;

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

  /// Save the signed-in patient's profile
  ///
  /// Creates the profile when `versionId` is absent, with every required field; updates it when `versionId` is the current version. Names are trimmed. Patients only. Not cached. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [SavePatientRequest] savePatientRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Patient?> saveMyPatient(
    SavePatientRequest savePatientRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await saveMyPatientWithHttpInfo(
      savePatientRequest,
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
        'Patient',
      ) as Patient;
    }
    return null;
  }

  /// Search the clinic's patients
  ///
  /// By name and, when given, birth date: the patients with the clinic's care consent and those it registered. For the clinic's admin, its receptionists and its practitioners (their own patients). Built in package F2 of docs/clinic-cms. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] name (required):
  ///
  /// * [DateTime] birthDate:
  Future<Response> searchClinicPatientsWithHttpInfo(
    String name, {
    DateTime? birthDate,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/patients';

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    queryParams.addAll(_queryParams('', 'name', name));
    if (birthDate != null) {
      queryParams.addAll(_queryParams('', 'birthDate', birthDate));
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

  /// Search the clinic's patients
  ///
  /// By name and, when given, birth date: the patients with the clinic's care consent and those it registered. For the clinic's admin, its receptionists and its practitioners (their own patients). Built in package F2 of docs/clinic-cms. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] name (required):
  ///
  /// * [DateTime] birthDate:
  Future<List<ClinicPatient>?> searchClinicPatients(
    String name, {
    DateTime? birthDate,
    Future<void>? abortTrigger,
  }) async {
    final response = await searchClinicPatientsWithHttpInfo(
      name,
      birthDate: birthDate,
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
              responseBody, 'List<ClinicPatient>') as List)
          .cast<ClinicPatient>()
          .toList(growable: false);
    }
    return null;
  }

  /// Record that a registered patient withdrew their consent
  ///
  /// The care consent the patient gave the clinic in person is revoked, with the caller as who recorded it, and the rest follows as for a patient's own revocation in the app: from the commit the clinic's practitioners lose access, and a job cancels the visits ahead (with their notices) and withdraws the waitlist entries. For the clinic's admin and its receptionists, on a patient the clinic registered. Built in package R2. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> withdrawClinicPatientConsentWithHttpInfo(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path =
        r'/clinic/patients/{id}/consent-withdrawal'.replaceAll('{id}', id);

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
      'POST',
      queryParams,
      postBody,
      headerParams,
      formParams,
      contentTypes.isEmpty ? null : contentTypes.first,
      abortTrigger: abortTrigger,
    );
  }

  /// Record that a registered patient withdrew their consent
  ///
  /// The care consent the patient gave the clinic in person is revoked, with the caller as who recorded it, and the rest follows as for a patient's own revocation in the app: from the commit the clinic's practitioners lose access, and a job cancels the visits ahead (with their notices) and withdraws the waitlist entries. For the clinic's admin and its receptionists, on a patient the clinic registered. Built in package R2. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<void> withdrawClinicPatientConsent(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await withdrawClinicPatientConsentWithHttpInfo(
      id,
      xSalusCSRF: xSalusCSRF,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
  }
}
