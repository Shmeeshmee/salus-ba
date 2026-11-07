//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class AttachmentsApi {
  AttachmentsApi([ApiClient? apiClient])
      : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// Remove a file from an answer
  ///
  /// The patient removes a file from their questionnaire while they may still change its answers; the file is deleted. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The file's id (FHIR Binary.id).
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> deleteBinaryWithHttpInfo(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/binaries/{id}'.replaceAll('{id}', id);

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

  /// Remove a file from an answer
  ///
  /// The patient removes a file from their questionnaire while they may still change its answers; the file is deleted. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The file's id (FHIR Binary.id).
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<void> deleteBinary(
    String id, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await deleteBinaryWithHttpInfo(
      id,
      xSalusCSRF: xSalusCSRF,
      abortTrigger: abortTrigger,
    );
    if (response.statusCode >= HttpStatus.badRequest) {
      throw ApiException(response.statusCode, await _decodeBodyBytes(response));
    }
  }

  /// Read a file's status
  ///
  /// A file's type, size, title and scan status, for whoever may read its questionnaire response in full: the patient, and the appointment's practitioner once the response is submitted. The app polls it until the file is `clean` or `rejected`. Not cached. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The file's id (FHIR Binary.id).
  Future<Response> getBinaryWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/binaries/{id}'.replaceAll('{id}', id);

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

  /// Read a file's status
  ///
  /// A file's type, size, title and scan status, for whoever may read its questionnaire response in full: the patient, and the appointment's practitioner once the response is submitted. The app polls it until the file is `clean` or `rejected`. Not cached. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The file's id (FHIR Binary.id).
  Future<Binary?> getBinary(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getBinaryWithHttpInfo(
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
        'Binary',
      ) as Binary;
    }
    return null;
  }

  /// Download a file
  ///
  /// The file's bytes, decrypted, once the scan says it is `clean`, for whoever may read its questionnaire response in full. One `Range` (`bytes=a-b`, `a-` or `-n`) gives 206 with that part, so a video seeks; a request with several ranges gets the whole file. Shown inline, never sniffed, never cached. Every request is audited, ranges included.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The file's id (FHIR Binary.id).
  ///
  /// * [String] range:
  ///   One byte range, `bytes=<first>-<last>`, `bytes=<first>-` or `bytes=-<suffix length>`.
  Future<Response> getBinaryContentWithHttpInfo(
    String id, {
    String? range,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/binaries/{id}/content'.replaceAll('{id}', id);

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

  /// Download a file
  ///
  /// The file's bytes, decrypted, once the scan says it is `clean`, for whoever may read its questionnaire response in full. One `Range` (`bytes=a-b`, `a-` or `-n`) gives 206 with that part, so a video seeks; a request with several ranges gets the whole file. Shown inline, never sniffed, never cached. Every request is audited, ranges included.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The file's id (FHIR Binary.id).
  ///
  /// * [String] range:
  ///   One byte range, `bytes=<first>-<last>`, `bytes=<first>-` or `bytes=-<suffix length>`.
  Future<MultipartFile?> getBinaryContent(
    String id, {
    String? range,
    Future<void>? abortTrigger,
  }) async {
    final response = await getBinaryContentWithHttpInfo(
      id,
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

  /// Attach a file to an answer
  ///
  /// The patient attaches one file to an attachment item of their questionnaire while it is in progress, while the appointment is booked and before it starts, with an active consent (FHIR Binary; the answer is a valueAttachment). The body is the file's bytes, streamed; `Content-Length` is required. The type is detected from the bytes, never from `Content-Type` or the name: PDF, JPEG, PNG or MP4 (H.264 video, AAC audio), as the item accepts. Limits: 20 MB per photo or document, 200 MB per video, the item's `maxFiles`, 500 MB per response. The title is the `filename*` of `Content-Disposition` (RFC 6266), cut to 200 characters, the path removed; never put it in the query string. The file is encrypted, stored and `quarantined` until the malware scan says it is `clean` (`GET /binaries/{id}`). An upload identical to a file already on the item (same SHA-1 and size) returns that file with 200 and stores nothing new, so the app may retry the whole file. Not cached. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [int] contentLength (required):
  ///   The file's size in bytes.
  ///
  /// * [String] id (required):
  ///   The response's id (FHIR QuestionnaireResponse.id).
  ///
  /// * [String] linkId (required):
  ///   The attachment item's linkId.
  ///
  /// * [MultipartFile] body (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  ///
  /// * [String] contentDisposition:
  ///   `attachment; filename*=UTF-8''<percent-encoded name>`, the file's name as its title.
  Future<Response> uploadAttachmentWithHttpInfo(
    int contentLength,
    String id,
    String linkId,
    MultipartFile body, {
    String? xSalusCSRF,
    String? contentDisposition,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/questionnaire-responses/{id}/items/{linkId}/attachments'
        .replaceAll('{id}', id)
        .replaceAll('{linkId}', linkId);

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

  /// Attach a file to an answer
  ///
  /// The patient attaches one file to an attachment item of their questionnaire while it is in progress, while the appointment is booked and before it starts, with an active consent (FHIR Binary; the answer is a valueAttachment). The body is the file's bytes, streamed; `Content-Length` is required. The type is detected from the bytes, never from `Content-Type` or the name: PDF, JPEG, PNG or MP4 (H.264 video, AAC audio), as the item accepts. Limits: 20 MB per photo or document, 200 MB per video, the item's `maxFiles`, 500 MB per response. The title is the `filename*` of `Content-Disposition` (RFC 6266), cut to 200 characters, the path removed; never put it in the query string. The file is encrypted, stored and `quarantined` until the malware scan says it is `clean` (`GET /binaries/{id}`). An upload identical to a file already on the item (same SHA-1 and size) returns that file with 200 and stores nothing new, so the app may retry the whole file. Not cached. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [int] contentLength (required):
  ///   The file's size in bytes.
  ///
  /// * [String] id (required):
  ///   The response's id (FHIR QuestionnaireResponse.id).
  ///
  /// * [String] linkId (required):
  ///   The attachment item's linkId.
  ///
  /// * [MultipartFile] body (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  ///
  /// * [String] contentDisposition:
  ///   `attachment; filename*=UTF-8''<percent-encoded name>`, the file's name as its title.
  Future<Binary?> uploadAttachment(
    int contentLength,
    String id,
    String linkId,
    MultipartFile body, {
    String? xSalusCSRF,
    String? contentDisposition,
    Future<void>? abortTrigger,
  }) async {
    final response = await uploadAttachmentWithHttpInfo(
      contentLength,
      id,
      linkId,
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
        'Binary',
      ) as Binary;
    }
    return null;
  }
}
