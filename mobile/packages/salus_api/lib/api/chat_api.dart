//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ChatApi {
  ChatApi([ApiClient? apiClient]) : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// Add a practitioner of the clinic, or invite one of another clinic (SAL-28)
  ///
  /// The conversation's treating practitioner only, while they may write in it; never into a side thread. A role of the clinic, active, is added at once (`status` `active`); a role of another clinic in the public directory is invited (`pending_consent`) and sees nothing of the conversation, its participants, messages or side threads until the patient grants it (`POST /me/chat-invitations/{id}/decision`). `history` `full` gives the whole history, `from` the messages after `from_seq` (at most the conversation's `last_seq`). The conversation gets the system message `participant_added` or `participant_invited` (`practitioner_role_id`, `organization_id`, `reason`), on the socket like any message. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  ///
  /// * [AddParticipantRequest] addParticipantRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> addConversationParticipantWithHttpInfo(
    String id,
    AddParticipantRequest addParticipantRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/conversations/{id}/participants'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = addParticipantRequest;

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

  /// Add a practitioner of the clinic, or invite one of another clinic (SAL-28)
  ///
  /// The conversation's treating practitioner only, while they may write in it; never into a side thread. A role of the clinic, active, is added at once (`status` `active`); a role of another clinic in the public directory is invited (`pending_consent`) and sees nothing of the conversation, its participants, messages or side threads until the patient grants it (`POST /me/chat-invitations/{id}/decision`). `history` `full` gives the whole history, `from` the messages after `from_seq` (at most the conversation's `last_seq`). The conversation gets the system message `participant_added` or `participant_invited` (`practitioner_role_id`, `organization_id`, `reason`), on the socket like any message. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  ///
  /// * [AddParticipantRequest] addParticipantRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<ConversationParticipant?> addConversationParticipant(
    String id,
    AddParticipantRequest addParticipantRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await addConversationParticipantWithHttpInfo(
      id,
      addParticipantRequest,
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
        'ConversationParticipant',
      ) as ConversationParticipant;
    }
    return null;
  }

  /// Add a webhook endpoint; its secret is shown in this answer only
  ///
  /// The clinic's own system receives each event as a `WebhookEvent`, POSTed as JSON with the headers `Salus-Event-Id` (the event's id, the same on every attempt: deduplicate on it) and `Salus-Signature`: `t=<unix seconds>,v1=<hex HMAC-SHA256 of \"<t>.<body>\" with the secret>` (check it, and refuse an old `t`). A status other than 2xx, or no answer within 10 seconds, is retried with backoff, 8 attempts in about an hour and a half, then recorded as a dead letter. `url` is https to a public host; addresses that are not public are refused at every connection. `include_clinical` (off by default) adds the message's text: the clinic is the controller of what it receives. At most 10 endpoints. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [NewWebhook] newWebhook (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> createWebhookWithHttpInfo(
    NewWebhook newWebhook, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/webhooks';

    // ignore: prefer_final_locals
    Object? postBody = newWebhook;

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

  /// Add a webhook endpoint; its secret is shown in this answer only
  ///
  /// The clinic's own system receives each event as a `WebhookEvent`, POSTed as JSON with the headers `Salus-Event-Id` (the event's id, the same on every attempt: deduplicate on it) and `Salus-Signature`: `t=<unix seconds>,v1=<hex HMAC-SHA256 of \"<t>.<body>\" with the secret>` (check it, and refuse an old `t`). A status other than 2xx, or no answer within 10 seconds, is retried with backoff, 8 attempts in about an hour and a half, then recorded as a dead letter. `url` is https to a public host; addresses that are not public are refused at every connection. `include_clinical` (off by default) adds the message's text: the clinic is the controller of what it receives. At most 10 endpoints. Audited.
  ///
  /// Parameters:
  ///
  /// * [NewWebhook] newWebhook (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<CreatedWebhook?> createWebhook(
    NewWebhook newWebhook, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await createWebhookWithHttpInfo(
      newWebhook,
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
        'CreatedWebhook',
      ) as CreatedWebhook;
    }
    return null;
  }

  /// Grant or decline an invitation (SAL-28)
  ///
  /// The patient's decision. `grant` records their FHIR Consent for that conversation and that practitioner role (revocable), needs their active care consent with the clinic (403 `consent_required` otherwise) and makes the doctor an active participant; `decline` ends the invitation. System message `consent_granted` or `consent_declined` (`practitioner_role_id`). Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The invitation's id (the invited participant's).
  ///
  /// * [ChatInvitationDecision] chatInvitationDecision (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> decideChatInvitationWithHttpInfo(
    String id,
    ChatInvitationDecision chatInvitationDecision, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/me/chat-invitations/{id}/decision'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = chatInvitationDecision;

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

  /// Grant or decline an invitation (SAL-28)
  ///
  /// The patient's decision. `grant` records their FHIR Consent for that conversation and that practitioner role (revocable), needs their active care consent with the clinic (403 `consent_required` otherwise) and makes the doctor an active participant; `decline` ends the invitation. System message `consent_granted` or `consent_declined` (`practitioner_role_id`). Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The invitation's id (the invited participant's).
  ///
  /// * [ChatInvitationDecision] chatInvitationDecision (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<void> decideChatInvitation(
    String id,
    ChatInvitationDecision chatInvitationDecision, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await decideChatInvitationWithHttpInfo(
      id,
      chatInvitationDecision,
      xSalusCSRF: xSalusCSRF,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
  }

  /// Delete a webhook endpoint and its dead letters
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The endpoint's id.
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> deleteWebhookWithHttpInfo(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/webhooks/{id}'.replaceAll('{id}', id);

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

  /// Delete a webhook endpoint and its dead letters
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The endpoint's id.
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<void> deleteWebhook(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await deleteWebhookWithHttpInfo(
      id,
      xSalusCSRF: xSalusCSRF,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
  }

  /// Edit the body of one's own text message
  ///
  /// `If-Match` is the message's version (its ETag). The earlier body is kept (`.../revisions`); the edit goes to every open socket as `message_edited`. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  ///
  /// * [String] messageId (required):
  ///   The message's id (FHIR Communication.id).
  ///
  /// * [String] ifMatch (required):
  ///
  /// * [EditMessageRequest] editMessageRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> editMessageWithHttpInfo(
    String id,
    String messageId,
    String ifMatch,
    EditMessageRequest editMessageRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/conversations/{id}/messages/{message_id}'
        .replaceAll('{id}', id)
        .replaceAll('{message_id}', messageId);

    // ignore: prefer_final_locals
    Object? postBody = editMessageRequest;

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
      'PATCH',
      queryParams,
      postBody,
      headerParams,
      formParams,
      contentTypes.isEmpty ? null : contentTypes.first,
      abortTrigger: abortTrigger,
    );
  }

  /// Edit the body of one's own text message
  ///
  /// `If-Match` is the message's version (its ETag). The earlier body is kept (`.../revisions`); the edit goes to every open socket as `message_edited`. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  ///
  /// * [String] messageId (required):
  ///   The message's id (FHIR Communication.id).
  ///
  /// * [String] ifMatch (required):
  ///
  /// * [EditMessageRequest] editMessageRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<ChatMessage?> editMessage(
    String id,
    String messageId,
    String ifMatch,
    EditMessageRequest editMessageRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await editMessageWithHttpInfo(
      id,
      messageId,
      ifMatch,
      editMessageRequest,
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
        'ChatMessage',
      ) as ChatMessage;
    }
    return null;
  }

  /// The conversation's messages, live, over a WebSocket
  ///
  /// A WebSocket (RFC 6455) upgrade (ADR 0012). Every message after `after` is replayed from the record, in order, then the live ones follow as `message` frames, and edits as `message_edited` (`ChatFrame`); a client drops a message whose `seq` it has. Without `after`, only messages sent from now on. The handshake must come from the API's own origin, or carry no Origin (the mobile app, with its bearer token). Access is checked again at least every 5 seconds, the session too: when it ends the socket closes with 4401 `unauthenticated` (a logout, a revoked or expired session), 4403 `consent_required` or `forbidden`, or 4404 `not_found` (staff after the patient's revocation); 1011 on a server error. The subscription is audited. The client may send `send` frames (`ChatCommand`), answered with `send_ack`, or `command_error` with the code of the JSON API, or `rate_limited` past five at once and one every 200 ms. Pass the conversation's or the last message's `cursor` as `after`.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  ///
  /// * [String] after:
  ///   The `cursor` of the last message the client has.
  Future<Response> followConversationWithHttpInfo(
    String id, {
    String? after,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/conversations/{id}/events'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    if (after != null) {
      queryParams.addAll(_queryParams('', 'after', after));
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

  /// The conversation's messages, live, over a WebSocket
  ///
  /// A WebSocket (RFC 6455) upgrade (ADR 0012). Every message after `after` is replayed from the record, in order, then the live ones follow as `message` frames, and edits as `message_edited` (`ChatFrame`); a client drops a message whose `seq` it has. Without `after`, only messages sent from now on. The handshake must come from the API's own origin, or carry no Origin (the mobile app, with its bearer token). Access is checked again at least every 5 seconds, the session too: when it ends the socket closes with 4401 `unauthenticated` (a logout, a revoked or expired session), 4403 `consent_required` or `forbidden`, or 4404 `not_found` (staff after the patient's revocation); 1011 on a server error. The subscription is audited. The client may send `send` frames (`ChatCommand`), answered with `send_ack`, or `command_error` with the code of the JSON API, or `rate_limited` past five at once and one every 200 ms. Pass the conversation's or the last message's `cursor` as `after`.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  ///
  /// * [String] after:
  ///   The `cursor` of the last message the client has.
  Future<void> followConversation(
    String id, {
    String? after,
    Future<void>? abortTrigger,
  }) async {
    final response = await followConversationWithHttpInfo(
      id,
      after: after,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
  }

  /// The clinic's chat hours, its response time, and whether it is within its hours now
  ///
  /// For the clinic's staff and for a patient with an active care consent with it; anyone else gets 404. `hours` are in the clinic's `time_zone`; `open_now` is computed there (true when the clinic states no hours). The app shows its out-of-hours notice from it.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The clinic's id.
  Future<Response> getChatHoursWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/organizations/{id}/chat-hours'.replaceAll('{id}', id);

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

  /// The clinic's chat hours, its response time, and whether it is within its hours now
  ///
  /// For the clinic's staff and for a patient with an active care consent with it; anyone else gets 404. `hours` are in the clinic's `time_zone`; `open_now` is computed there (true when the clinic states no hours). The app shows its out-of-hours notice from it.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The clinic's id.
  Future<ChatHours?> getChatHours(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getChatHoursWithHttpInfo(
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
        'ChatHours',
      ) as ChatHours;
    }
    return null;
  }

  /// The admin's clinic's chat settings
  ///
  /// The defaults (no hours, 24 hours, 60 minutes) until the admin sets them.
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> getChatSettingsWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/chat-settings';

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

  /// The admin's clinic's chat settings
  ///
  /// The defaults (no hours, 24 hours, 60 minutes) until the admin sets them.
  Future<ChatSettings?> getChatSettings({
    Future<void>? abortTrigger,
  }) async {
    final response = await getChatSettingsWithHttpInfo(
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
        'ChatSettings',
      ) as ChatSettings;
    }
    return null;
  }

  /// One conversation with its active participants
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  Future<Response> getConversationWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/conversations/{id}'.replaceAll('{id}', id);

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

  /// One conversation with its active participants
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  Future<Conversation?> getConversation(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getConversationWithHttpInfo(
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
        'Conversation',
      ) as Conversation;
    }
    return null;
  }

  /// The patient's invitations of doctors from other clinics, pending or granted (SAL-28)
  ///
  /// Oldest first: the pending invitations (`status` `pending_consent`, the default), or the granted ones (`status` `active`), whose `id` `POST /me/chat-invitations/{id}/revoke` takes, so a consent is withdrawn as easily as it was given (GDPR Art. 7(3)). A doctor of the conversation's own clinic is never one. The patient only (403 to staff). Audited as one search.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] status:
  ///   Which invitations; `pending_consent` when absent.
  Future<Response> listChatInvitationsWithHttpInfo({
    String? status,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/me/chat-invitations';

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

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

  /// The patient's invitations of doctors from other clinics, pending or granted (SAL-28)
  ///
  /// Oldest first: the pending invitations (`status` `pending_consent`, the default), or the granted ones (`status` `active`), whose `id` `POST /me/chat-invitations/{id}/revoke` takes, so a consent is withdrawn as easily as it was given (GDPR Art. 7(3)). A doctor of the conversation's own clinic is never one. The patient only (403 to staff). Audited as one search.
  ///
  /// Parameters:
  ///
  /// * [String] status:
  ///   Which invitations; `pending_consent` when absent.
  Future<ChatInvitationList?> listChatInvitations({
    String? status,
    Future<void>? abortTrigger,
  }) async {
    final response = await listChatInvitationsWithHttpInfo(
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
        'ChatInvitationList',
      ) as ChatInvitationList;
    }
    return null;
  }

  /// The side threads of the conversation the caller takes part in, newest first (SAL-28)
  ///
  /// For the practitioners of `POST`; never the patient (403). With the caller's unread counts. Audited as one search.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  Future<Response> listChatThreadsWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/conversations/{id}/threads'.replaceAll('{id}', id);

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

  /// The side threads of the conversation the caller takes part in, newest first (SAL-28)
  ///
  /// For the practitioners of `POST`; never the patient (403). With the caller's unread counts. Audited as one search.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  Future<ConversationList?> listChatThreads(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await listChatThreadsWithHttpInfo(
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
        'ConversationList',
      ) as ConversationList;
    }
    return null;
  }

  /// The caller's conversations, newest activity first, with their unread counts
  ///
  /// The conversations the caller takes part in, 50 a page; a practitioner's only while the patient's care consent is active. Audited as one search.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] before:
  ///   The `next` of the previous page.
  Future<Response> listConversationsWithHttpInfo({
    String? before,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/conversations';

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    if (before != null) {
      queryParams.addAll(_queryParams('', 'before', before));
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

  /// The caller's conversations, newest activity first, with their unread counts
  ///
  /// The conversations the caller takes part in, 50 a page; a practitioner's only while the patient's care consent is active. Audited as one search.
  ///
  /// Parameters:
  ///
  /// * [String] before:
  ///   The `next` of the previous page.
  Future<ConversationList?> listConversations({
    String? before,
    Future<void>? abortTrigger,
  }) async {
    final response = await listConversationsWithHttpInfo(
      before: before,
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
        'ConversationList',
      ) as ConversationList;
    }
    return null;
  }

  /// The earlier bodies of an edited message, oldest first
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  ///
  /// * [String] messageId (required):
  ///   The message's id (FHIR Communication.id).
  Future<Response> listMessageRevisionsWithHttpInfo(
    String id,
    String messageId, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/conversations/{id}/messages/{message_id}/revisions'
        .replaceAll('{id}', id)
        .replaceAll('{message_id}', messageId);

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

  /// The earlier bodies of an edited message, oldest first
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  ///
  /// * [String] messageId (required):
  ///   The message's id (FHIR Communication.id).
  Future<MessageRevisions?> listMessageRevisions(
    String id,
    String messageId, {
    Future<void>? abortTrigger,
  }) async {
    final response = await listMessageRevisionsWithHttpInfo(
      id,
      messageId,
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
        'MessageRevisions',
      ) as MessageRevisions;
    }
    return null;
  }

  /// A page of the conversation's messages, newest first
  ///
  /// Before the `before` cursor, or the newest; `next` is the cursor of the older page and absent at the start. A practitioner added later reads from their `history_from_seq`. Audited as one read of the rows returned.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  ///
  /// * [String] before:
  ///   A message's `cursor`, or a page's `next`.
  ///
  /// * [int] limit:
  Future<Response> listMessagesWithHttpInfo(
    String id, {
    String? before,
    int? limit,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/conversations/{id}/messages'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    if (before != null) {
      queryParams.addAll(_queryParams('', 'before', before));
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

  /// A page of the conversation's messages, newest first
  ///
  /// Before the `before` cursor, or the newest; `next` is the cursor of the older page and absent at the start. A practitioner added later reads from their `history_from_seq`. Audited as one read of the rows returned.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  ///
  /// * [String] before:
  ///   A message's `cursor`, or a page's `next`.
  ///
  /// * [int] limit:
  Future<MessagePage?> listMessages(
    String id, {
    String? before,
    int? limit,
    Future<void>? abortTrigger,
  }) async {
    final response = await listMessagesWithHttpInfo(
      id,
      before: before,
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
        'MessagePage',
      ) as MessagePage;
    }
    return null;
  }

  /// The admin's clinic's webhook endpoints, without their secrets
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> listWebhooksWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/webhooks';

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

  /// The admin's clinic's webhook endpoints, without their secrets
  Future<WebhookList?> listWebhooks({
    Future<void>? abortTrigger,
  }) async {
    final response = await listWebhooksWithHttpInfo(
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
        'WebhookList',
      ) as WebhookList;
    }
    return null;
  }

  /// Move the caller's read position
  ///
  /// The caller has read the messages up to `seq`; the position never goes back. Not audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  ///
  /// * [ReadPosition] readPosition (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> markConversationReadWithHttpInfo(
    String id,
    ReadPosition readPosition, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/conversations/{id}/read'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = readPosition;

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

  /// Move the caller's read position
  ///
  /// The caller has read the messages up to `seq`; the position never goes back. Not audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  ///
  /// * [ReadPosition] readPosition (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<void> markConversationRead(
    String id,
    ReadPosition readPosition, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await markConversationReadWithHttpInfo(
      id,
      readPosition,
      xSalusCSRF: xSalusCSRF,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
  }

  /// Open a doctor-only side thread of the conversation (SAL-28)
  ///
  /// For a practitioner who may write in the conversation, or a doctor of another clinic the patient granted. The side thread is a conversation (`staff_only`, `parent_id`) whose participants are the conversation's active practitioners now: a doctor added later is not in it, one removed from the conversation leaves it. Its messages, socket and read position are those of any conversation. The patient never sees it: 404 on every path of the side thread, 403 here. No body. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> openChatThreadWithHttpInfo(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/conversations/{id}/threads'.replaceAll('{id}', id);

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

  /// Open a doctor-only side thread of the conversation (SAL-28)
  ///
  /// For a practitioner who may write in the conversation, or a doctor of another clinic the patient granted. The side thread is a conversation (`staff_only`, `parent_id`) whose participants are the conversation's active practitioners now: a doctor added later is not in it, one removed from the conversation leaves it. Its messages, socket and read position are those of any conversation. The patient never sees it: 404 on every path of the side thread, 403 here. No body. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Conversation?> openChatThread(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await openChatThreadWithHttpInfo(
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
        'Conversation',
      ) as Conversation;
    }
    return null;
  }

  /// Open the conversation with a patient or a practitioner, or return the one there is
  ///
  /// Staff name the patient and open it with their own role; a patient names the clinic and the practitioner role. Allowed with the patient's active care consent with the clinic and an appointment of the patient with that role, of any time or status (docs/plans/chat.md). One conversation per patient, role and encounter (none counts as one): opening it again answers 200 with it. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [OpenConversationRequest] openConversationRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> openConversationWithHttpInfo(
    OpenConversationRequest openConversationRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/conversations';

    // ignore: prefer_final_locals
    Object? postBody = openConversationRequest;

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

  /// Open the conversation with a patient or a practitioner, or return the one there is
  ///
  /// Staff name the patient and open it with their own role; a patient names the clinic and the practitioner role. Allowed with the patient's active care consent with the clinic and an appointment of the patient with that role, of any time or status (docs/plans/chat.md). One conversation per patient, role and encounter (none counts as one): opening it again answers 200 with it. Audited.
  ///
  /// Parameters:
  ///
  /// * [OpenConversationRequest] openConversationRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Conversation?> openConversation(
    OpenConversationRequest openConversationRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await openConversationWithHttpInfo(
      openConversationRequest,
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
        'Conversation',
      ) as Conversation;
    }
    return null;
  }

  /// Register the patient app's push token
  ///
  /// The patient app registers its FCM (Android) or APNs (iOS) token at every start (docs/plans/chat.md, SAL-29). A token is one device: registered by another account, it moves to this one. A push says only that there is a new message from the clinic; it carries no text, name or id. Staff get 403: the app is the patients'. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [PushDevice] pushDevice (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> registerDeviceWithHttpInfo(
    PushDevice pushDevice, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/me/devices';

    // ignore: prefer_final_locals
    Object? postBody = pushDevice;

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

  /// Register the patient app's push token
  ///
  /// The patient app registers its FCM (Android) or APNs (iOS) token at every start (docs/plans/chat.md, SAL-29). A token is one device: registered by another account, it moves to this one. A push says only that there is a new message from the clinic; it carries no text, name or id. Staff get 403: the app is the patients'. Audited.
  ///
  /// Parameters:
  ///
  /// * [PushDevice] pushDevice (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<void> registerDevice(
    PushDevice pushDevice, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await registerDeviceWithHttpInfo(
      pushDevice,
      xSalusCSRF: xSalusCSRF,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
  }

  /// Remove a practitioner, or withdraw an invitation (SAL-28)
  ///
  /// The treating practitioner only; never the patient or themselves. The practitioner leaves the conversation's side threads with it and sees nothing of it from then on. System message `participant_removed` (`practitioner_role_id`). Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  ///
  /// * [String] participantId (required):
  ///   The participant's id.
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> removeConversationParticipantWithHttpInfo(
    String id,
    String participantId, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/conversations/{id}/participants/{participant_id}'
        .replaceAll('{id}', id)
        .replaceAll('{participant_id}', participantId);

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

  /// Remove a practitioner, or withdraw an invitation (SAL-28)
  ///
  /// The treating practitioner only; never the patient or themselves. The practitioner leaves the conversation's side threads with it and sees nothing of it from then on. System message `participant_removed` (`practitioner_role_id`). Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  ///
  /// * [String] participantId (required):
  ///   The participant's id.
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<void> removeConversationParticipant(
    String id,
    String participantId, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await removeConversationParticipantWithHttpInfo(
      id,
      participantId,
      xSalusCSRF: xSalusCSRF,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
  }

  /// Revoke a granted invitation (SAL-28)
  ///
  /// The patient's chat consent for the doctor ends and the doctor leaves the conversation and its side threads: they see nothing of it from then on, and an open socket of theirs closes with 4404 at its next check (within 5 seconds). System message `participant_removed` (`practitioner_role_id`). Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The invitation's id (the invited participant's).
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> revokeChatInvitationWithHttpInfo(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/me/chat-invitations/{id}/revoke'.replaceAll('{id}', id);

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

  /// Revoke a granted invitation (SAL-28)
  ///
  /// The patient's chat consent for the doctor ends and the doctor leaves the conversation and its side threads: they see nothing of it from then on, and an open socket of theirs closes with 4404 at its next check (within 5 seconds). System message `participant_removed` (`practitioner_role_id`). Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The invitation's id (the invited participant's).
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<void> revokeChatInvitation(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await revokeChatInvitationWithHttpInfo(
      id,
      xSalusCSRF: xSalusCSRF,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
  }

  /// Send a text message
  ///
  /// Idempotent by `client_message_id` per sender: a repeat answers 200 with the first message, which is not sent again. The patient writes while their care consent is active; a practitioner while they take part and the consent is active. Audited; the message goes to every open socket of the conversation.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  ///
  /// * [SendMessageRequest] sendMessageRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> sendMessageWithHttpInfo(
    String id,
    SendMessageRequest sendMessageRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/conversations/{id}/messages'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = sendMessageRequest;

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

  /// Send a text message
  ///
  /// Idempotent by `client_message_id` per sender: a repeat answers 200 with the first message, which is not sent again. The patient writes while their care consent is active; a practitioner while they take part and the consent is active. Audited; the message goes to every open socket of the conversation.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  ///
  /// * [SendMessageRequest] sendMessageRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<ChatMessage?> sendMessage(
    String id,
    SendMessageRequest sendMessageRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await sendMessageWithHttpInfo(
      id,
      sendMessageRequest,
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
        'ChatMessage',
      ) as ChatMessage;
    }
    return null;
  }

  /// Replace the admin's clinic's chat settings
  ///
  /// Every field required. `hours`: up to 21 stretches, each within one weekday, `start` before `end` (`24:00` for midnight), none overlapping. `response_time_hours` 1 to 168. `email_after_minutes` 5 to 10080: how long a message stays unread before its reader gets an e-mail. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [ChatSettings] chatSettings (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> setChatSettingsWithHttpInfo(
    ChatSettings chatSettings, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/chat-settings';

    // ignore: prefer_final_locals
    Object? postBody = chatSettings;

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

  /// Replace the admin's clinic's chat settings
  ///
  /// Every field required. `hours`: up to 21 stretches, each within one weekday, `start` before `end` (`24:00` for midnight), none overlapping. `response_time_hours` 1 to 168. `email_after_minutes` 5 to 10080: how long a message stays unread before its reader gets an e-mail. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [ChatSettings] chatSettings (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<ChatSettings?> setChatSettings(
    ChatSettings chatSettings, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await setChatSettingsWithHttpInfo(
      chatSettings,
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
        'ChatSettings',
      ) as ChatSettings;
    }
    return null;
  }

  /// Unregister the app's push token (on sign-out)
  ///
  /// Deletes the account's device of the token; nothing for a token it does not have. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [PushDevice] pushDevice (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> unregisterDeviceWithHttpInfo(
    PushDevice pushDevice, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/me/devices';

    // ignore: prefer_final_locals
    Object? postBody = pushDevice;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    if (xSalusCSRF != null) {
      headerParams[r'X-Salus-CSRF'] = parameterToString(xSalusCSRF);
    }

    const contentTypes = <String>['application/json'];

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

  /// Unregister the app's push token (on sign-out)
  ///
  /// Deletes the account's device of the token; nothing for a token it does not have. Audited.
  ///
  /// Parameters:
  ///
  /// * [PushDevice] pushDevice (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<void> unregisterDevice(
    PushDevice pushDevice, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await unregisterDeviceWithHttpInfo(
      pushDevice,
      xSalusCSRF: xSalusCSRF,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
  }
}
