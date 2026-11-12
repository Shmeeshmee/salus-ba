//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class DocumentsApi {
  DocumentsApi([ApiClient? apiClient])
      : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// Create a document template
  ///
  /// Creates a template of the caller's clinic as version 1. Staff of the clinic. A translation must be complete: its title, every field's label, and an instruction exactly where the original has one. Any language of the enum but Arabic, whose script the PDF does not set yet. The answer carries the version as its ETag. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [SaveDocumentTemplateRequest] saveDocumentTemplateRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> createDocumentTemplateWithHttpInfo(
    SaveDocumentTemplateRequest saveDocumentTemplateRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/document-templates';

    // ignore: prefer_final_locals
    Object? postBody = saveDocumentTemplateRequest;

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

  /// Create a document template
  ///
  /// Creates a template of the caller's clinic as version 1. Staff of the clinic. A translation must be complete: its title, every field's label, and an instruction exactly where the original has one. Any language of the enum but Arabic, whose script the PDF does not set yet. The answer carries the version as its ETag. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [SaveDocumentTemplateRequest] saveDocumentTemplateRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<DocumentTemplate?> createDocumentTemplate(
    SaveDocumentTemplateRequest saveDocumentTemplateRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await createDocumentTemplateWithHttpInfo(
      saveDocumentTemplateRequest,
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
        'DocumentTemplate',
      ) as DocumentTemplate;
    }
    return null;
  }

  /// A signed document's PDF
  ///
  /// The PDF of a signed document (the DocumentReference's attachment url), decrypted as it is sent, as a download. For the encounter's practitioner with the patient's active consent, and for the encounter's patient whatever the consent (their own record). Audited as a read of the DocumentReference, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The signed document's FHIR DocumentReference.id.
  Future<Response> downloadDocumentWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/document-references/{id}/content'.replaceAll('{id}', id);

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

  /// A signed document's PDF
  ///
  /// The PDF of a signed document (the DocumentReference's attachment url), decrypted as it is sent, as a download. For the encounter's practitioner with the patient's active consent, and for the encounter's patient whatever the consent (their own record). Audited as a read of the DocumentReference, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The signed document's FHIR DocumentReference.id.
  Future<MultipartFile?> downloadDocument(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await downloadDocumentWithHttpInfo(
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
        'MultipartFile',
      ) as MultipartFile;
    }
    return null;
  }

  /// Draft a document from the transcript
  ///
  /// Drafts (or redrafts) a template from the encounter's stored transcript so far, during or after the call, and answers with the draft once it is stored; it is also pushed on the encounter's event stream as a `DraftMessage`. The model fills the fields, each filled field citing the transcript segments it came from; a value without sources, citing a segment the transcript does not have, or not of its field's type is not kept, and the field stays missing. A redraft replaces the open draft of the template as its next version. Without a transcript every field is missing. For the encounter's practitioner with the patient's active consent and active transcription consent only (SAL-12; checked again once the model answered). Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The encounter's id (FHIR Encounter.id).
  ///
  /// * [GenerateDraftRequest] generateDraftRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> generateDraftWithHttpInfo(
    String id,
    GenerateDraftRequest generateDraftRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/encounters/{id}/drafts'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = generateDraftRequest;

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

  /// Draft a document from the transcript
  ///
  /// Drafts (or redrafts) a template from the encounter's stored transcript so far, during or after the call, and answers with the draft once it is stored; it is also pushed on the encounter's event stream as a `DraftMessage`. The model fills the fields, each filled field citing the transcript segments it came from; a value without sources, citing a segment the transcript does not have, or not of its field's type is not kept, and the field stays missing. A redraft replaces the open draft of the template as its next version. Without a transcript every field is missing. For the encounter's practitioner with the patient's active consent and active transcription consent only (SAL-12; checked again once the model answered). Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The encounter's id (FHIR Encounter.id).
  ///
  /// * [GenerateDraftRequest] generateDraftRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<DraftEvent?> generateDraft(
    String id,
    GenerateDraftRequest generateDraftRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await generateDraftWithHttpInfo(
      id,
      generateDraftRequest,
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
        'DraftEvent',
      ) as DraftEvent;
    }
    return null;
  }

  /// The explained terms of a document
  ///
  /// Where the terms with a plain-language explanation are in a document (SAL-13), per field and line, tagged in the language it was written in: an overlay for the tooltip, the document is never changed. Under the Composition's rule and audited as its read: the encounter's practitioner with the patient's active consent, and the encounter's patient for a signed document only.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The document's FHIR Composition.id.
  Future<Response> getDocumentTermsWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/compositions/{id}/terms'.replaceAll('{id}', id);

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

  /// The explained terms of a document
  ///
  /// Where the terms with a plain-language explanation are in a document (SAL-13), per field and line, tagged in the language it was written in: an overlay for the tooltip, the document is never changed. Under the Composition's rule and audited as its read: the encounter's practitioner with the patient's active consent, and the encounter's patient for a signed document only.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The document's FHIR Composition.id.
  Future<DocumentTerms?> getDocumentTerms(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getDocumentTermsWithHttpInfo(
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
        'DocumentTerms',
      ) as DocumentTerms;
    }
    return null;
  }

  /// The clinic's document templates
  ///
  /// The caller's clinic's templates (a referral, a findings report), by title, with their translations. Staff of the clinic, practitioners and admins alike. Not audited: a template holds no personal data.
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> listDocumentTemplatesWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/document-templates';

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

  /// The clinic's document templates
  ///
  /// The caller's clinic's templates (a referral, a findings report), by title, with their translations. Staff of the clinic, practitioners and admins alike. Not audited: a template holds no personal data.
  Future<DocumentTemplates?> listDocumentTemplates({
    Future<void>? abortTrigger,
  }) async {
    final response = await listDocumentTemplatesWithHttpInfo(
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
        'DocumentTemplates',
      ) as DocumentTemplates;
    }
    return null;
  }

  /// The encounter's drafts
  ///
  /// Every draft of the encounter, oldest first, preliminary or final, for the encounter's practitioner with the patient's active consent; what a page reads when the event stream has expired. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The encounter's id (FHIR Encounter.id).
  Future<Response> listDraftsWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/encounters/{id}/drafts'.replaceAll('{id}', id);

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

  /// The encounter's drafts
  ///
  /// Every draft of the encounter, oldest first, preliminary or final, for the encounter's practitioner with the patient's active consent; what a page reads when the event stream has expired. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The encounter's id (FHIR Encounter.id).
  Future<Drafts?> listDrafts(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await listDraftsWithHttpInfo(
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
        'Drafts',
      ) as Drafts;
    }
    return null;
  }

  /// Sign a draft
  ///
  /// Signs the draft, as it was read (If-Match), with the value the practitioner left in every field: a value as drafted keeps its sources, an edited or typed one has none, an empty one is missing. The draft becomes the final FHIR Composition, attested by the encounter's practitioner; its PDF is stored encrypted as a DocumentReference, and a Provenance records the signature. The final draft is pushed on the event stream. For the encounter's practitioner with the patient's active consent only. Every filled coded field (an ICD-10 diagnosis) needs the practitioner's confirmation (`confirmed`) and becomes a confirmed FHIR Condition of the encounter's patient, written and audited with the signature (SAL-19, docs/plans/analytics.md §3). Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///   The draft's id (FHIR Composition.id).
  ///
  /// * [SignDraftRequest] signDraftRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> signDraftWithHttpInfo(
    String ifMatch,
    String id,
    SignDraftRequest signDraftRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/drafts/{id}/sign'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = signDraftRequest;

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
      'POST',
      queryParams,
      postBody,
      headerParams,
      formParams,
      contentTypes.isEmpty ? null : contentTypes.first,
      abortTrigger: abortTrigger,
    );
  }

  /// Sign a draft
  ///
  /// Signs the draft, as it was read (If-Match), with the value the practitioner left in every field: a value as drafted keeps its sources, an edited or typed one has none, an empty one is missing. The draft becomes the final FHIR Composition, attested by the encounter's practitioner; its PDF is stored encrypted as a DocumentReference, and a Provenance records the signature. The final draft is pushed on the event stream. For the encounter's practitioner with the patient's active consent only. Every filled coded field (an ICD-10 diagnosis) needs the practitioner's confirmation (`confirmed`) and becomes a confirmed FHIR Condition of the encounter's patient, written and audited with the signature (SAL-19, docs/plans/analytics.md §3). Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///   The draft's id (FHIR Composition.id).
  ///
  /// * [SignDraftRequest] signDraftRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<DraftEvent?> signDraft(
    String ifMatch,
    String id,
    SignDraftRequest signDraftRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await signDraftWithHttpInfo(
      ifMatch,
      id,
      signDraftRequest,
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
        'DraftEvent',
      ) as DraftEvent;
    }
    return null;
  }

  /// Edit a document template
  ///
  /// Replaces the template's content and translations as its next version. Send If-Match with the version read. Drafts keep the texts they were drafted with. Staff of the clinic. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///
  /// * [SaveDocumentTemplateRequest] saveDocumentTemplateRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> updateDocumentTemplateWithHttpInfo(
    String ifMatch,
    String id,
    SaveDocumentTemplateRequest saveDocumentTemplateRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/document-templates/{id}'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = saveDocumentTemplateRequest;

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
      'PUT',
      queryParams,
      postBody,
      headerParams,
      formParams,
      contentTypes.isEmpty ? null : contentTypes.first,
      abortTrigger: abortTrigger,
    );
  }

  /// Edit a document template
  ///
  /// Replaces the template's content and translations as its next version. Send If-Match with the version read. Drafts keep the texts they were drafted with. Staff of the clinic. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///
  /// * [SaveDocumentTemplateRequest] saveDocumentTemplateRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<DocumentTemplate?> updateDocumentTemplate(
    String ifMatch,
    String id,
    SaveDocumentTemplateRequest saveDocumentTemplateRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await updateDocumentTemplateWithHttpInfo(
      ifMatch,
      id,
      saveDocumentTemplateRequest,
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
        'DocumentTemplate',
      ) as DocumentTemplate;
    }
    return null;
  }
}
