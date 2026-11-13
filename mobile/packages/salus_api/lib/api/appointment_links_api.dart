//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class AppointmentLinksApi {
  AppointmentLinksApi([ApiClient? apiClient])
      : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// Confirm or cancel through a reminder's one-time link
  ///
  /// The page's form, the second tap: `confirm` while unconfirmed, `cancel` (with the box `confirm_cancel` ticked) while the patient's 24-hour window is open. Uses the link once (`UPDATE … WHERE used_at IS NULL`, a used link never changes), in the transaction that confirms or cancels, as the link's patient on its appointment only; the `Origin` must be the page's own (its host the request's). Answers 303 to the page, which shows the result, also when the link no longer works, the confirmation was made already or the 24 hours have begun (the page says what is true now); a refused action leaves the link unused. No session.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] token (required):
  ///   The one-time link's token, 32 random bytes in base64url. Never logged; only its SHA-256 is stored.
  ///
  /// * [String] action (required):
  ///
  /// * [String] confirmCancel:
  ///   The ticked box a cancel needs (ADR 0015 \\\"Scanners that click\\\"); ignored for a confirm.
  Future<Response> actOnAppointmentLinkWithHttpInfo(
    String token,
    String action, {
    String? confirmCancel,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/a/{token}'.replaceAll('{token}', token);

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    const contentTypes = <String>['application/x-www-form-urlencoded'];

    if (action != null) {
      formParams[r'action'] = parameterToString(action);
    }
    if (confirmCancel != null) {
      formParams[r'confirm_cancel'] = parameterToString(confirmCancel);
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

  /// Confirm or cancel through a reminder's one-time link
  ///
  /// The page's form, the second tap: `confirm` while unconfirmed, `cancel` (with the box `confirm_cancel` ticked) while the patient's 24-hour window is open. Uses the link once (`UPDATE … WHERE used_at IS NULL`, a used link never changes), in the transaction that confirms or cancels, as the link's patient on its appointment only; the `Origin` must be the page's own (its host the request's). Answers 303 to the page, which shows the result, also when the link no longer works, the confirmation was made already or the 24 hours have begun (the page says what is true now); a refused action leaves the link unused. No session.
  ///
  /// Parameters:
  ///
  /// * [String] token (required):
  ///   The one-time link's token, 32 random bytes in base64url. Never logged; only its SHA-256 is stored.
  ///
  /// * [String] action (required):
  ///
  /// * [String] confirmCancel:
  ///   The ticked box a cancel needs (ADR 0015 \\\"Scanners that click\\\"); ignored for a confirm.
  Future<void> actOnAppointmentLink(
    String token,
    String action, {
    String? confirmCancel,
    Future<void>? abortTrigger,
  }) async {
    final response = await actOnAppointmentLinkWithHttpInfo(
      token,
      action,
      confirmCancel: confirmCancel,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
  }

  /// The page of a reminder's one-time link
  ///
  /// An HTML page without JavaScript, cookies or third-party requests (ADR 0015 \"One-tap links\"): the clinic, the date and time, the appointment's state, and the actions still possible as buttons. Changes nothing, however often it is fetched (link prefetchers). A used link shows its holder its own outcome (\"You confirmed this appointment\", \"You cancelled this appointment\", with the clinic, date and time only) and offers nothing more: a link is single use for one action. An unknown or expired link, or one issued for another start than the appointment's now, or one of an appointment no longer booked or under a consent no longer active, answers the same page saying it no longer works. A GET never changes anything. No session. Audited as a read by the patient's account with the agent `action_link`. Lookups are limited per address (an HMAC of it in Redis).
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] token (required):
  ///   The one-time link's token, 32 random bytes in base64url. Never logged; only its SHA-256 is stored.
  Future<Response> getAppointmentLinkPageWithHttpInfo(
    String token, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/a/{token}'.replaceAll('{token}', token);

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

  /// The page of a reminder's one-time link
  ///
  /// An HTML page without JavaScript, cookies or third-party requests (ADR 0015 \"One-tap links\"): the clinic, the date and time, the appointment's state, and the actions still possible as buttons. Changes nothing, however often it is fetched (link prefetchers). A used link shows its holder its own outcome (\"You confirmed this appointment\", \"You cancelled this appointment\", with the clinic, date and time only) and offers nothing more: a link is single use for one action. An unknown or expired link, or one issued for another start than the appointment's now, or one of an appointment no longer booked or under a consent no longer active, answers the same page saying it no longer works. A GET never changes anything. No session. Audited as a read by the patient's account with the agent `action_link`. Lookups are limited per address (an HMAC of it in Redis).
  ///
  /// Parameters:
  ///
  /// * [String] token (required):
  ///   The one-time link's token, 32 random bytes in base64url. Never logged; only its SHA-256 is stored.
  Future<String?> getAppointmentLinkPage(
    String token, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getAppointmentLinkPageWithHttpInfo(
      token,
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
}
