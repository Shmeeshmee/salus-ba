//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class DirectoryApi {
  DirectoryApi([ApiClient? apiClient])
      : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// Download a ready export
  ///
  /// By the clinic's admin, until it expires (package M2). Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Response> downloadClinicExportWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/export/{id}/archive'.replaceAll('{id}', id);

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
  /// By the clinic's admin, until it expires (package M2). Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<String?> downloadClinicExport(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await downloadClinicExportWithHttpInfo(
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

  /// Where an export stands
  ///
  /// By the clinic's admin (package M2). Not audited; it holds no personal data.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Response> getClinicExportWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/export/{id}'.replaceAll('{id}', id);

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

  /// Where an export stands
  ///
  /// By the clinic's admin (package M2). Not audited; it holds no personal data.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<ClinicExport?> getClinicExport(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getClinicExportWithHttpInfo(
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
        'ClinicExport',
      ) as ClinicExport;
    }
    return null;
  }

  /// The clinic's profile for patients abroad
  ///
  /// For the clinic's staff. Built in package X2. Not personal data; not audited.
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> getClinicProfileWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/profile';

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

  /// The clinic's profile for patients abroad
  ///
  /// For the clinic's staff. Built in package X2. Not personal data; not audited.
  Future<ClinicProfile?> getClinicProfile({
    Future<void>? abortTrigger,
  }) async {
    final response = await getClinicProfileWithHttpInfo(
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
        'ClinicProfile',
      ) as ClinicProfile;
    }
    return null;
  }

  /// A listed clinic's profile for patients
  ///
  /// Public, as the directory: the languages its staff speak, its description (in the patient's language when translated) and whether it welcomes patients from abroad. A clinic whose membership is suspended or cancelled is out of the directory; its own staff and the patients with a care consent with it, active or revoked, still read it. Built in package X2. Not personal data; not audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Response> getOrganizationProfileWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/organizations/{id}/profile'.replaceAll('{id}', id);

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

  /// A listed clinic's profile for patients
  ///
  /// Public, as the directory: the languages its staff speak, its description (in the patient's language when translated) and whether it welcomes patients from abroad. A clinic whose membership is suspended or cancelled is out of the directory; its own staff and the patients with a care consent with it, active or revoked, still read it. Built in package X2. Not personal data; not audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<ClinicProfile?> getOrganizationProfile(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getOrganizationProfileWithHttpInfo(
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
        'ClinicProfile',
      ) as ClinicProfile;
    }
    return null;
  }

  /// List the specialties a role can have
  ///
  /// The specialties an admin may give a practitioner role, in display order: the directory's SNOMED CT codes from FHIR R4 ValueSet c80-practice-codes, with the name the apps show. Fixed and public; needs no session. Not audited: it holds no personal data.
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> listSpecialtiesWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/specialties';

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

  /// List the specialties a role can have
  ///
  /// The specialties an admin may give a practitioner role, in display order: the directory's SNOMED CT codes from FHIR R4 ValueSet c80-practice-codes, with the name the apps show. Fixed and public; needs no session. Not audited: it holds no personal data.
  Future<SpecialtyList?> listSpecialties({
    Future<void>? abortTrigger,
  }) async {
    final response = await listSpecialtiesWithHttpInfo(
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
        'SpecialtyList',
      ) as SpecialtyList;
    }
    return null;
  }

  /// Ask for the clinic's data export
  ///
  /// One zip archive built by a job (package M2, joined in the wave 4 integration): the price list, staff, appointments, receipts and quotes as CSV, and the record of each patient the clinic registered as a FHIR Bundle (`patients/<id>.json`: the profile, the clinic's consents, the patient's appointments and receipts at the clinic), with a `manifest.txt` that says what the archive holds and what it leaves out (clinical documents, questionnaire answers and history). Encrypted at rest, downloadable for 7 days after the request, then deleted. One export at a time per clinic: asking again while one is queued returns that one. By the clinic's admin. Audited, every read made for it too.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> requestClinicExportWithHttpInfo({
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/export';

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

  /// Ask for the clinic's data export
  ///
  /// One zip archive built by a job (package M2, joined in the wave 4 integration): the price list, staff, appointments, receipts and quotes as CSV, and the record of each patient the clinic registered as a FHIR Bundle (`patients/<id>.json`: the profile, the clinic's consents, the patient's appointments and receipts at the clinic), with a `manifest.txt` that says what the archive holds and what it leaves out (clinical documents, questionnaire answers and history). Encrypted at rest, downloadable for 7 days after the request, then deleted. One export at a time per clinic: asking again while one is queued returns that one. By the clinic's admin. Audited, every read made for it too.
  ///
  /// Parameters:
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<ClinicExport?> requestClinicExport({
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await requestClinicExportWithHttpInfo(
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
        'ClinicExport',
      ) as ClinicExport;
    }
    return null;
  }

  /// Save the clinic's profile
  ///
  /// By the clinic's admin, with the organization's version. Built in package X2. Audited as an organization change.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [ClinicProfile] clinicProfile (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> saveClinicProfileWithHttpInfo(
    String ifMatch,
    ClinicProfile clinicProfile, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/profile';

    // ignore: prefer_final_locals
    Object? postBody = clinicProfile;

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

  /// Save the clinic's profile
  ///
  /// By the clinic's admin, with the organization's version. Built in package X2. Audited as an organization change.
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [ClinicProfile] clinicProfile (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<ClinicProfile?> saveClinicProfile(
    String ifMatch,
    ClinicProfile clinicProfile, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await saveClinicProfileWithHttpInfo(
      ifMatch,
      clinicProfile,
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
        'ClinicProfile',
      ) as ClinicProfile;
    }
    return null;
  }

  /// Set a practitioner role's specialty
  ///
  /// The clinic's admin sets the specialty of a role of the clinic: one of the twelve SNOMED CT codes the directory offers (FHIR R4 ValueSet c80-practice-codes). Patients see it when the clinic is listed. Send If-Match with the version the role was read at (`PractitionerRole.meta.versionId`); the answer carries the new version as its ETag. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] ifMatch (required):
  ///   The role's version as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [SetSpecialtyRequest] setSpecialtyRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> setPractitionerRoleSpecialtyWithHttpInfo(
    String id,
    String ifMatch,
    SetSpecialtyRequest setSpecialtyRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/practitioner-roles/{id}/specialty'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = setSpecialtyRequest;

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

  /// Set a practitioner role's specialty
  ///
  /// The clinic's admin sets the specialty of a role of the clinic: one of the twelve SNOMED CT codes the directory offers (FHIR R4 ValueSet c80-practice-codes). Patients see it when the clinic is listed. Send If-Match with the version the role was read at (`PractitionerRole.meta.versionId`); the answer carries the new version as its ETag. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] ifMatch (required):
  ///   The role's version as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [SetSpecialtyRequest] setSpecialtyRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<RoleSpecialty?> setPractitionerRoleSpecialty(
    String id,
    String ifMatch,
    SetSpecialtyRequest setSpecialtyRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await setPractitionerRoleSpecialtyWithHttpInfo(
      id,
      ifMatch,
      setSpecialtyRequest,
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
        'RoleSpecialty',
      ) as RoleSpecialty;
    }
    return null;
  }
}
