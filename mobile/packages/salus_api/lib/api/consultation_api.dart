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
