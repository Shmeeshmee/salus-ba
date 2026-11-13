//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class WaitlistApi {
  WaitlistApi([ApiClient? apiClient])
      : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// The patient accepts an offered slot, which books it
  ///
  /// While the offer is open and unexpired: the slot moves from held to booked and the appointment is stored (`bookedVia` `waitlist`), in one transaction; paid through the purchase step, which charges nothing until the payments epic adds prices. Audited, also when refused. Built in SAL-35 (404 until then).
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> acceptWaitlistOfferWithHttpInfo(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/waitlist/offers/{id}/accept'.replaceAll('{id}', id);

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

  /// The patient accepts an offered slot, which books it
  ///
  /// While the offer is open and unexpired: the slot moves from held to booked and the appointment is stored (`bookedVia` `waitlist`), in one transaction; paid through the purchase step, which charges nothing until the payments epic adds prices. Audited, also when refused. Built in SAL-35 (404 until then).
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Appointment?> acceptWaitlistOffer(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await acceptWaitlistOfferWithHttpInfo(
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
        'Appointment',
      ) as Appointment;
    }
    return null;
  }

  /// The patient declines an offered slot
  ///
  /// The entry stays active; the slot goes to the next entry. Built in SAL-35 (404 until then).
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> declineWaitlistOfferWithHttpInfo(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/waitlist/offers/{id}/decline'.replaceAll('{id}', id);

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

  /// The patient declines an offered slot
  ///
  /// The entry stays active; the slot goes to the next entry. Built in SAL-35 (404 until then).
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<void> declineWaitlistOffer(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await declineWaitlistOfferWithHttpInfo(
      id,
      xSalusCSRF: xSalusCSRF,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
  }

  /// The patient joins a clinic's waitlist
  ///
  /// For a practitioner role or a specialty of the clinic, between two dates and, optionally, a time of day (the clinic's zone). Needs the clinic's waitlist on and the patient's active care consent; at most 5 active entries per clinic, dates at most 90 days ahead. Audited, also when refused. Built in SAL-35 (404 until then).
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [NewWaitlistEntry] newWaitlistEntry (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> joinWaitlistWithHttpInfo(
    NewWaitlistEntry newWaitlistEntry, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/waitlist';

    // ignore: prefer_final_locals
    Object? postBody = newWaitlistEntry;

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

  /// The patient joins a clinic's waitlist
  ///
  /// For a practitioner role or a specialty of the clinic, between two dates and, optionally, a time of day (the clinic's zone). Needs the clinic's waitlist on and the patient's active care consent; at most 5 active entries per clinic, dates at most 90 days ahead. Audited, also when refused. Built in SAL-35 (404 until then).
  ///
  /// Parameters:
  ///
  /// * [NewWaitlistEntry] newWaitlistEntry (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<WaitlistEntry?> joinWaitlist(
    NewWaitlistEntry newWaitlistEntry, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await joinWaitlistWithHttpInfo(
      newWaitlistEntry,
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
        'WaitlistEntry',
      ) as WaitlistEntry;
    }
    return null;
  }

  /// The clinic's waitlist, in offer order
  ///
  /// For the clinic's staff: entries by `priority` then join time, each with the patient's name at the level the care consent gives and its open offer. Audited. Built in SAL-35 (404 until then).
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] practitionerRole:
  ///
  /// * [WaitlistStatus] status:
  Future<Response> listClinicWaitlistWithHttpInfo({
    String? practitionerRole,
    WaitlistStatus? status,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/waitlist';

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    if (practitionerRole != null) {
      queryParams
          .addAll(_queryParams('', 'practitioner_role', practitionerRole));
    }
    if (status != null) {
      queryParams.addAll(_queryParams('', 'status', status));
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

  /// The clinic's waitlist, in offer order
  ///
  /// For the clinic's staff: entries by `priority` then join time, each with the patient's name at the level the care consent gives and its open offer. Audited. Built in SAL-35 (404 until then).
  ///
  /// Parameters:
  ///
  /// * [String] practitionerRole:
  ///
  /// * [WaitlistStatus] status:
  Future<ClinicWaitlist?> listClinicWaitlist({
    String? practitionerRole,
    WaitlistStatus? status,
    Future<void>? abortTrigger,
  }) async {
    final response = await listClinicWaitlistWithHttpInfo(
      practitionerRole: practitionerRole,
      status: status,
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
        'ClinicWaitlist',
      ) as ClinicWaitlist;
    }
    return null;
  }

  /// The patient's waitlist entries, with their open offers
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> listMyWaitlistWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/me/waitlist';

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

  /// The patient's waitlist entries, with their open offers
  Future<WaitlistEntryList?> listMyWaitlist({
    Future<void>? abortTrigger,
  }) async {
    final response = await listMyWaitlistWithHttpInfo(
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
        'WaitlistEntryList',
      ) as WaitlistEntryList;
    }
    return null;
  }

  /// The patient withdraws an entry, or the clinic's staff remove it
  ///
  /// An open offer of the entry ends with it, and its slot goes to the next entry. Audited. Built in SAL-35.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> removeWaitlistEntryWithHttpInfo(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/waitlist/{id}'.replaceAll('{id}', id);

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

  /// The patient withdraws an entry, or the clinic's staff remove it
  ///
  /// An open offer of the entry ends with it, and its slot goes to the next entry. Audited. Built in SAL-35.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<void> removeWaitlistEntry(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await removeWaitlistEntryWithHttpInfo(
      id,
      xSalusCSRF: xSalusCSRF,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
  }

  /// Staff move an entry up or down the waitlist
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [WaitlistPriority] waitlistPriority (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> setWaitlistPriorityWithHttpInfo(
    String id,
    WaitlistPriority waitlistPriority, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/waitlist/{id}/priority'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = waitlistPriority;

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

  /// Staff move an entry up or down the waitlist
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [WaitlistPriority] waitlistPriority (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<ClinicWaitlistEntry?> setWaitlistPriority(
    String id,
    WaitlistPriority waitlistPriority, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await setWaitlistPriorityWithHttpInfo(
      id,
      waitlistPriority,
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
        'ClinicWaitlistEntry',
      ) as ClinicWaitlistEntry;
    }
    return null;
  }
}
