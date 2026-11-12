//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class AnalyticsApi {
  AnalyticsApi([ApiClient? apiClient])
      : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// A breakdown of one All of Salus table
  ///
  /// One table of the published months (gd: geography × diagnosis, gs: geography × specialty, sd: specialty × diagnosis), some of its dimensions fixed, broken down by one of them or by month (§5, §8). Every key of the dimension is returned; a count is the sum of its published month cells, rounded to 5, and null when none is, for 0 and for suppressed alike (A8). `partial` is true when a count leaves out a month that is null for its key (A9: patient-months). A code is a key only in the months it is published at geography = all. Any other parameter, or a combination outside the table, is a 400. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] table (required):
  ///
  /// * [String] from (required):
  ///   The first month, 2026-01.
  ///
  /// * [String] to (required):
  ///   The last month; at most 60 months from the first.
  ///
  /// * [String] by (required):
  ///
  /// * [String] entity:
  ///
  /// * [String] region:
  ///
  /// * [String] city:
  ///   A city's Wikidata id.
  ///
  /// * [String] group:
  ///
  /// * [String] subgroup:
  ///
  /// * [String] code:
  ///   An ICD-10 code.
  ///
  /// * [String] specialty:
  ///   A SNOMED CT code.
  Future<Response> getPlatformCellsWithHttpInfo(
    String table,
    String from,
    String to,
    String by, {
    String? entity,
    String? region,
    String? city,
    String? group,
    String? subgroup,
    String? code,
    String? specialty,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/analytics/platform/cells';

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    queryParams.addAll(_queryParams('', 'table', table));
    queryParams.addAll(_queryParams('', 'from', from));
    queryParams.addAll(_queryParams('', 'to', to));
    queryParams.addAll(_queryParams('', 'by', by));
    if (entity != null) {
      queryParams.addAll(_queryParams('', 'entity', entity));
    }
    if (region != null) {
      queryParams.addAll(_queryParams('', 'region', region));
    }
    if (city != null) {
      queryParams.addAll(_queryParams('', 'city', city));
    }
    if (group != null) {
      queryParams.addAll(_queryParams('', 'group', group));
    }
    if (subgroup != null) {
      queryParams.addAll(_queryParams('', 'subgroup', subgroup));
    }
    if (code != null) {
      queryParams.addAll(_queryParams('', 'code', code));
    }
    if (specialty != null) {
      queryParams.addAll(_queryParams('', 'specialty', specialty));
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

  /// A breakdown of one All of Salus table
  ///
  /// One table of the published months (gd: geography × diagnosis, gs: geography × specialty, sd: specialty × diagnosis), some of its dimensions fixed, broken down by one of them or by month (§5, §8). Every key of the dimension is returned; a count is the sum of its published month cells, rounded to 5, and null when none is, for 0 and for suppressed alike (A8). `partial` is true when a count leaves out a month that is null for its key (A9: patient-months). A code is a key only in the months it is published at geography = all. Any other parameter, or a combination outside the table, is a 400. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] table (required):
  ///
  /// * [String] from (required):
  ///   The first month, 2026-01.
  ///
  /// * [String] to (required):
  ///   The last month; at most 60 months from the first.
  ///
  /// * [String] by (required):
  ///
  /// * [String] entity:
  ///
  /// * [String] region:
  ///
  /// * [String] city:
  ///   A city's Wikidata id.
  ///
  /// * [String] group:
  ///
  /// * [String] subgroup:
  ///
  /// * [String] code:
  ///   An ICD-10 code.
  ///
  /// * [String] specialty:
  ///   A SNOMED CT code.
  Future<PlatformBreakdown?> getPlatformCells(
    String table,
    String from,
    String to,
    String by, {
    String? entity,
    String? region,
    String? city,
    String? group,
    String? subgroup,
    String? code,
    String? specialty,
    Future<void>? abortTrigger,
  }) async {
    final response = await getPlatformCellsWithHttpInfo(
      table,
      from,
      to,
      by,
      entity: entity,
      region: region,
      city: city,
      group: group,
      subgroup: subgroup,
      code: code,
      specialty: specialty,
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
        'PlatformBreakdown',
      ) as PlatformBreakdown;
    }
    return null;
  }

  /// A published quarter's fulfilled appointments by weekday and hour
  ///
  /// The 7 × 24 cells (ISO weekday, 1 = Monday; the clinics' local hour) of a published quarter, of a region, an entity or all of Salus (§5): distinct patients rounded to 5, null for 0 and for suppressed alike. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] quarter (required):
  ///
  /// * [String] region:
  ///   A leaf region or an entity; all of Salus when left out.
  Future<Response> getPlatformHeatmapWithHttpInfo(
    String quarter, {
    String? region,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/analytics/platform/heatmap';

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    queryParams.addAll(_queryParams('', 'quarter', quarter));
    if (region != null) {
      queryParams.addAll(_queryParams('', 'region', region));
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

  /// A published quarter's fulfilled appointments by weekday and hour
  ///
  /// The 7 × 24 cells (ISO weekday, 1 = Monday; the clinics' local hour) of a published quarter, of a region, an entity or all of Salus (§5): distinct patients rounded to 5, null for 0 and for suppressed alike. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] quarter (required):
  ///
  /// * [String] region:
  ///   A leaf region or an entity; all of Salus when left out.
  Future<PlatformHeatmap?> getPlatformHeatmap(
    String quarter, {
    String? region,
    Future<void>? abortTrigger,
  }) async {
    final response = await getPlatformHeatmapWithHttpInfo(
      quarter,
      region: region,
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
        'PlatformHeatmap',
      ) as PlatformHeatmap;
    }
    return null;
  }

  /// The published months and quarters of the All of Salus figures
  ///
  /// The calendar months and quarters built and published (docs/plans/analytics.md §5), oldest first, and k, the minimum count. Only while ANALYTICS_PLATFORM_ENABLED is on (otherwise every /analytics/platform path is a 404, §2a), for staff of a clinic that contributes (O5, O6). Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> getPlatformMonthsWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/analytics/platform/months';

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

  /// The published months and quarters of the All of Salus figures
  ///
  /// The calendar months and quarters built and published (docs/plans/analytics.md §5), oldest first, and k, the minimum count. Only while ANALYTICS_PLATFORM_ENABLED is on (otherwise every /analytics/platform path is a 404, §2a), for staff of a clinic that contributes (O5, O6). Audited.
  Future<PlatformMonths?> getPlatformMonths({
    Future<void>? abortTrigger,
  }) async {
    final response = await getPlatformMonthsWithHttpInfo(
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
        'PlatformMonths',
      ) as PlatformMonths;
    }
    return null;
  }

  /// Unusual rises of a diagnosis group in a region
  ///
  /// The rise flags of the last built ISO weeks (§6): week, leaf region and diagnosis group, no count. A flag compares the week with the weeks before; it is not a public-health alert. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] region:
  ///   A leaf region; all when left out.
  ///
  /// * [int] weeks:
  Future<Response> getPlatformRisesWithHttpInfo({
    String? region,
    int? weeks,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/analytics/platform/rises';

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    if (region != null) {
      queryParams.addAll(_queryParams('', 'region', region));
    }
    if (weeks != null) {
      queryParams.addAll(_queryParams('', 'weeks', weeks));
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

  /// Unusual rises of a diagnosis group in a region
  ///
  /// The rise flags of the last built ISO weeks (§6): week, leaf region and diagnosis group, no count. A flag compares the week with the weeks before; it is not a public-health alert. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] region:
  ///   A leaf region; all when left out.
  ///
  /// * [int] weeks:
  Future<PlatformRises?> getPlatformRises({
    String? region,
    int? weeks,
    Future<void>? abortTrigger,
  }) async {
    final response = await getPlatformRisesWithHttpInfo(
      region: region,
      weeks: weeks,
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
        'PlatformRises',
      ) as PlatformRises;
    }
    return null;
  }
}
