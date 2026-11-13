//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class NotificationsApi {
  NotificationsApi([ApiClient? apiClient])
      : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// The patient's reminder channels, quiet hours and time zone
  ///
  /// Push and e-mail on and SMS off, quiet hours 22:00 to 08:00 and no zone (the clinic's is used) until the patient changes them. Patients only. Built in SAL-33 (404 until then).
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> getNotificationSettingsWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/me/notification-settings';

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

  /// The patient's reminder channels, quiet hours and time zone
  ///
  /// Push and e-mail on and SMS off, quiet hours 22:00 to 08:00 and no zone (the clinic's is used) until the patient changes them. Patients only. Built in SAL-33 (404 until then).
  Future<NotificationSettings?> getNotificationSettings({
    Future<void>? abortTrigger,
  }) async {
    final response = await getNotificationSettingsWithHttpInfo(
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
        'NotificationSettings',
      ) as NotificationSettings;
    }
    return null;
  }

  /// The admin's clinic's SMS usage per month
  ///
  /// Messages and parts sent per calendar month (UTC), oldest first. Built in SAL-33 (404 until then).
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] from (required):
  ///
  /// * [String] to (required):
  Future<Response> getSmsUsageWithHttpInfo(
    String from,
    String to, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/sms-usage';

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

  /// The admin's clinic's SMS usage per month
  ///
  /// Messages and parts sent per calendar month (UTC), oldest first. Built in SAL-33 (404 until then).
  ///
  /// Parameters:
  ///
  /// * [String] from (required):
  ///
  /// * [String] to (required):
  Future<SmsUsage?> getSmsUsage(
    String from,
    String to, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getSmsUsageWithHttpInfo(
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
        'SmsUsage',
      ) as SmsUsage;
    }
    return null;
  }

  /// Replace the patient's reminder channels, quiet hours and time zone
  ///
  /// SMS on needs a phone in the profile, and reaches the patient only at clinics that offer it. Omit `quiet_hours` for none. `time_zone` is the device's IANA zone, sent by the app. Built in SAL-33 (404 until then).
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [NotificationSettings] notificationSettings (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> setNotificationSettingsWithHttpInfo(
    NotificationSettings notificationSettings, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/me/notification-settings';

    // ignore: prefer_final_locals
    Object? postBody = notificationSettings;

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

  /// Replace the patient's reminder channels, quiet hours and time zone
  ///
  /// SMS on needs a phone in the profile, and reaches the patient only at clinics that offer it. Omit `quiet_hours` for none. `time_zone` is the device's IANA zone, sent by the app. Built in SAL-33 (404 until then).
  ///
  /// Parameters:
  ///
  /// * [NotificationSettings] notificationSettings (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<NotificationSettings?> setNotificationSettings(
    NotificationSettings notificationSettings, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await setNotificationSettingsWithHttpInfo(
      notificationSettings,
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
        'NotificationSettings',
      ) as NotificationSettings;
    }
    return null;
  }
}
