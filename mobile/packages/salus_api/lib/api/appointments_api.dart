//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class AppointmentsApi {
  AppointmentsApi([ApiClient? apiClient])
      : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// Book a free slot
  ///
  /// The signed-in patient books a free slot of a listed clinic (FHIR Appointment, docs/booking/SPEC.md \"Booking rules\"): booked at once, the slot held in the same transaction. Needs a profile and an active consent with the clinic. A confirmation is e-mailed, and a reminder 24 hours before the start when that is ahead. Patients only. Not cached. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [BookAppointmentRequest] bookAppointmentRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> bookAppointmentWithHttpInfo(
    BookAppointmentRequest bookAppointmentRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/appointments';

    // ignore: prefer_final_locals
    Object? postBody = bookAppointmentRequest;

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

  /// Book a free slot
  ///
  /// The signed-in patient books a free slot of a listed clinic (FHIR Appointment, docs/booking/SPEC.md \"Booking rules\"): booked at once, the slot held in the same transaction. Needs a profile and an active consent with the clinic. A confirmation is e-mailed, and a reminder 24 hours before the start when that is ahead. Patients only. Not cached. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [BookAppointmentRequest] bookAppointmentRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Appointment?> bookAppointment(
    BookAppointmentRequest bookAppointmentRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await bookAppointmentWithHttpInfo(
      bookAppointmentRequest,
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

  /// Cancel an appointment
  ///
  /// Cancels a booked appointment: the patient their own up to 24 hours before the start (reason `pat`), the appointment's practitioner and the clinic's admin at any time (reason `prov`). The slot is freed while its start is ahead, and the patient is e-mailed. Not cached. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> cancelAppointmentWithHttpInfo(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/appointments/{id}/cancel'.replaceAll('{id}', id);

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

  /// Cancel an appointment
  ///
  /// Cancels a booked appointment: the patient their own up to 24 hours before the start (reason `pat`), the appointment's practitioner and the clinic's admin at any time (reason `prov`). The slot is freed while its start is ahead, and the patient is e-mailed. Not cached. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Appointment?> cancelAppointment(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await cancelAppointmentWithHttpInfo(
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

  /// The patient confirms they will come
  ///
  /// The patient's own appointment, while booked and ahead; confirming again answers the same. Recorded in its history (`via` `app`). Not cached. Audited, also when refused. Built in SAL-34 (404 until then).
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> confirmAppointmentWithHttpInfo(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/appointments/{id}/confirm'.replaceAll('{id}', id);

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

  /// The patient confirms they will come
  ///
  /// The patient's own appointment, while booked and ahead; confirming again answers the same. Recorded in its history (`via` `app`). Not cached. Audited, also when refused. Built in SAL-34 (404 until then).
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Appointment?> confirmAppointment(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await confirmAppointmentWithHttpInfo(
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

  /// What the patient may do with their appointment now
  ///
  /// For the patient's app: whether confirming, cancelling and rescheduling are possible, until when, and how many reschedules are left, by the clinic's rules. Not cached. Built in SAL-34 (404 until then).
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Response> getAppointmentActionsWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/appointments/{id}/actions'.replaceAll('{id}', id);

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

  /// What the patient may do with their appointment now
  ///
  /// For the patient's app: whether confirming, cancelling and rescheduling are possible, until when, and how many reschedules are left, by the clinic's rules. Not cached. Built in SAL-34 (404 until then).
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<AppointmentActions?> getAppointmentActions(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getAppointmentActionsWithHttpInfo(
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
        'AppointmentActions',
      ) as AppointmentActions;
    }
    return null;
  }

  /// The admin's clinic's appointment settings
  ///
  /// The defaults (reminders 24 hours and 1 hour before, no SMS, rescheduling until 24 hours before and twice at most, no waitlist with a 15-minute hold, no no-show flag) until the admin sets them (docs/plans/appointments.md).
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> getAppointmentSettingsWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/appointment-settings';

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

  /// The admin's clinic's appointment settings
  ///
  /// The defaults (reminders 24 hours and 1 hour before, no SMS, rescheduling until 24 hours before and twice at most, no waitlist with a 15-minute hold, no no-show flag) until the admin sets them (docs/plans/appointments.md).
  Future<AppointmentSettings?> getAppointmentSettings({
    Future<void>? abortTrigger,
  }) async {
    final response = await getAppointmentSettingsWithHttpInfo(
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
        'AppointmentSettings',
      ) as AppointmentSettings;
    }
    return null;
  }

  /// An appointment's history
  ///
  /// Its confirmations, reschedules and statuses, oldest first, for whoever sees the appointment in full: the patient their own, its practitioner and the clinic's admin with the patient's consent. Who made a change is never in it. Not cached. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Response> listAppointmentChangesWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/appointments/{id}/changes'.replaceAll('{id}', id);

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

  /// An appointment's history
  ///
  /// Its confirmations, reschedules and statuses, oldest first, for whoever sees the appointment in full: the patient their own, its practitioner and the clinic's admin with the patient's consent. Who made a change is never in it. Not cached. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<AppointmentChangeList?> listAppointmentChanges(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await listAppointmentChangesWithHttpInfo(
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
        'AppointmentChangeList',
      ) as AppointmentChangeList;
    }
    return null;
  }

  /// Repeat no-show flags beside the calendar's appointments
  ///
  /// For the clinic's staff, with the clinic's threshold on: the appointments starting in [from, to) (at most 62 days) the caller sees in full whose patient missed at least the threshold of appointments at the clinic in the last `window_days` days (`noshows` of them); empty when the rule is off. Computed on read, never stored, never shown to the patient and never a reason to refuse a booking (docs/plans/appointments.md Q7). Each flagged appointment's read is audited. Not cached.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [DateTime] from (required):
  ///
  /// * [DateTime] to (required):
  Future<Response> listNoShowFlagsWithHttpInfo(
    DateTime from,
    DateTime to, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/noshow-flags';

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    queryParams.addAll(_queryParams('', 'from', from));
    queryParams.addAll(_queryParams('', 'to', to));

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

  /// Repeat no-show flags beside the calendar's appointments
  ///
  /// For the clinic's staff, with the clinic's threshold on: the appointments starting in [from, to) (at most 62 days) the caller sees in full whose patient missed at least the threshold of appointments at the clinic in the last `window_days` days (`noshows` of them); empty when the rule is off. Computed on read, never stored, never shown to the patient and never a reason to refuse a booking (docs/plans/appointments.md Q7). Each flagged appointment's read is audited. Not cached.
  ///
  /// Parameters:
  ///
  /// * [DateTime] from (required):
  ///
  /// * [DateTime] to (required):
  Future<NoShowFlags?> listNoShowFlags(
    DateTime from,
    DateTime to, {
    Future<void>? abortTrigger,
  }) async {
    final response = await listNoShowFlagsWithHttpInfo(
      from,
      to,
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
        'NoShowFlags',
      ) as NoShowFlags;
    }
    return null;
  }

  /// The patient moves their appointment to another free slot of the same practitioner role
  ///
  /// The same appointment at a new time (ADR 0015): within the clinic's notice and limit, never once the consultation's join window has opened. The new slot is held and the old one freed (and offered to the waitlist) in one transaction; the reminders are scheduled again; choosing the time confirms it. Not cached. Audited, also when refused. Built in SAL-34 (404 until then).
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [RescheduleAppointmentRequest] rescheduleAppointmentRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> rescheduleAppointmentWithHttpInfo(
    String id,
    RescheduleAppointmentRequest rescheduleAppointmentRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/appointments/{id}/reschedule'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = rescheduleAppointmentRequest;

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

  /// The patient moves their appointment to another free slot of the same practitioner role
  ///
  /// The same appointment at a new time (ADR 0015): within the clinic's notice and limit, never once the consultation's join window has opened. The new slot is held and the old one freed (and offered to the waitlist) in one transaction; the reminders are scheduled again; choosing the time confirms it. Not cached. Audited, also when refused. Built in SAL-34 (404 until then).
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [RescheduleAppointmentRequest] rescheduleAppointmentRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Appointment?> rescheduleAppointment(
    String id,
    RescheduleAppointmentRequest rescheduleAppointmentRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await rescheduleAppointmentWithHttpInfo(
      id,
      rescheduleAppointmentRequest,
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

  /// Replace the admin's clinic's appointment settings
  ///
  /// Every field required; the ranges are the schema's, and the reminder offsets are distinct. A change applies to appointments booked, accepted or rescheduled after it. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [AppointmentSettings] appointmentSettings (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> setAppointmentSettingsWithHttpInfo(
    AppointmentSettings appointmentSettings, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/appointment-settings';

    // ignore: prefer_final_locals
    Object? postBody = appointmentSettings;

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

  /// Replace the admin's clinic's appointment settings
  ///
  /// Every field required; the ranges are the schema's, and the reminder offsets are distinct. A change applies to appointments booked, accepted or rescheduled after it. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [AppointmentSettings] appointmentSettings (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<AppointmentSettings?> setAppointmentSettings(
    AppointmentSettings appointmentSettings, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await setAppointmentSettingsWithHttpInfo(
      appointmentSettings,
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
        'AppointmentSettings',
      ) as AppointmentSettings;
    }
    return null;
  }

  /// Mark an appointment fulfilled or no-show, or correct it
  ///
  /// The appointment's practitioner marks it `fulfilled` or `noshow` once its start has passed, once, kept in its history (GET /appointments/{id}/changes). The practitioner and the clinic's admin also correct a `noshow` to `fulfilled` and back within 30 days of the start (SAL-36, docs/plans/appointments.md Q6), kept in its history as `status_corrected`; 409 `correction_window_closed` after. Not cached. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [SetAppointmentStatusRequest] setAppointmentStatusRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> setAppointmentStatusWithHttpInfo(
    String id,
    SetAppointmentStatusRequest setAppointmentStatusRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/appointments/{id}/status'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = setAppointmentStatusRequest;

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

  /// Mark an appointment fulfilled or no-show, or correct it
  ///
  /// The appointment's practitioner marks it `fulfilled` or `noshow` once its start has passed, once, kept in its history (GET /appointments/{id}/changes). The practitioner and the clinic's admin also correct a `noshow` to `fulfilled` and back within 30 days of the start (SAL-36, docs/plans/appointments.md Q6), kept in its history as `status_corrected`; 409 `correction_window_closed` after. Not cached. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [SetAppointmentStatusRequest] setAppointmentStatusRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Appointment?> setAppointmentStatus(
    String id,
    SetAppointmentStatusRequest setAppointmentStatusRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await setAppointmentStatusWithHttpInfo(
      id,
      setAppointmentStatusRequest,
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
}
