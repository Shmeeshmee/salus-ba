//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class LocationsApi {
  LocationsApi([ApiClient? apiClient])
      : apiClient = apiClient ?? defaultApiClient;

  final ApiClient apiClient;

  /// Add a location to the clinic
  ///
  /// By the clinic's admin. A clinic has at least one location: the first is its address of before wave 6, made by the migration. Built in package LC1. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [SaveLocationRequest] saveLocationRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> addClinicLocationWithHttpInfo(
    SaveLocationRequest saveLocationRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/locations';

    // ignore: prefer_final_locals
    Object? postBody = saveLocationRequest;

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

  /// Add a location to the clinic
  ///
  /// By the clinic's admin. A clinic has at least one location: the first is its address of before wave 6, made by the migration. Built in package LC1. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [SaveLocationRequest] saveLocationRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Location?> addClinicLocation(
    SaveLocationRequest saveLocationRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await addClinicLocationWithHttpInfo(
      saveLocationRequest,
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
        'Location',
      ) as Location;
    }
    return null;
  }

  /// Add a device to a location
  ///
  /// By the clinic's admin. A device a price-list service may require; two visits that need one device never overlap, the booking is refused with `resource_busy` (U4). Built in package RM1. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [SaveDeviceRequest] saveDeviceRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> addClinicLocationDeviceWithHttpInfo(
    String id,
    SaveDeviceRequest saveDeviceRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/locations/{id}/devices'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = saveDeviceRequest;

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

  /// Add a device to a location
  ///
  /// By the clinic's admin. A device a price-list service may require; two visits that need one device never overlap, the booking is refused with `resource_busy` (U4). Built in package RM1. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [SaveDeviceRequest] saveDeviceRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Device?> addClinicLocationDevice(
    String id,
    SaveDeviceRequest saveDeviceRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await addClinicLocationDeviceWithHttpInfo(
      id,
      saveDeviceRequest,
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
        'Device',
      ) as Device;
    }
    return null;
  }

  /// Add a room to a location
  ///
  /// By the clinic's admin. A room an in-person availability rule may name; two practitioners' in-person hours never overlap in one room (U3). Built in package RM1. Audited, also when refused.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [SaveRoomRequest] saveRoomRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> addClinicLocationRoomWithHttpInfo(
    String id,
    SaveRoomRequest saveRoomRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/locations/{id}/rooms'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = saveRoomRequest;

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

  /// Add a room to a location
  ///
  /// By the clinic's admin. A room an in-person availability rule may name; two practitioners' in-person hours never overlap in one room (U3). Built in package RM1. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  ///
  /// * [SaveRoomRequest] saveRoomRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Room?> addClinicLocationRoom(
    String id,
    SaveRoomRequest saveRoomRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await addClinicLocationRoomWithHttpInfo(
      id,
      saveRoomRequest,
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
        'Room',
      ) as Room;
    }
    return null;
  }

  /// A location's devices
  ///
  /// A device a price-list service may require; two visits that need one device never overlap, the booking is refused with `resource_busy` (U4). By name. For the clinic's staff. Not personal data. Built in package RM1.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Response> listClinicLocationDevicesWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/locations/{id}/devices'.replaceAll('{id}', id);

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

  /// A location's devices
  ///
  /// A device a price-list service may require; two visits that need one device never overlap, the booking is refused with `resource_busy` (U4). By name. For the clinic's staff. Not personal data. Built in package RM1.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<List<Device>?> listClinicLocationDevices(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await listClinicLocationDevicesWithHttpInfo(
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
      final responseBody = await _decodeBodyBytes(response);
      return (await apiClient.deserializeAsync(responseBody, 'List<Device>')
              as List)
          .cast<Device>()
          .toList(growable: false);
    }
    return null;
  }

  /// A location's rooms
  ///
  /// A room an in-person availability rule may name; two practitioners' in-person hours never overlap in one room (U3). By name. For the clinic's staff. Not personal data. Built in package RM1.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Response> listClinicLocationRoomsWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/locations/{id}/rooms'.replaceAll('{id}', id);

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

  /// A location's rooms
  ///
  /// A room an in-person availability rule may name; two practitioners' in-person hours never overlap in one room (U3). By name. For the clinic's staff. Not personal data. Built in package RM1.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<List<Room>?> listClinicLocationRooms(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await listClinicLocationRoomsWithHttpInfo(
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
      final responseBody = await _decodeBodyBytes(response);
      return (await apiClient.deserializeAsync(responseBody, 'List<Room>')
              as List)
          .cast<Room>()
          .toList(growable: false);
    }
    return null;
  }

  /// The clinic's locations
  ///
  /// Every site of the clinic, active first, by name (U1 of docs/clinic-cms/WAVE6.md). For the clinic's staff. Not personal data. Built in package LC1.
  ///
  /// Note: This method returns the HTTP [Response].
  Future<Response> listClinicLocationsWithHttpInfo({
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/locations';

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

  /// The clinic's locations
  ///
  /// Every site of the clinic, active first, by name (U1 of docs/clinic-cms/WAVE6.md). For the clinic's staff. Not personal data. Built in package LC1.
  Future<List<Location>?> listClinicLocations({
    Future<void>? abortTrigger,
  }) async {
    final response = await listClinicLocationsWithHttpInfo(
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
      return (await apiClient.deserializeAsync(responseBody, 'List<Location>')
              as List)
          .cast<Location>()
          .toList(growable: false);
    }
    return null;
  }

  /// A listed clinic's locations
  ///
  /// The active locations of a listed clinic, by name, for the patient booking in person: they pick the location before the time (U2). Public, as `/organizations/{id}/services`. Built in package LC1.
  ///
  /// Note: This method returns the HTTP [Response].
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<Response> listOrganizationLocationsWithHttpInfo(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/organizations/{id}/locations'.replaceAll('{id}', id);

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

  /// A listed clinic's locations
  ///
  /// The active locations of a listed clinic, by name, for the patient booking in person: they pick the location before the time (U2). Public, as `/organizations/{id}/services`. Built in package LC1.
  ///
  /// Parameters:
  ///
  /// * [String] id (required):
  Future<List<Location>?> listOrganizationLocations(
    String id, {
    Future<void>? abortTrigger,
  }) async {
    final response = await listOrganizationLocationsWithHttpInfo(
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
      final responseBody = await _decodeBodyBytes(response);
      return (await apiClient.deserializeAsync(responseBody, 'List<Location>')
              as List)
          .cast<Location>()
          .toList(growable: false);
    }
    return null;
  }

  /// Change a device
  ///
  /// By the clinic's admin, at the version of If-Match. Never deleted: `active` false takes it out of use; it never moves to another location. Built in package RM1. Audited, also when refused.
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
  /// * [SaveDeviceRequest] saveDeviceRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> updateClinicDeviceWithHttpInfo(
    String ifMatch,
    String id,
    SaveDeviceRequest saveDeviceRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/devices/{id}'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = saveDeviceRequest;

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

  /// Change a device
  ///
  /// By the clinic's admin, at the version of If-Match. Never deleted: `active` false takes it out of use; it never moves to another location. Built in package RM1. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///
  /// * [SaveDeviceRequest] saveDeviceRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Device?> updateClinicDevice(
    String ifMatch,
    String id,
    SaveDeviceRequest saveDeviceRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await updateClinicDeviceWithHttpInfo(
      ifMatch,
      id,
      saveDeviceRequest,
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
        'Device',
      ) as Device;
    }
    return null;
  }

  /// Change a location of the clinic
  ///
  /// By the clinic's admin, at the version of If-Match. Never deleted: `active` false takes it off the booking; past visits keep its address. Built in package LC1. Audited, also when refused.
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
  /// * [SaveLocationRequest] saveLocationRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> updateClinicLocationWithHttpInfo(
    String ifMatch,
    String id,
    SaveLocationRequest saveLocationRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/locations/{id}'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = saveLocationRequest;

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

  /// Change a location of the clinic
  ///
  /// By the clinic's admin, at the version of If-Match. Never deleted: `active` false takes it off the booking; past visits keep its address. Built in package LC1. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///
  /// * [SaveLocationRequest] saveLocationRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Location?> updateClinicLocation(
    String ifMatch,
    String id,
    SaveLocationRequest saveLocationRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await updateClinicLocationWithHttpInfo(
      ifMatch,
      id,
      saveLocationRequest,
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
        'Location',
      ) as Location;
    }
    return null;
  }

  /// Change a room
  ///
  /// By the clinic's admin, at the version of If-Match. Never deleted: `active` false takes it out of use; it never moves to another location. Built in package RM1. Audited, also when refused.
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
  /// * [SaveRoomRequest] saveRoomRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Response> updateClinicRoomWithHttpInfo(
    String ifMatch,
    String id,
    SaveRoomRequest saveRoomRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    // ignore: prefer_const_declarations
    final path = r'/clinic/rooms/{id}'.replaceAll('{id}', id);

    // ignore: prefer_final_locals
    Object? postBody = saveRoomRequest;

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

  /// Change a room
  ///
  /// By the clinic's admin, at the version of If-Match. Never deleted: `active` false takes it out of use; it never moves to another location. Built in package RM1. Audited, also when refused.
  ///
  /// Parameters:
  ///
  /// * [String] ifMatch (required):
  ///   The version read, as a weak ETag, `W/\"<versionId>\"`.
  ///
  /// * [String] id (required):
  ///
  /// * [SaveRoomRequest] saveRoomRequest (required):
  ///
  /// * [String] xSalusCSRF:
  ///   Sent as `1` by the web app on every write. Required with the session cookie on POST, PUT, PATCH and DELETE (403 `csrf` without it). On the login steps it asks for the session in the cookie instead of the body.
  Future<Room?> updateClinicRoom(
    String ifMatch,
    String id,
    SaveRoomRequest saveRoomRequest, {
    String? xSalusCSRF,
    Future<void>? abortTrigger,
  }) async {
    final response = await updateClinicRoomWithHttpInfo(
      ifMatch,
      id,
      saveRoomRequest,
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
        'Room',
      ) as Room;
    }
    return null;
  }
}
