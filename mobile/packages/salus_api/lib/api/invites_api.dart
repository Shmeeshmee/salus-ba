//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class InvitesApi {
  InvitesApi([ApiClient? apiClient])
      : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// Accept an invite
  ///
  /// With the token from the link's fragment and a password: creates the account (and, for a practitioner, the Practitioner and PractitionerRole) and returns a pending TOTP secret with a challenge. Confirm the first code with `/auth/mfa/confirm` and the challenge (valid 5 minutes) to finish enrolment and open the session; if the challenge lapses, sign in with the password to enrol again. Only the newest link of an invite works. Not cached. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [AcceptInviteRequest] acceptInviteRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> acceptInviteWithHttpInfo(
    AcceptInviteRequest acceptInviteRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/invites/accept';

    // ignore: prefer_final_locals
    Object? postBody = acceptInviteRequest;

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

  /// Accept an invite
  ///
  /// With the token from the link's fragment and a password: creates the account (and, for a practitioner, the Practitioner and PractitionerRole) and returns a pending TOTP secret with a challenge. Confirm the first code with `/auth/mfa/confirm` and the challenge (valid 5 minutes) to finish enrolment and open the session; if the challenge lapses, sign in with the password to enrol again. Only the newest link of an invite works. Not cached. Audited.
  ///
  /// Parameters:
  ///
  /// * [AcceptInviteRequest] acceptInviteRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Acceptance?> acceptInvite(
    AcceptInviteRequest acceptInviteRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await acceptInviteWithHttpInfo(
      acceptInviteRequest,
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
        'Acceptance',
      ) as Acceptance;
    }
    return null;
  }

  /// Invite a practitioner or an admin to the clinic
  ///
  /// An admin of the clinic invites someone by e-mail: a practitioner with their given and family name, or another admin. The invite is open for 72 hours; an open invite for the same e-mail in the clinic is replaced. The e-mail with the link (`<web>/invite#<token>`) is sent by the notifier. Whether the e-mail already has an account is not disclosed here. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [InviteRequest] inviteRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> createInviteWithHttpInfo(
    InviteRequest inviteRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/invites';

    // ignore: prefer_final_locals
    Object? postBody = inviteRequest;

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

  /// Invite a practitioner or an admin to the clinic
  ///
  /// An admin of the clinic invites someone by e-mail: a practitioner with their given and family name, or another admin. The invite is open for 72 hours; an open invite for the same e-mail in the clinic is replaced. The e-mail with the link (`<web>/invite#<token>`) is sent by the notifier. Whether the e-mail already has an account is not disclosed here. Audited.
  ///
  /// Parameters:
  ///
  /// * [InviteRequest] inviteRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Invite?> createInvite(
    InviteRequest inviteRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await createInviteWithHttpInfo(
      inviteRequest,
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
        'Invite',
      ) as Invite;
    }
    return null;
  }

  /// The clinic's open invites
  ///
  /// Invites of the admin's clinic that are neither accepted nor expired, oldest first. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> listInvitesWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/invites';

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

  /// The clinic's open invites
  ///
  /// Invites of the admin's clinic that are neither accepted nor expired, oldest first. Audited.
  Future<InviteList?> listInvites({
    Future<void>? abortTrigger,
  }) async {
    final response = await listInvitesWithHttpInfo(
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
        'InviteList',
      ) as InviteList;
    }
    return null;
  }
}
