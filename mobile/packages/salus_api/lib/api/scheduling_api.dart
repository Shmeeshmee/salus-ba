//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class SchedulingApi {
  SchedulingApi([ApiClient? apiClient])
      : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// Read a practitioner role's weekly availability
  ///
  /// The role's schedule: its slot length and weekly rules, times of day in the clinic's time zone. Before the first edit it is empty at version 0 (no `id`, no `slotMinutes`). The practitioner reads their own role's, the clinic's admin any role's. The answer carries the version as its ETag. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Response> getAvailabilityWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path =
        r'/practitioner-roles/{id}/availability'.replaceAll('{id}', id);

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

  /// Read a practitioner role's weekly availability
  ///
  /// The role's schedule: its slot length and weekly rules, times of day in the clinic's time zone. Before the first edit it is empty at version 0 (no `id`, no `slotMinutes`). The practitioner reads their own role's, the clinic's admin any role's. The answer carries the version as its ETag. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Availability?> getAvailability(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getAvailabilityWithHttpInfo(
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
        'Availability',
      ) as Availability;
    }
    return null;
  }

  /// Set a practitioner role's weekly availability
  ///
  /// Sets the slot length and the weekly rules of the role; the first edit creates its schedule. Send If-Match with the version read (`W/\"0\"` before the first edit); the answer carries the new version as its ETag. The role's free slots for the next 28 days follow within seconds; booked slots are never changed. The practitioner edits their own role's, the clinic's admin any role's. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The availability's version as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///
  /// * [SaveAvailabilityRequest] saveAvailabilityRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> saveAvailabilityWithHttpInfo(
    String ifMatch,
    String id,
    SaveAvailabilityRequest saveAvailabilityRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path =
        r'/practitioner-roles/{id}/availability'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = saveAvailabilityRequest;

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

  /// Set a practitioner role's weekly availability
  ///
  /// Sets the slot length and the weekly rules of the role; the first edit creates its schedule. Send If-Match with the version read (`W/\"0\"` before the first edit); the answer carries the new version as its ETag. The role's free slots for the next 28 days follow within seconds; booked slots are never changed. The practitioner edits their own role's, the clinic's admin any role's. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The availability's version as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///
  /// * [SaveAvailabilityRequest] saveAvailabilityRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Availability?> saveAvailability(
    String ifMatch,
    String id,
    SaveAvailabilityRequest saveAvailabilityRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await saveAvailabilityWithHttpInfo(
      ifMatch,
      id,
      saveAvailabilityRequest,
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
        'Availability',
      ) as Availability;
    }
    return null;
  }
}
