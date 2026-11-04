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
}
