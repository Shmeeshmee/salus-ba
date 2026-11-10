//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class TerminologyApi {
  TerminologyApi([ApiClient? apiClient])
      : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// Edit a term's labels in one language
  ///
  /// Replaces the term's preferred label and synonyms in the language. If-Match is the translation's version read, `W/\"0\"` when the term has no label in the language yet. A change sets the translation to `unreviewed`, an approved one included, and keeps its last review; the same labels again change nothing. The term's own status follows its translations (docs/terminology/SPEC.md §3 \"Review\"). Reviewers only. Audited.
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
  /// * [TermLanguage] language (required):
  ///
  /// * [TermLabelsEdit] termLabelsEdit (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> editTermTranslationWithHttpInfo(
    String ifMatch,
    String id,
    TermLanguage language,
    TermLabelsEdit termLabelsEdit, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/terminology/terms/{id}/translations/{language}'
        .replaceAll('{id}', id)
        .replaceAll('{language}', language.toString());

    // ignore: prefer_final_locals
    Object? postBody = termLabelsEdit;

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

  /// Edit a term's labels in one language
  ///
  /// Replaces the term's preferred label and synonyms in the language. If-Match is the translation's version read, `W/\"0\"` when the term has no label in the language yet. A change sets the translation to `unreviewed`, an approved one included, and keeps its last review; the same labels again change nothing. The term's own status follows its translations (docs/terminology/SPEC.md §3 \"Review\"). Reviewers only. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///
  /// * [TermLanguage] language (required):
  ///
  /// * [TermLabelsEdit] termLabelsEdit (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<TermTranslation?> editTermTranslation(
    String ifMatch,
    String id,
    TermLanguage language,
    TermLabelsEdit termLabelsEdit, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await editTermTranslationWithHttpInfo(
      ifMatch,
      id,
      language,
      termLabelsEdit,
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
        'TermTranslation',
      ) as TermTranslation;
    }
    return null;
  }

  /// How far the review is in each language
  ///
  /// Per term language, how many terms there are and how many of their entries in that language are approved or need changes (statuses as in listTermReview). Staff only. Not audited.
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> getTermReviewProgressWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/terminology/review/progress';

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

  /// How far the review is in each language
  ///
  /// Per term language, how many terms there are and how many of their entries in that language are approved or need changes (statuses as in listTermReview). Staff only. Not audited.
  Future<TermReviewProgress?> getTermReviewProgress({
    Future<void>? abortTrigger,
  }) async {
    final response = await getTermReviewProgressWithHttpInfo(
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
        'TermReviewProgress',
      ) as TermReviewProgress;
    }
    return null;
  }

  /// The review queue of the medical terms in one language
  ///
  /// The review panel's queue (docs/terminology/SPEC.md §3 \"Review\"): one entry per term in the language, with the term's labels in that language, Latin and English, the review of those labels (the translation) and the term's explanation in that language. An entry's status is `needs_changes` when its translation or its explanation needs changes, else `unreviewed` when the term has no label in the language yet or either is unreviewed, else `approved` (an explanation that does not exist yet does not count). Unreviewed entries first, then those that need changes, then the approved ones, each the more common term first. Every staff member reads; only a reviewer writes. Not audited: reference data, no patient data.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [TermLanguage] language (required):
  ///
  /// * [ReviewStatus] status:
  ///   Only entries with this status; any when absent.
  ///
  /// * [String] after:
  ///   The `next` of the page before; the first page when absent.
  ///
  /// * [int] limit:
  Future<Response> listTermReviewWithHttpInfo(
    TermLanguage language, {
    ReviewStatus? status,
    String? after,
    int? limit,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/terminology/review';

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    queryParams.addAll(_queryParams('', 'language', language));
    if (status != null) {
      queryParams.addAll(_queryParams('', 'status', status));
    }
    if (after != null) {
      queryParams.addAll(_queryParams('', 'after', after));
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

  /// The review queue of the medical terms in one language
  ///
  /// The review panel's queue (docs/terminology/SPEC.md §3 \"Review\"): one entry per term in the language, with the term's labels in that language, Latin and English, the review of those labels (the translation) and the term's explanation in that language. An entry's status is `needs_changes` when its translation or its explanation needs changes, else `unreviewed` when the term has no label in the language yet or either is unreviewed, else `approved` (an explanation that does not exist yet does not count). Unreviewed entries first, then those that need changes, then the approved ones, each the more common term first. Every staff member reads; only a reviewer writes. Not audited: reference data, no patient data.
  ///
  /// Parameters:
  ///
  /// * [TermLanguage] language (required):
  ///
  /// * [ReviewStatus] status:
  ///   Only entries with this status; any when absent.
  ///
  /// * [String] after:
  ///   The `next` of the page before; the first page when absent.
  ///
  /// * [int] limit:
  Future<TermReviewPage?> listTermReview(
    TermLanguage language, {
    ReviewStatus? status,
    String? after,
    int? limit,
    Future<void>? abortTrigger,
  }) async {
    final response = await listTermReviewWithHttpInfo(
      language,
      status: status,
      after: after,
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
        'TermReviewPage',
      ) as TermReviewPage;
    }
    return null;
  }

  /// Approve a term's explanation, or mark it as needing changes
  ///
  /// As reviewTermTranslation, for the explanation in the language. Reviewers only. Audited.
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
  /// * [Language] language (required):
  ///
  /// * [ReviewDecision] reviewDecision (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> reviewTermExplanationWithHttpInfo(
    String ifMatch,
    String id,
    Language language,
    ReviewDecision reviewDecision, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/terminology/terms/{id}/explanations/{language}/review'
        .replaceAll('{id}', id)
        .replaceAll('{language}', language.toString());

    // ignore: prefer_final_locals
    Object? postBody = reviewDecision;

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

  /// Approve a term's explanation, or mark it as needing changes
  ///
  /// As reviewTermTranslation, for the explanation in the language. Reviewers only. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///
  /// * [Language] language (required):
  ///
  /// * [ReviewDecision] reviewDecision (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<TermExplanation?> reviewTermExplanation(
    String ifMatch,
    String id,
    Language language,
    ReviewDecision reviewDecision, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await reviewTermExplanationWithHttpInfo(
      ifMatch,
      id,
      language,
      reviewDecision,
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
        'TermExplanation',
      ) as TermExplanation;
    }
    return null;
  }

  /// Approve a term's labels in one language, or mark them as needing changes
  ///
  /// Sets the translation's status with the reviewer as `reviewedBy`, now as `reviewedAt` and the note (needed for `needs_changes`, optional for `approved`, cleared when absent). If-Match is the version read. Reviewers only. Audited.
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
  /// * [TermLanguage] language (required):
  ///
  /// * [ReviewDecision] reviewDecision (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> reviewTermTranslationWithHttpInfo(
    String ifMatch,
    String id,
    TermLanguage language,
    ReviewDecision reviewDecision, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/terminology/terms/{id}/translations/{language}/review'
        .replaceAll('{id}', id)
        .replaceAll('{language}', language.toString());

    // ignore: prefer_final_locals
    Object? postBody = reviewDecision;

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

  /// Approve a term's labels in one language, or mark them as needing changes
  ///
  /// Sets the translation's status with the reviewer as `reviewedBy`, now as `reviewedAt` and the note (needed for `needs_changes`, optional for `approved`, cleared when absent). If-Match is the version read. Reviewers only. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///
  /// * [TermLanguage] language (required):
  ///
  /// * [ReviewDecision] reviewDecision (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<TermTranslation?> reviewTermTranslation(
    String ifMatch,
    String id,
    TermLanguage language,
    ReviewDecision reviewDecision, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await reviewTermTranslationWithHttpInfo(
      ifMatch,
      id,
      language,
      reviewDecision,
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
        'TermTranslation',
      ) as TermTranslation;
    }
    return null;
  }

  /// Search the medical terminology
  ///
  /// The cheatsheet's search (docs/terminology/SPEC.md §3 \"API and UI\"). Matches q against every label of every term in every language, case and accents aside: an exact label first, then a label that starts with q, then a word within trigram distance (later words of a label, misspellings), then the more common term. Each match carries its labels in the requested languages. Staff only (practitioners and admins). Not audited: it reads no patient data. The search text is never logged or stored, and no error repeats it.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] q (required):
  ///   The search text, 1 to 100 characters.
  ///
  /// * [Set<TermLanguage>] languages:
  ///   The languages whose labels come back, in the order the client shows them. Default: la, the caller's UI language, en.
  ///
  /// * [TermCategory] category:
  ///   Only terms of this category; any when absent.
  ///
  /// * [int] limit:
  ///   How many terms at most.
  Future<Response> searchTerminologyWithHttpInfo(
    String q, {
    Set<TermLanguage>? languages,
    TermCategory? category,
    int? limit,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/terminology/search';

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    queryParams.addAll(_queryParams('', 'q', q));
    if (languages != null) {
      queryParams.addAll(_queryParams('csv', 'languages', languages));
    }
    if (category != null) {
      queryParams.addAll(_queryParams('', 'category', category));
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

  /// Search the medical terminology
  ///
  /// The cheatsheet's search (docs/terminology/SPEC.md §3 \"API and UI\"). Matches q against every label of every term in every language, case and accents aside: an exact label first, then a label that starts with q, then a word within trigram distance (later words of a label, misspellings), then the more common term. Each match carries its labels in the requested languages. Staff only (practitioners and admins). Not audited: it reads no patient data. The search text is never logged or stored, and no error repeats it.
  ///
  /// Parameters:
  ///
  /// * [String] q (required):
  ///   The search text, 1 to 100 characters.
  ///
  /// * [Set<TermLanguage>] languages:
  ///   The languages whose labels come back, in the order the client shows them. Default: la, the caller's UI language, en.
  ///
  /// * [TermCategory] category:
  ///   Only terms of this category; any when absent.
  ///
  /// * [int] limit:
  ///   How many terms at most.
  Future<TermList?> searchTerminology(
    String q, {
    Set<TermLanguage>? languages,
    TermCategory? category,
    int? limit,
    Future<void>? abortTrigger,
  }) async {
    final response = await searchTerminologyWithHttpInfo(
      q,
      languages: languages,
      category: category,
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
        'TermList',
      ) as TermList;
    }
    return null;
  }

  /// Write a term's explanation in one language
  ///
  /// Writes the plain-language explanation, trimmed, 1 to 500 characters. If-Match is the explanation's version read, `W/\"0\"` when there is none yet. A new or changed explanation is `unreviewed`, an approved one included, and keeps its last review; the same text again changes nothing. Reviewers only. Audited.
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
  /// * [Language] language (required):
  ///
  /// * [TermExplanationEdit] termExplanationEdit (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> writeTermExplanationWithHttpInfo(
    String ifMatch,
    String id,
    Language language,
    TermExplanationEdit termExplanationEdit, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/terminology/terms/{id}/explanations/{language}'
        .replaceAll('{id}', id)
        .replaceAll('{language}', language.toString());

    // ignore: prefer_final_locals
    Object? postBody = termExplanationEdit;

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

  /// Write a term's explanation in one language
  ///
  /// Writes the plain-language explanation, trimmed, 1 to 500 characters. If-Match is the explanation's version read, `W/\"0\"` when there is none yet. A new or changed explanation is `unreviewed`, an approved one included, and keeps its last review; the same text again changes nothing. Reviewers only. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///
  /// * [Language] language (required):
  ///
  /// * [TermExplanationEdit] termExplanationEdit (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<TermExplanation?> writeTermExplanation(
    String ifMatch,
    String id,
    Language language,
    TermExplanationEdit termExplanationEdit, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await writeTermExplanationWithHttpInfo(
      ifMatch,
      id,
      language,
      termExplanationEdit,
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
        'TermExplanation',
      ) as TermExplanation;
    }
    return null;
  }
}
