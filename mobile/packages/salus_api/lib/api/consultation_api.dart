//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ConsultationApi {
  ConsultationApi([ApiClient? apiClient])
      : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// End an appointment's video consultation
  ///
  /// The appointment's practitioner ends the consultation once they joined (also after the patient revoked the consent): the room is closed for both, the Encounter finishes and the appointment becomes `fulfilled`, or `noshow` if the patient never joined (unless it was marked by hand). Repeated, it answers 200 again. Not cached. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> endConsultationWithHttpInfo(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/appointments/{id}/consultation/end'.replaceAll('{id}', id);

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

  /// End an appointment's video consultation
  ///
  /// The appointment's practitioner ends the consultation once they joined (also after the patient revoked the consent): the room is closed for both, the Encounter finishes and the appointment becomes `fulfilled`, or `noshow` if the patient never joined (unless it was marked by hand). Repeated, it answers 200 again. Not cached. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<ConsultationStatus?> endConsultation(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await endConsultationWithHttpInfo(
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
        'ConsultationStatus',
      ) as ConsultationStatus;
    }
    return null;
  }

  /// The consultation's stored transcript
  ///
  /// Every final segment of the encounter's transcript, in the order it was said, with its translation, as `TranscriptEvent`s (`is_final` true). For the encounter's practitioner with the patient's active consent only; the patient's own copy comes with the export feature. Not cached. Audited as a read with the number of segments, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The encounter's id (FHIR Encounter.id).
  Future<Response> exportTranscriptWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/encounters/{id}/transcript'.replaceAll('{id}', id);

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

  /// The consultation's stored transcript
  ///
  /// Every final segment of the encounter's transcript, in the order it was said, with its translation, as `TranscriptEvent`s (`is_final` true). For the encounter's practitioner with the patient's active consent only; the patient's own copy comes with the export feature. Not cached. Audited as a read with the number of segments, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The encounter's id (FHIR Encounter.id).
  Future<Transcript?> exportTranscript(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await exportTranscriptWithHttpInfo(
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
        'Transcript',
      ) as Transcript;
    }
    return null;
  }

  /// The consultation's live events, over a WebSocket
  ///
  /// A WebSocket (RFC 6455) upgrade: the encounter's Redis Stream, live, to the encounter's practitioner only, with the patient's active consent (not the clinic's admin, not the patient). Every entry of the stream is sent as two text frames: a `cursor` message with the entry's stream id, then the entry's message (`ConsultMessage`) exactly as its producer wrote it; together they are `ConsultFrame`. A client keeps the id of a cursor once the message after it has arrived, and on reconnecting passes it as `after`: it gets every entry after that one, in order, then the live ones, with no gap and no duplicate. Without `after` it gets every entry still on the stream (the stream is capped and expires after the encounter; the stored transcript is `GET /encounters/{id}/transcript`). The handshake must come from the API's own origin, or carry no Origin (no browser). Access is checked again after every read of the stream: when it ends (a revocation, a cancellation) nothing more is sent and the socket closes within a second, with code 4403 and reason `consent_required` or `forbidden`, or 4404 `not_found`; 1011 on a server error. Each subscription is audited as an operation on the encounter, also when refused. The client may send text frames of `ConsultCommand` (at most 512 bytes; a longer frame ends the socket): a `suggestion_status` message dismisses, pins or unpins a suggestion of the encounter, under the same rule, audited, refusals included. It is answered with nothing on success (the suggestion with its new status comes on the stream, to every open view), and with a `command_error` frame otherwise.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The encounter's id (FHIR Encounter.id).
  ///
  /// * [String] after:
  ///   The stream id of the last entry received (`<milliseconds>-<sequence>`), or `0` for all.
  Future<Response> followConsultationEventsWithHttpInfo(
    String id, {
    String? after,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/encounters/{id}/events'.replaceAll('{id}', id);

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

  /// The consultation's live events, over a WebSocket
  ///
  /// A WebSocket (RFC 6455) upgrade: the encounter's Redis Stream, live, to the encounter's practitioner only, with the patient's active consent (not the clinic's admin, not the patient). Every entry of the stream is sent as two text frames: a `cursor` message with the entry's stream id, then the entry's message (`ConsultMessage`) exactly as its producer wrote it; together they are `ConsultFrame`. A client keeps the id of a cursor once the message after it has arrived, and on reconnecting passes it as `after`: it gets every entry after that one, in order, then the live ones, with no gap and no duplicate. Without `after` it gets every entry still on the stream (the stream is capped and expires after the encounter; the stored transcript is `GET /encounters/{id}/transcript`). The handshake must come from the API's own origin, or carry no Origin (no browser). Access is checked again after every read of the stream: when it ends (a revocation, a cancellation) nothing more is sent and the socket closes within a second, with code 4403 and reason `consent_required` or `forbidden`, or 4404 `not_found`; 1011 on a server error. Each subscription is audited as an operation on the encounter, also when refused. The client may send text frames of `ConsultCommand` (at most 512 bytes; a longer frame ends the socket): a `suggestion_status` message dismisses, pins or unpins a suggestion of the encounter, under the same rule, audited, refusals included. It is answered with nothing on success (the suggestion with its new status comes on the stream, to every open view), and with a `command_error` frame otherwise.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The encounter's id (FHIR Encounter.id).
  ///
  /// * [String] after:
  ///   The stream id of the last entry received (`<milliseconds>-<sequence>`), or `0` for all.
  Future<void> followConsultationEvents(
    String id, {
    String? after,
    Future<void>? abortTrigger,
  }) async {
    final response = await followConsultationEventsWithHttpInfo(
      id,
      after: after,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
  }

  /// Where an appointment's video consultation stands
  ///
  /// The state of the consultation (`not_started` before anyone asked for a token, `open` while its room exists, `ended`), when the patient and the practitioner first joined, when and why it ended, and why the patient was removed. For the appointment's patient, its practitioner and the clinic's admin, as they may see the appointment (after a revocation, a past one too); and for its practitioner once they joined the room, also after a revocation hid the appointment from the clinic. The clients read `reason` when the other side leaves or the room closes. Not cached. Audited as a read of the appointment, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Response> getConsultationWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/appointments/{id}/consultation'.replaceAll('{id}', id);

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

  /// Where an appointment's video consultation stands
  ///
  /// The state of the consultation (`not_started` before anyone asked for a token, `open` while its room exists, `ended`), when the patient and the practitioner first joined, when and why it ended, and why the patient was removed. For the appointment's patient, its practitioner and the clinic's admin, as they may see the appointment (after a revocation, a past one too); and for its practitioner once they joined the room, also after a revocation hid the appointment from the clinic. The clients read `reason` when the other side leaves or the room closes. Not cached. Audited as a read of the appointment, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<ConsultationStatus?> getConsultation(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getConsultationWithHttpInfo(
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
        'ConsultationStatus',
      ) as ConsultationStatus;
    }
    return null;
  }

  /// A token to join an appointment's video consultation
  ///
  /// A LiveKit join token for the appointment's room, to its patient or to its practitioner only, while the appointment is booked, the patient's consent with the clinic is active, and the join window is open (10 minutes before the start to 30 minutes after the end). The token grants that one room, camera and microphone, for 5 minutes. The first request creates the room. `peerName` is the other person's name: the practitioner's public name for the patient, the patient's name for the practitioner (a read of the patient, audited). Not cached. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> joinConsultationWithHttpInfo(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/appointments/{id}/consultation/join'.replaceAll('{id}', id);

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

  /// A token to join an appointment's video consultation
  ///
  /// A LiveKit join token for the appointment's room, to its patient or to its practitioner only, while the appointment is booked, the patient's consent with the clinic is active, and the join window is open (10 minutes before the start to 30 minutes after the end). The token grants that one room, camera and microphone, for 5 minutes. The first request creates the room. `peerName` is the other person's name: the practitioner's public name for the patient, the patient's name for the practitioner (a read of the patient, audited). Not cached. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<ConsultationJoin?> joinConsultation(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await joinConsultationWithHttpInfo(
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
        'ConsultationJoin',
      ) as ConsultationJoin;
    }
    return null;
  }

  /// LiveKit's webhook
  ///
  /// Called only by our LiveKit server, never by the clients. No session and no CSRF header: the `Authorization` header is LiveKit's JWT, signed with the API key pair and carrying the body's SHA-256, verified before the body is read. The body is read up to 64 KiB. Handled idempotently and in any order: a participant's join is recorded (the practitioner's first one starts the Encounter; a patient whose access has gone is removed from the room), and once the join window has closed the last one leaving ends the consultation. Every other event, and one for a room we do not know, has no effect.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [LiveKitWebhookEvent] liveKitWebhookEvent (required):
  Future<Response> receiveLiveKitWebhookWithHttpInfo(
    LiveKitWebhookEvent liveKitWebhookEvent, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/webhooks/livekit';

    // ignore: prefer_final_locals
    Object? postBody = liveKitWebhookEvent;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    const contentTypes = <String>['application/webhook+json'];

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

  /// LiveKit's webhook
  ///
  /// Called only by our LiveKit server, never by the clients. No session and no CSRF header: the `Authorization` header is LiveKit's JWT, signed with the API key pair and carrying the body's SHA-256, verified before the body is read. The body is read up to 64 KiB. Handled idempotently and in any order: a participant's join is recorded (the practitioner's first one starts the Encounter; a patient whose access has gone is removed from the room), and once the join window has closed the last one leaving ends the consultation. Every other event, and one for a room we do not know, has no effect.
  ///
  /// Parameters:
  ///
  /// * [LiveKitWebhookEvent] liveKitWebhookEvent (required):
  Future<Object?> receiveLiveKitWebhook(
    LiveKitWebhookEvent liveKitWebhookEvent, {
    Future<void>? abortTrigger,
  }) async {
    final response = await receiveLiveKitWebhookWithHttpInfo(
      liveKitWebhookEvent,
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
        'Object',
      ) as Object;
    }
    return null;
  }
}
