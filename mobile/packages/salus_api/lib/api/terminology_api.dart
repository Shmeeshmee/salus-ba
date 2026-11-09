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
}
