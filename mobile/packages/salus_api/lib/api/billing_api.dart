//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class BillingApi {
  BillingApi([ApiClient? apiClient])
      : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// Add a service to the clinic's price list
  ///
  /// By the clinic's admin. Prices are shown to patients at booking and paid at the clinic, never charged in the app. Built in package B. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [SaveServiceRequest] saveServiceRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> createClinicServiceWithHttpInfo(
    SaveServiceRequest saveServiceRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/services';

    // ignore: prefer_final_locals
    Object? postBody = saveServiceRequest;

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

  /// Add a service to the clinic's price list
  ///
  /// By the clinic's admin. Prices are shown to patients at booking and paid at the clinic, never charged in the app. Built in package B. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [SaveServiceRequest] saveServiceRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Service?> createClinicService(
    SaveServiceRequest saveServiceRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await createClinicServiceWithHttpInfo(
      saveServiceRequest,
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
        'Service',
      ) as Service;
    }
    return null;
  }

  /// The clinic's billing details
  ///
  /// Its legal name, tax id (JIB or IDB), VAT registration and the e-mail Salus's invoices go to. For the clinic's admin. Built in package D. Not personal data.
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> getBillingDetailsWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/billing-details';

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

  /// The clinic's billing details
  ///
  /// Its legal name, tax id (JIB or IDB), VAT registration and the e-mail Salus's invoices go to. For the clinic's admin. Built in package D. Not personal data.
  Future<BillingDetails?> getBillingDetails({
    Future<void>? abortTrigger,
  }) async {
    final response = await getBillingDetailsWithHttpInfo(
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
        'BillingDetails',
      ) as BillingDetails;
    }
    return null;
  }

  /// A receipt of the clinic
  ///
  /// For the clinic's admin and the practitioner who issued it. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Response> getClinicReceiptWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/receipts/{id}'.replaceAll('{id}', id);

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

  /// A receipt of the clinic
  ///
  /// For the clinic's admin and the practitioner who issued it. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Receipt?> getClinicReceipt(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getClinicReceiptWithHttpInfo(
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
        'Receipt',
      ) as Receipt;
    }
    return null;
  }

  /// A receipt as a PDF on the clinic's letterhead
  ///
  /// The clinic's name, legal name, tax id and address, the lines, totals, the fiscal code and its QR. For the staff who may see the receipt; a patient downloads their own at /me/receipts/{id}/pdf. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Response> getClinicReceiptPdfWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/receipts/{id}/pdf'.replaceAll('{id}', id);

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

  /// A receipt as a PDF on the clinic's letterhead
  ///
  /// The clinic's name, legal name, tax id and address, the lines, totals, the fiscal code and its QR. For the staff who may see the receipt; a patient downloads their own at /me/receipts/{id}/pdf. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<String?> getClinicReceiptPdf(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getClinicReceiptPdfWithHttpInfo(
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
        'String',
      ) as String;
    }
    return null;
  }

  /// The clinic's membership
  ///
  /// Its plan, status and current period; the admin's banner reads `status` and `pastDueSince`. For the clinic's admin. Built in package D of docs/plans/clinic-cms.md. Not personal data.
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> getClinicSubscriptionWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/subscription';

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

  /// The clinic's membership
  ///
  /// Its plan, status and current period; the admin's banner reads `status` and `pastDueSince`. For the clinic's admin. Built in package D of docs/plans/clinic-cms.md. Not personal data.
  Future<Subscription?> getClinicSubscription({
    Future<void>? abortTrigger,
  }) async {
    final response = await getClinicSubscriptionWithHttpInfo(
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
        'Subscription',
      ) as Subscription;
    }
    return null;
  }

  /// The patient's own receipt as a PDF
  ///
  /// The same document as the clinic's. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Response> getMyReceiptPdfWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/me/receipts/{id}/pdf'.replaceAll('{id}', id);

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

  /// The patient's own receipt as a PDF
  ///
  /// The same document as the clinic's. Audited.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<String?> getMyReceiptPdf(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getMyReceiptPdfWithHttpInfo(
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
        'String',
      ) as String;
    }
    return null;
  }

  /// The clinic's receipts summed for a day or a period, for its accountant
  ///
  /// Totals by payment method and by service of the receipts issued in [from, to), reversals included; as JSON, or as CSV with `format=csv`. For the clinic's admin. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [DateTime] from (required):
  ///
  /// * [DateTime] to (required):
  ///
  /// * [String] format:
  Future<Response> getReceiptsReportWithHttpInfo(
    DateTime from,
    DateTime to, {
    String? format,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/reports/receipts';

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    queryParams.addAll(_queryParams('', 'from', from));
    queryParams.addAll(_queryParams('', 'to', to));
    if (format != null) {
      queryParams.addAll(_queryParams('', 'format', format));
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

  /// The clinic's receipts summed for a day or a period, for its accountant
  ///
  /// Totals by payment method and by service of the receipts issued in [from, to), reversals included; as JSON, or as CSV with `format=csv`. For the clinic's admin. Audited.
  ///
  /// Parameters:
  ///
  /// * [DateTime] from (required):
  ///
  /// * [DateTime] to (required):
  ///
  /// * [String] format:
  Future<ReceiptsReport?> getReceiptsReport(
    DateTime from,
    DateTime to, {
    String? format,
    Future<void>? abortTrigger,
  }) async {
    final response = await getReceiptsReportWithHttpInfo(
      from,
      to,
      format: format,
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
        'ReceiptsReport',
      ) as ReceiptsReport;
    }
    return null;
  }

  /// A membership invoice as a PDF
  ///
  /// Salus's invoice to the clinic: the seller, the clinic as the buyer, the month, the seats, the amount and how to pay, in the clinic's default language. For the clinic's admin; also e-mailed to its billing e-mail when issued. Not personal data.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Response> getSubscriptionInvoicePdfWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path =
        r'/clinic/subscription/invoices/{id}/pdf'.replaceAll('{id}', id);

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

  /// A membership invoice as a PDF
  ///
  /// Salus's invoice to the clinic: the seller, the clinic as the buyer, the month, the seats, the amount and how to pay, in the clinic's default language. For the clinic's admin; also e-mailed to its billing e-mail when issued. Not personal data.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<String?> getSubscriptionInvoicePdf(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await getSubscriptionInvoicePdfWithHttpInfo(
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
        'String',
      ) as String;
    }
    return null;
  }

  /// Issue a receipt for a visit or on its own
  ///
  /// By the clinic's admin or the practitioner of the visit (the reception role waits for Q1). The lines come from the price list at issue; the number is the clinic's next one; the receipt is reported to the tax authority by a job after it is stored (`fiscalStatus` `pending` until then). An issued receipt never changes; a mistake is a reversal. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [IssueReceiptRequest] issueReceiptRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> issueReceiptWithHttpInfo(
    IssueReceiptRequest issueReceiptRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/receipts';

    // ignore: prefer_final_locals
    Object? postBody = issueReceiptRequest;

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

  /// Issue a receipt for a visit or on its own
  ///
  /// By the clinic's admin or the practitioner of the visit (the reception role waits for Q1). The lines come from the price list at issue; the number is the clinic's next one; the receipt is reported to the tax authority by a job after it is stored (`fiscalStatus` `pending` until then). An issued receipt never changes; a mistake is a reversal. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [IssueReceiptRequest] issueReceiptRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Receipt?> issueReceipt(
    IssueReceiptRequest issueReceiptRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await issueReceiptWithHttpInfo(
      issueReceiptRequest,
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
        'Receipt',
      ) as Receipt;
    }
    return null;
  }

  /// The clinic's receipts issued in a period
  ///
  /// Newest first, with their lines and fiscal status. For the clinic's admin. Package C of docs/plans/clinic-cms.md. Health data (a patient's receipts): audited.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [DateTime] from (required):
  ///
  /// * [DateTime] to (required):
  Future<Response> listClinicReceiptsWithHttpInfo(
    DateTime from,
    DateTime to, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/receipts';

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    queryParams.addAll(_queryParams('', 'from', from));
    queryParams.addAll(_queryParams('', 'to', to));

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

  /// The clinic's receipts issued in a period
  ///
  /// Newest first, with their lines and fiscal status. For the clinic's admin. Package C of docs/plans/clinic-cms.md. Health data (a patient's receipts): audited.
  ///
  /// Parameters:
  ///
  /// * [DateTime] from (required):
  ///
  /// * [DateTime] to (required):
  Future<List<Receipt>?> listClinicReceipts(
    DateTime from,
    DateTime to, {
    Future<void>? abortTrigger,
  }) async {
    final response = await listClinicReceiptsWithHttpInfo(
      from,
      to,
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
      final responseBody = await _decodeBodyBytes(response);
      return (await apiClient.deserializeAsync(responseBody, 'List<Receipt>')
              as List)
          .cast<Receipt>()
          .toList(growable: false);
    }
    return null;
  }

  /// The clinic's price list
  ///
  /// Every service of the caller's clinic, active or not, by name, with its translations. For the clinic's staff. Built in package B of docs/plans/clinic-cms.md. Not personal data; not audited.
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> listClinicServicesWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/services';

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

  /// The clinic's price list
  ///
  /// Every service of the caller's clinic, active or not, by name, with its translations. For the clinic's staff. Built in package B of docs/plans/clinic-cms.md. Not personal data; not audited.
  Future<List<Service>?> listClinicServices({
    Future<void>? abortTrigger,
  }) async {
    final response = await listClinicServicesWithHttpInfo(
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
      final responseBody = await _decodeBodyBytes(response);
      return (await apiClient.deserializeAsync(responseBody, 'List<Service>')
              as List)
          .cast<Service>()
          .toList(growable: false);
    }
    return null;
  }

  /// The patient's own receipts, from every clinic
  ///
  /// Newest first. Audited.
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> listMyReceiptsWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/me/receipts';

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

  /// The patient's own receipts, from every clinic
  ///
  /// Newest first. Audited.
  Future<List<Receipt>?> listMyReceipts({
    Future<void>? abortTrigger,
  }) async {
    final response = await listMyReceiptsWithHttpInfo(
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
      final responseBody = await _decodeBodyBytes(response);
      return (await apiClient.deserializeAsync(responseBody, 'List<Receipt>')
              as List)
          .cast<Receipt>()
          .toList(growable: false);
    }
    return null;
  }

  /// The services a listed clinic offers, with their prices, for booking
  ///
  /// The active services of a listed clinic offered in `mode`, by name, each in `language` where the clinic translated it and in the clinic's language otherwise. Public, as the directory is; anyone else's clinic gets 404. Built in package B.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The clinic's id.
  ///
  /// * [VisitMode] mode (required):
  ///
  /// * [String] language:
  ///   A BCP 47 tag of the language enum.
  Future<Response> listOfferedServicesWithHttpInfo(
    String id,
    VisitMode mode, {
    String? language,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/organizations/{id}/services'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody;

    final queryParams = <QueryParam>[];
    final headerParams = <String, String>{};
    final formParams = <String, String>{};

    queryParams.addAll(_queryParams('', 'mode', mode));
    if (language != null) {
      queryParams.addAll(_queryParams('', 'language', language));
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

  /// The services a listed clinic offers, with their prices, for booking
  ///
  /// The active services of a listed clinic offered in `mode`, by name, each in `language` where the clinic translated it and in the clinic's language otherwise. Public, as the directory is; anyone else's clinic gets 404. Built in package B.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///   The clinic's id.
  ///
  /// * [VisitMode] mode (required):
  ///
  /// * [String] language:
  ///   A BCP 47 tag of the language enum.
  Future<List<OfferedService>?> listOfferedServices(
    String id,
    VisitMode mode, {
    String? language,
    Future<void>? abortTrigger,
  }) async {
    final response = await listOfferedServicesWithHttpInfo(
      id,
      mode,
      language: language,
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
      final responseBody = await _decodeBodyBytes(response);
      return (await apiClient.deserializeAsync(
              responseBody, 'List<OfferedService>') as List)
          .cast<OfferedService>()
          .toList(growable: false);
    }
    return null;
  }

  /// Salus's monthly invoices to the clinic
  ///
  /// Newest first. For the clinic's admin. Built in package D.
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> listSubscriptionInvoicesWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/subscription/invoices';

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

  /// Salus's monthly invoices to the clinic
  ///
  /// Newest first. For the clinic's admin. Built in package D.
  Future<List<SubscriptionInvoice>?> listSubscriptionInvoices({
    Future<void>? abortTrigger,
  }) async {
    final response = await listSubscriptionInvoicesWithHttpInfo(
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
      final responseBody = await _decodeBodyBytes(response);
      return (await apiClient.deserializeAsync(
              responseBody, 'List<SubscriptionInvoice>') as List)
          .cast<SubscriptionInvoice>()
          .toList(growable: false);
    }
    return null;
  }

  /// Reverse an issued receipt
  ///
  /// Issues the receipt that reverses it (its lines negated), once; it is reported like any receipt. By the clinic's admin. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [ReverseReceiptRequest] reverseReceiptRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> reverseReceiptWithHttpInfo(
    String id,
    ReverseReceiptRequest reverseReceiptRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/receipts/{id}/reverse'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = reverseReceiptRequest;

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

  /// Reverse an issued receipt
  ///
  /// Issues the receipt that reverses it (its lines negated), once; it is reported like any receipt. By the clinic's admin. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [ReverseReceiptRequest] reverseReceiptRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Receipt?> reverseReceipt(
    String id,
    ReverseReceiptRequest reverseReceiptRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await reverseReceiptWithHttpInfo(
      id,
      reverseReceiptRequest,
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
        'Receipt',
      ) as Receipt;
    }
    return null;
  }

  /// Replace the clinic's billing details
  ///
  /// By the clinic's admin. Built in package D. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [BillingDetails] billingDetails (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> setBillingDetailsWithHttpInfo(
    BillingDetails billingDetails, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/billing-details';

    // ignore: prefer_final_locals
    Object? postBody = billingDetails;

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

  /// Replace the clinic's billing details
  ///
  /// By the clinic's admin. Built in package D. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [BillingDetails] billingDetails (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<BillingDetails?> setBillingDetails(
    BillingDetails billingDetails, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await setBillingDetailsWithHttpInfo(
      billingDetails,
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
        'BillingDetails',
      ) as BillingDetails;
    }
    return null;
  }

  /// Change a service of the clinic's price list
  ///
  /// By the clinic's admin, at the version of If-Match. A change never changes an issued receipt, whose lines are copied at issue. A service is never deleted: `active` false takes it off the list. Built in package B. Audited, also when refused.
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
  /// * [SaveServiceRequest] saveServiceRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> updateClinicServiceWithHttpInfo(
    String ifMatch,
    String id,
    SaveServiceRequest saveServiceRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/services/{id}'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = saveServiceRequest;

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

  /// Change a service of the clinic's price list
  ///
  /// By the clinic's admin, at the version of If-Match. A change never changes an issued receipt, whose lines are copied at issue. A service is never deleted: `active` false takes it off the list. Built in package B. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///
  /// * [SaveServiceRequest] saveServiceRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Service?> updateClinicService(
    String ifMatch,
    String id,
    SaveServiceRequest saveServiceRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await updateClinicServiceWithHttpInfo(
      ifMatch,
      id,
      saveServiceRequest,
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
        'Service',
      ) as Service;
    }
    return null;
  }
}
