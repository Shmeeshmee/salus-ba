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
  /// A WebSocket (RFC 6455) upgrade (ADR 0012). Every message after `after` is replayed from the record, in order, then the live ones follow as `message` frames, and edits as `message_edited` (`ChatFrame`); a client drops a message whose `seq` it has. Without `after`, only messages sent from now on. The handshake must come from the API's own origin, or carry no Origin (the mobile app, with its bearer token). Access is checked again at least every 5 seconds, the session too: when it ends the socket closes with 4401 `unauthenticated` (a logout, a revoked or expired session), 4403 `consent_required` or `forbidden`, or 4404 `not_found` (staff after the patient's revocation); 1011 on a server error. The subscription is audited. The client may send `send` frames (`ChatCommand`), answered with `send_ack`, or `command_error` with the code of the JSON API, or `rate_limited` past five at once and one every 200 ms. Pass the conversation's or the last message's `cursor` as `after`. Receipts and typing (SAL-30), ephemeral: once replayed, the socket sends a `receipt` frame per participant who has read (as kept for 7 days), then the others' `receipt` and `typing` frames as they come; a client drops its own. The client's `read` frame (`ChatReadCommand`) moves its read position as `markConversationRead` does and counts against the same limit as `send`; its `typing` frame (`ChatTypingCommand`) goes out at most once a second; neither is answered unless refused. An app shows a participant typing for 5 seconds after their last `typing` frame. Message frames carry their `attachments` but no `terms` (the JSON API's reads do).
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
  /// A WebSocket (RFC 6455) upgrade (ADR 0012). Every message after `after` is replayed from the record, in order, then the live ones follow as `message` frames, and edits as `message_edited` (`ChatFrame`); a client drops a message whose `seq` it has. Without `after`, only messages sent from now on. The handshake must come from the API's own origin, or carry no Origin (the mobile app, with its bearer token). Access is checked again at least every 5 seconds, the session too: when it ends the socket closes with 4401 `unauthenticated` (a logout, a revoked or expired session), 4403 `consent_required` or `forbidden`, or 4404 `not_found` (staff after the patient's revocation); 1011 on a server error. The subscription is audited. The client may send `send` frames (`ChatCommand`), answered with `send_ack`, or `command_error` with the code of the JSON API, or `rate_limited` past five at once and one every 200 ms. Pass the conversation's or the last message's `cursor` as `after`. Receipts and typing (SAL-30), ephemeral: once replayed, the socket sends a `receipt` frame per participant who has read (as kept for 7 days), then the others' `receipt` and `typing` frames as they come; a client drops its own. The client's `read` frame (`ChatReadCommand`) moves its read position as `markConversationRead` does and counts against the same limit as `send`; its `typing` frame (`ChatTypingCommand`) goes out at most once a second; neither is answered unless refused. An app shows a participant typing for 5 seconds after their last `typing` frame. Message frames carry their `attachments` but no `terms` (the JSON API's reads do).
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

  /// Download a message's file
  ///
  /// The file's bytes, decrypted, once the scan says it is `clean`, for whoever may read its message (a participant who joined later only from their starting point; staff never after the patient's revocation), or, before it is sent, for its uploader. One `Range` gives 206 with that part. Shown inline, never sniffed, never cached. Every request is audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  ///
  /// * [String] attachmentId (required):
  ///
  /// * [String] range:
  ///   One byte range, `bytes=<first>-<last>`, `bytes=<first>-` or `bytes=-<suffix length>`.
  Future<Response> getChatAttachmentContentWithHttpInfo(
    String id,
    String attachmentId, {
    String? range,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/conversations/{id}/attachments/{attachment_id}/content'
        .replaceAll('{id}', id)
        .replaceAll('{attachment_id}', attachmentId);

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    if (range != null) {
      headerParams[r'Range'] = parameterToString(range);
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

  /// Download a message's file
  ///
  /// The file's bytes, decrypted, once the scan says it is `clean`, for whoever may read its message (a participant who joined later only from their starting point; staff never after the patient's revocation), or, before it is sent, for its uploader. One `Range` gives 206 with that part. Shown inline, never sniffed, never cached. Every request is audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  ///
  /// * [String] attachmentId (required):
  ///
  /// * [String] range:
  ///   One byte range, `bytes=<first>-<last>`, `bytes=<first>-` or `bytes=-<suffix length>`.
  Future<MultipartFile?> getChatAttachmentContent(
    String id,
    String attachmentId, {
    String? range,
    Future<void>? abortTrigger,
  }) async {
    final response = await getChatAttachmentContentWithHttpInfo(
      id,
      attachmentId,
      range: range,
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
        'MultipartFile',
      ) as MultipartFile;
    }
    return null;
  }

  /// Download a message's image thumbnail
  ///
  /// A JPEG of at most 320 pixels a side, made by the server from the clean image (`thumbnail` true on the attachment); none for a PDF (404). Read as the file is, audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  ///
  /// * [String] attachmentId (required):
  Future<Response> getChatAttachmentThumbnailWithHttpInfo(
    String id,
    String attachmentId, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/conversations/{id}/attachments/{attachment_id}/thumbnail'
        .replaceAll('{id}', id)
        .replaceAll('{attachment_id}', attachmentId);

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

  /// Download a message's image thumbnail
  ///
  /// A JPEG of at most 320 pixels a side, made by the server from the clean image (`thumbnail` true on the attachment); none for a PDF (404). Read as the file is, audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  ///
  /// * [String] attachmentId (required):
  Future<MultipartFile?> getChatAttachmentThumbnail(
    String id,
    String attachmentId, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getChatAttachmentThumbnailWithHttpInfo(
      id,
      attachmentId,
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
        'MultipartFile',
      ) as MultipartFile;
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

  /// Search a conversation's messages
  ///
  /// The messages of the conversation whose text has every word of `q`, each as a word's beginning, case and accents aside, in any of the app's languages; newest first, at most 50; only those the caller reads. Audited as a search; `q` is never logged or stored.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  ///
  /// * [String] q (required):
  ///   Words to find, 1 to 200 characters: runs of letters and digits, of two or more, the first eight count; anything else separates them.
  Future<Response> searchConversationWithHttpInfo(
    String id,
    String q, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/conversations/{id}/search'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    queryParams.addAll(_queryParams('', 'q', q));

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

  /// Search a conversation's messages
  ///
  /// The messages of the conversation whose text has every word of `q`, each as a word's beginning, case and accents aside, in any of the app's languages; newest first, at most 50; only those the caller reads. Audited as a search; `q` is never logged or stored.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  ///
  /// * [String] q (required):
  ///   Words to find, 1 to 200 characters: runs of letters and digits, of two or more, the first eight count; anything else separates them.
  Future<ChatSearchResults?> searchConversation(
    String id,
    String q, {
    Future<void>? abortTrigger,
  }) async {
    final response = await searchConversationWithHttpInfo(
      id,
      q,
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
        'ChatSearchResults',
      ) as ChatSearchResults;
    }
    return null;
  }

  /// Search the caller's conversations
  ///
  /// As `searchConversation`, across the caller's conversations: a patient's own (never a doctor-only thread); a practitioner's, those they take part in, from their starting point, while the patient's care consent with the clinic is active. Audited as a search.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] q (required):
  ///   Words to find, 1 to 200 characters: runs of letters and digits, of two or more, the first eight count; anything else separates them.
  Future<Response> searchConversationsWithHttpInfo(
    String q, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/conversations/search';

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    queryParams.addAll(_queryParams('', 'q', q));

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

  /// Search the caller's conversations
  ///
  /// As `searchConversation`, across the caller's conversations: a patient's own (never a doctor-only thread); a practitioner's, those they take part in, from their starting point, while the patient's care consent with the clinic is active. Audited as a search.
  ///
  /// Parameters:
  ///
  /// * [String] q (required):
  ///   Words to find, 1 to 200 characters: runs of letters and digits, of two or more, the first eight count; anything else separates them.
  Future<ChatSearchResults?> searchConversations(
    String q, {
    Future<void>? abortTrigger,
  }) async {
    final response = await searchConversationsWithHttpInfo(
      q,
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
        'ChatSearchResults',
      ) as ChatSearchResults;
    }
    return null;
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

  /// Upload a file to send in the conversation
  ///
  /// A participant who may write uploads one file (SAL-30), then names it in `attachment_ids` of their next message. The body is the file's bytes, streamed; `Content-Length` is required. The type is detected from the bytes, never from `Content-Type` or the name: a photo (JPEG, PNG) or a PDF (lab results among them), at most 20 MB. The file name is the `filename*` of `Content-Disposition` (RFC 6266), cut to 200 characters, the path removed; never put it in the query string. The file is encrypted, stored and `scanning` until the malware scan says it is `clean` (an image then gets a thumbnail) or `rejected`; only its uploader sees it before it is sent. A participant has at most 10 files waiting to be sent (scanning or clean, not yet in a message). A file never sent is deleted after 24 hours. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  ///
  /// * [int] contentLength (required):
  ///   The file's size in bytes.
  ///
  /// * [MultipartFile] body (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  ///
  /// * [String] contentDisposition:
  ///   `attachment; filename*=UTF-8''<percent-encoded name>`, the file's name.
  Future<Response> uploadChatAttachmentWithHttpInfo(
    String id,
    int contentLength,
    MultipartFile body, {
    String? xSalusCSRF,
    String? contentDisposition,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/conversations/{id}/attachments'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = body;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    if (xSalusCSRF != null) {
      headerParams[r'X-Salus-CSRF'] = parameterToString(xSalusCSRF);
    }
    headerParams[r'Content-Length'] = parameterToString(contentLength);
    if (contentDisposition != null) {
      headerParams[r'Content-Disposition'] =
          parameterToString(contentDisposition);
    }

    const contentTypes = <String>['application/octet-stream'];

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

  /// Upload a file to send in the conversation
  ///
  /// A participant who may write uploads one file (SAL-30), then names it in `attachment_ids` of their next message. The body is the file's bytes, streamed; `Content-Length` is required. The type is detected from the bytes, never from `Content-Type` or the name: a photo (JPEG, PNG) or a PDF (lab results among them), at most 20 MB. The file name is the `filename*` of `Content-Disposition` (RFC 6266), cut to 200 characters, the path removed; never put it in the query string. The file is encrypted, stored and `scanning` until the malware scan says it is `clean` (an image then gets a thumbnail) or `rejected`; only its uploader sees it before it is sent. A participant has at most 10 files waiting to be sent (scanning or clean, not yet in a message). A file never sent is deleted after 24 hours. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The conversation's id.
  ///
  /// * [int] contentLength (required):
  ///   The file's size in bytes.
  ///
  /// * [MultipartFile] body (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  ///
  /// * [String] contentDisposition:
  ///   `attachment; filename*=UTF-8''<percent-encoded name>`, the file's name.
  Future<ChatAttachment?> uploadChatAttachment(
    String id,
    int contentLength,
    MultipartFile body, {
    String? xSalusCSRF,
    String? contentDisposition,
    Future<void>? abortTrigger,
  }) async {
    final response = await uploadChatAttachmentWithHttpInfo(
      id,
      contentLength,
      body,
      xSalusCSRF: xSalusCSRF,
      contentDisposition: contentDisposition,
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
        'ChatAttachment',
      ) as ChatAttachment;
    }
    return null;
  }
}
