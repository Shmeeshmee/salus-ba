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

  /// The clinic's appointments over time
  ///
  /// The caller's clinic's appointments (docs/plans/analytics.md §4, §8), from the pre-computed counts (refreshed within seconds of a change, and nightly): per month of the clinic's local days from `from` to `to`, `booked` counting every appointment and the others those with that status now; the heatmap of the ones not cancelled by ISO weekday (1 is Monday) and local hour, busy cells only; and, for the clinic's admin only, the fulfilled ones per practitioner role (absent for anyone else). A practitioner who is not an admin may narrow to their own role only, never to a specialty. Every read is audited on the clinic.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [DateTime] from (required):
  ///   The first local day of the clinic (YYYY-MM-DD).
  ///
  /// * [DateTime] to (required):
  ///   The last local day, included; at most ten years after `from`.
  ///
  /// * [String] practitionerRole:
  ///   Only this practitioner role's figures.
  ///
  /// * [String] specialty:
  ///   Only the roles of this SNOMED CT specialty code.
  Future<Response> getClinicAppointmentStatsWithHttpInfo(
    DateTime from,
    DateTime to, {
    String? practitionerRole,
    String? specialty,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/analytics/clinic/appointments';

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    queryParams.addAll(_queryParams('', 'from', from));
    queryParams.addAll(_queryParams('', 'to', to));
    if (practitionerRole != null) {
      queryParams
          .addAll(_queryParams('', 'practitionerRole', practitionerRole));
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

  /// The clinic's appointments over time
  ///
  /// The caller's clinic's appointments (docs/plans/analytics.md §4, §8), from the pre-computed counts (refreshed within seconds of a change, and nightly): per month of the clinic's local days from `from` to `to`, `booked` counting every appointment and the others those with that status now; the heatmap of the ones not cancelled by ISO weekday (1 is Monday) and local hour, busy cells only; and, for the clinic's admin only, the fulfilled ones per practitioner role (absent for anyone else). A practitioner who is not an admin may narrow to their own role only, never to a specialty. Every read is audited on the clinic.
  ///
  /// Parameters:
  ///
  /// * [DateTime] from (required):
  ///   The first local day of the clinic (YYYY-MM-DD).
  ///
  /// * [DateTime] to (required):
  ///   The last local day, included; at most ten years after `from`.
  ///
  /// * [String] practitionerRole:
  ///   Only this practitioner role's figures.
  ///
  /// * [String] specialty:
  ///   Only the roles of this SNOMED CT specialty code.
  Future<ClinicAppointments?> getClinicAppointmentStats(
    DateTime from,
    DateTime to, {
    String? practitionerRole,
    String? specialty,
    Future<void>? abortTrigger,
  }) async {
    final response = await getClinicAppointmentStatsWithHttpInfo(
      from,
      to,
      practitionerRole: practitionerRole,
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
        'ClinicAppointments',
      ) as ClinicAppointments;
    }
    return null;
  }

  /// The clinic's diagnoses, counted by a breakdown
  ///
  /// The caller's clinic's confirmed diagnoses (the coded fields of signed documents, docs/plans/analytics.md §4, §8) recorded on the clinic's local days from `from` to `to`, counted by `by`, most first (months in their order); a key is a diagnosis group or subgroup code, an ICD-10 code, a practitioner role's id, a SNOMED CT specialty code or a month (YYYY-MM), empty for no subgroup or no specialty. Exact counts. An admin gets the clinic's; a practitioner who is not an admin their own role's only, whatever they ask for. Every read is audited on the clinic.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [DateTime] from (required):
  ///   The first local day of the clinic (YYYY-MM-DD).
  ///
  /// * [DateTime] to (required):
  ///   The last local day, included; at most ten years after `from`.
  ///
  /// * [String] by (required):
  ///
  /// * [String] group:
  ///   Only this diagnosis group.
  ///
  /// * [String] subgroup:
  ///   Only this diagnosis subgroup.
  ///
  /// * [String] practitionerRole:
  ///   Only this practitioner role's figures.
  ///
  /// * [String] specialty:
  ///   Only the roles of this SNOMED CT specialty code.
  Future<Response> getClinicDiagnosisStatsWithHttpInfo(
    DateTime from,
    DateTime to,
    String by, {
    String? group,
    String? subgroup,
    String? practitionerRole,
    String? specialty,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/analytics/clinic/diagnoses';

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    queryParams.addAll(_queryParams('', 'from', from));
    queryParams.addAll(_queryParams('', 'to', to));
    queryParams.addAll(_queryParams('', 'by', by));
    if (group != null) {
      queryParams.addAll(_queryParams('', 'group', group));
    }
    if (subgroup != null) {
      queryParams.addAll(_queryParams('', 'subgroup', subgroup));
    }
    if (practitionerRole != null) {
      queryParams
          .addAll(_queryParams('', 'practitionerRole', practitionerRole));
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

  /// The clinic's diagnoses, counted by a breakdown
  ///
  /// The caller's clinic's confirmed diagnoses (the coded fields of signed documents, docs/plans/analytics.md §4, §8) recorded on the clinic's local days from `from` to `to`, counted by `by`, most first (months in their order); a key is a diagnosis group or subgroup code, an ICD-10 code, a practitioner role's id, a SNOMED CT specialty code or a month (YYYY-MM), empty for no subgroup or no specialty. Exact counts. An admin gets the clinic's; a practitioner who is not an admin their own role's only, whatever they ask for. Every read is audited on the clinic.
  ///
  /// Parameters:
  ///
  /// * [DateTime] from (required):
  ///   The first local day of the clinic (YYYY-MM-DD).
  ///
  /// * [DateTime] to (required):
  ///   The last local day, included; at most ten years after `from`.
  ///
  /// * [String] by (required):
  ///
  /// * [String] group:
  ///   Only this diagnosis group.
  ///
  /// * [String] subgroup:
  ///   Only this diagnosis subgroup.
  ///
  /// * [String] practitionerRole:
  ///   Only this practitioner role's figures.
  ///
  /// * [String] specialty:
  ///   Only the roles of this SNOMED CT specialty code.
  Future<ClinicDiagnoses?> getClinicDiagnosisStats(
    DateTime from,
    DateTime to,
    String by, {
    String? group,
    String? subgroup,
    String? practitionerRole,
    String? specialty,
    Future<void>? abortTrigger,
  }) async {
    final response = await getClinicDiagnosisStatsWithHttpInfo(
      from,
      to,
      by,
      group: group,
      subgroup: subgroup,
      practitionerRole: practitionerRole,
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
        'ClinicDiagnoses',
      ) as ClinicDiagnoses;
    }
    return null;
  }
}
