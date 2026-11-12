//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of salus_api;

class ApiClient {
  ApiClient({
    this.basePath = 'http://localhost',
    this.authentication,
  });

  final String basePath;
  final Authentication? authentication;

  var _client = Client();
  final _defaultHeaderMap = <String, String>{};

  /// Returns the current HTTP [Client] instance to use in this class.
  ///
  /// The return value is guaranteed to never be null.
  Client get client => _client;

  /// Requests to use a new HTTP [Client] in this class.
  set client(Client newClient) {
    _client = newClient;
  }

  Map<String, String> get defaultHeaderMap => _defaultHeaderMap;

  void addDefaultHeader(String key, String value) {
    _defaultHeaderMap[key] = value;
  }

  // We don't use a Map<String, String> for queryParams.
  // If collectionFormat is 'multi', a key might appear multiple times.
  Future<Response> invokeAPI(
    String path,
    String method,
    List<QueryParam> queryParams,
    Object? body,
    Map<String, String> headerParams,
    Map<String, String> formParams,
    String? contentType, {
    Future<void>? abortTrigger,
  }) async {
    await authentication?.applyToParams(queryParams, headerParams);

    headerParams.addAll(_defaultHeaderMap);
    if (contentType != null) {
      headerParams['Content-Type'] = contentType;
    }

    final urlEncodedQueryParams = queryParams.map((param) => '$param');
    final queryString = urlEncodedQueryParams.isNotEmpty
        ? '?${urlEncodedQueryParams.join('&')}'
        : '';
    final uri = Uri.parse('$basePath$path$queryString');

    try {
      // Special case for uploading a single file which isn't a 'multipart/form-data'.
      if (body is MultipartFile &&
          (contentType == null ||
              !contentType.toLowerCase().startsWith('multipart/form-data'))) {
        final request =
            AbortableStreamedRequest(method, uri, abortTrigger: abortTrigger);
        request.headers.addAll(headerParams);
        request.contentLength = body.length;
        body.finalize().listen(
              request.sink.add,
              onDone: request.sink.close,
              // ignore: avoid_types_on_closure_parameters
              onError: (Object error, StackTrace trace) => request.sink.close(),
              cancelOnError: true,
            );
        final response = await _client.send(request);
        return Response.fromStream(response);
      }

      if (body is MultipartRequest) {
        final request =
            AbortableMultipartRequest(method, uri, abortTrigger: abortTrigger);
        request.fields.addAll(body.fields);
        request.files.addAll(body.files);
        request.headers.addAll(body.headers);
        request.headers.addAll(headerParams);
        final response = await _client.send(request);
        return Response.fromStream(response);
      }

      final msgBody = contentType == 'application/x-www-form-urlencoded'
          ? formParams
          : await serializeAsync(body);
      final nullableHeaderParams = headerParams.isEmpty ? null : headerParams;

      final request = AbortableRequest(method, uri, abortTrigger: abortTrigger);
      if (nullableHeaderParams != null) {
        request.headers.addAll(nullableHeaderParams);
      }
      if (msgBody is String && msgBody.isNotEmpty) {
        request.body = msgBody;
      } else if (msgBody is List<int> && msgBody.isNotEmpty) {
        request.bodyBytes = msgBody;
      } else if (msgBody is Map<String, String>) {
        request.bodyFields = msgBody;
      }
      final response = await _client.send(request);
      return Response.fromStream(response);
    } on SocketException catch (error, trace) {
      throw ApiException.withInner(
        HttpStatus.badRequest,
        'Socket operation failed: $method $path',
        error,
        trace,
      );
    } on TlsException catch (error, trace) {
      throw ApiException.withInner(
        HttpStatus.badRequest,
        'TLS/SSL communication failed: $method $path',
        error,
        trace,
      );
    } on IOException catch (error, trace) {
      throw ApiException.withInner(
        HttpStatus.badRequest,
        'I/O operation failed: $method $path',
        error,
        trace,
      );
    } on ClientException catch (error, trace) {
      throw ApiException.withInner(
        HttpStatus.badRequest,
        'HTTP connection failed: $method $path',
        error,
        trace,
      );
    } on Exception catch (error, trace) {
      throw ApiException.withInner(
        HttpStatus.badRequest,
        'Exception occurred: $method $path',
        error,
        trace,
      );
    }
  }

  Future<dynamic> deserializeAsync(
    String value,
    String targetType, {
    bool growable = false,
  }) async =>
      // ignore: deprecated_member_use_from_same_package
      deserialize(value, targetType, growable: growable);

  @Deprecated(
      'Scheduled for removal in OpenAPI Generator 6.x. Use deserializeAsync() instead.')
  dynamic deserialize(
    String value,
    String targetType, {
    bool growable = false,
  }) {
    // Remove all spaces. Necessary for regular expressions as well.
    targetType =
        targetType.replaceAll(' ', ''); // ignore: parameter_assignments

    // If the expected target type is String, nothing to do...
    return targetType == 'String'
        ? value
        : fromJson(json.decode(value), targetType, growable: growable);
  }

  // ignore: deprecated_member_use_from_same_package
  Future<String> serializeAsync(Object? value) async => serialize(value);

  @Deprecated(
      'Scheduled for removal in OpenAPI Generator 6.x. Use serializeAsync() instead.')
  String serialize(Object? value) => value == null ? '' : json.encode(value);

  /// Returns a native instance of an OpenAPI class matching the [specified type][targetType].
  static dynamic fromJson(
    dynamic value,
    String targetType, {
    bool growable = false,
  }) {
    try {
      switch (targetType) {
        case 'String':
          return value is String ? value : value.toString();
        case 'int':
          return value is int ? value : int.parse('$value');
        case 'double':
          return value is double ? value : double.parse('$value');
        case 'bool':
          if (value is bool) {
            return value;
          }
          final valueString = '$value'.toLowerCase();
          return valueString == 'true' || valueString == '1';
        case 'DateTime':
          return value is DateTime ? value : DateTime.tryParse(value);
        case 'AcceptInviteRequest':
          return AcceptInviteRequest.fromJson(value);
        case 'Acceptance':
          return Acceptance.fromJson(value);
        case 'Account':
          return Account.fromJson(value);
        case 'Answer':
          return Answer.fromJson(value);
        case 'AnswerValue':
          return AnswerValue.fromJson(value);
        case 'Appointment':
          return Appointment.fromJson(value);
        case 'Availability':
          return Availability.fromJson(value);
        case 'AvailabilityRule':
          return AvailabilityRule.fromJson(value);
        case 'Binary':
          return Binary.fromJson(value);
        case 'BookAppointmentRequest':
          return BookAppointmentRequest.fromJson(value);
        case 'ChatCommand':
          return ChatCommand.fromJson(value);
        case 'ChatCommandError':
          return ChatCommandError.fromJson(value);
        case 'ChatCommandErrorFrame':
          return ChatCommandErrorFrame.fromJson(value);
        case 'ChatFrame':
          return ChatFrame.fromJson(value);
        case 'ChatMessage':
          return ChatMessage.fromJson(value);
        case 'ChatMessageFrame':
          return ChatMessageFrame.fromJson(value);
        case 'CommandError':
          return CommandError.fromJson(value);
        case 'CommandErrorMessage':
          return CommandErrorMessage.fromJson(value);
        case 'ConfirmRequest':
          return ConfirmRequest.fromJson(value);
        case 'Consent':
          return Consent.fromJson(value);
        case 'ConsentKind':
          return ConsentKindTypeTransformer().decode(value);
        case 'ConsentList':
          return ConsentList.fromJson(value);
        case 'ConsultCommand':
          return ConsultCommand.fromJson(value);
        case 'ConsultFrame':
          return ConsultFrame.fromJson(value);
        case 'ConsultMessage':
          return ConsultMessage.fromJson(value);
        case 'ConsultationJoin':
          return ConsultationJoin.fromJson(value);
        case 'ConsultationStatus':
          return ConsultationStatus.fromJson(value);
        case 'Conversation':
          return Conversation.fromJson(value);
        case 'ConversationList':
          return ConversationList.fromJson(value);
        case 'ConversationParticipant':
          return ConversationParticipant.fromJson(value);
        case 'Cursor':
          return Cursor.fromJson(value);
        case 'CursorMessage':
          return CursorMessage.fromJson(value);
        case 'DocumentFieldTexts':
          return DocumentFieldTexts.fromJson(value);
        case 'DocumentTemplate':
          return DocumentTemplate.fromJson(value);
        case 'DocumentTemplateField':
          return DocumentTemplateField.fromJson(value);
        case 'DocumentTemplateTranslation':
          return DocumentTemplateTranslation.fromJson(value);
        case 'DocumentTemplates':
          return DocumentTemplates.fromJson(value);
        case 'DocumentTerms':
          return DocumentTerms.fromJson(value);
        case 'DocumentTermsSection':
          return DocumentTermsSection.fromJson(value);
        case 'DraftEvent':
          return DraftEvent.fromJson(value);
        case 'DraftField':
          return DraftField.fromJson(value);
        case 'DraftMessage':
          return DraftMessage.fromJson(value);
        case 'DraftValue':
          return DraftValue.fromJson(value);
        case 'Drafts':
          return Drafts.fromJson(value);
        case 'EditMessageRequest':
          return EditMessageRequest.fromJson(value);
        case 'EnrolRequest':
          return EnrolRequest.fromJson(value);
        case 'Enrolment':
          return Enrolment.fromJson(value);
        case 'Error':
          return Error.fromJson(value);
        case 'ExplainedTerm':
          return ExplainedTerm.fromJson(value);
        case 'ExplainedTermList':
          return ExplainedTermList.fromJson(value);
        case 'GenerateDraftRequest':
          return GenerateDraftRequest.fromJson(value);
        case 'GrantConsentRequest':
          return GrantConsentRequest.fromJson(value);
        case 'Invite':
          return Invite.fromJson(value);
        case 'InviteList':
          return InviteList.fromJson(value);
        case 'InviteRequest':
          return InviteRequest.fromJson(value);
        case 'KeyedError':
          return KeyedError.fromJson(value);
        case 'Language':
          return LanguageTypeTransformer().decode(value);
        case 'LiveKitWebhookEvent':
          return LiveKitWebhookEvent.fromJson(value);
        case 'LoginRequest':
          return LoginRequest.fromJson(value);
        case 'LoginResponse':
          return LoginResponse.fromJson(value);
        case 'MessagePage':
          return MessagePage.fromJson(value);
        case 'MessageRevision':
          return MessageRevision.fromJson(value);
        case 'MessageRevisions':
          return MessageRevisions.fromJson(value);
        case 'OpenConversationRequest':
          return OpenConversationRequest.fromJson(value);
        case 'Patient':
          return Patient.fromJson(value);
        case 'Questionnaire':
          return Questionnaire.fromJson(value);
        case 'QuestionnaireDefault':
          return QuestionnaireDefault.fromJson(value);
        case 'QuestionnaireItem':
          return QuestionnaireItem.fromJson(value);
        case 'QuestionnaireItemAnswerOptionInner':
          return QuestionnaireItemAnswerOptionInner.fromJson(value);
        case 'QuestionnaireItemEnableWhen':
          return QuestionnaireItemEnableWhen.fromJson(value);
        case 'QuestionnaireItemTranslation':
          return QuestionnaireItemTranslation.fromJson(value);
        case 'QuestionnaireResponse':
          return QuestionnaireResponse.fromJson(value);
        case 'QuestionnaireTranslation':
          return QuestionnaireTranslation.fromJson(value);
        case 'ReadPosition':
          return ReadPosition.fromJson(value);
        case 'RegisterRequest':
          return RegisterRequest.fromJson(value);
        case 'RequestQuestionnaireRequest':
          return RequestQuestionnaireRequest.fromJson(value);
        case 'Review':
          return Review.fromJson(value);
        case 'ReviewDecision':
          return ReviewDecision.fromJson(value);
        case 'ReviewStatus':
          return ReviewStatusTypeTransformer().decode(value);
        case 'RoleSpecialty':
          return RoleSpecialty.fromJson(value);
        case 'SaveAnswersRequest':
          return SaveAnswersRequest.fromJson(value);
        case 'SaveAvailabilityRequest':
          return SaveAvailabilityRequest.fromJson(value);
        case 'SaveDocumentTemplateRequest':
          return SaveDocumentTemplateRequest.fromJson(value);
        case 'SavePatientRequest':
          return SavePatientRequest.fromJson(value);
        case 'SaveQuestionnaireRequest':
          return SaveQuestionnaireRequest.fromJson(value);
        case 'SendAck':
          return SendAck.fromJson(value);
        case 'SendAckFrame':
          return SendAckFrame.fromJson(value);
        case 'SendMessageRequest':
          return SendMessageRequest.fromJson(value);
        case 'SetAppointmentStatusRequest':
          return SetAppointmentStatusRequest.fromJson(value);
        case 'SetRoleQuestionnaireRequest':
          return SetRoleQuestionnaireRequest.fromJson(value);
        case 'SetSpecialtyRequest':
          return SetSpecialtyRequest.fromJson(value);
        case 'SignDraftRequest':
          return SignDraftRequest.fromJson(value);
        case 'Specialty':
          return Specialty.fromJson(value);
        case 'SpecialtyList':
          return SpecialtyList.fromJson(value);
        case 'StartQuestionnaireRequest':
          return StartQuestionnaireRequest.fromJson(value);
        case 'Status':
          return Status.fromJson(value);
        case 'SuggestionEvent':
          return SuggestionEvent.fromJson(value);
        case 'SuggestionMessage':
          return SuggestionMessage.fromJson(value);
        case 'SuggestionStatus':
          return SuggestionStatus.fromJson(value);
        case 'SuggestionStatusMessage':
          return SuggestionStatusMessage.fromJson(value);
        case 'SummaryRequest':
          return SummaryRequest.fromJson(value);
        case 'SummaryResponse':
          return SummaryResponse.fromJson(value);
        case 'SystemMessage':
          return SystemMessage.fromJson(value);
        case 'Term':
          return Term.fromJson(value);
        case 'TermCategory':
          return TermCategoryTypeTransformer().decode(value);
        case 'TermCodes':
          return TermCodes.fromJson(value);
        case 'TermExplanation':
          return TermExplanation.fromJson(value);
        case 'TermExplanationEdit':
          return TermExplanationEdit.fromJson(value);
        case 'TermLabels':
          return TermLabels.fromJson(value);
        case 'TermLabelsEdit':
          return TermLabelsEdit.fromJson(value);
        case 'TermLanguage':
          return TermLanguageTypeTransformer().decode(value);
        case 'TermList':
          return TermList.fromJson(value);
        case 'TermReviewEntry':
          return TermReviewEntry.fromJson(value);
        case 'TermReviewPage':
          return TermReviewPage.fromJson(value);
        case 'TermReviewProgress':
          return TermReviewProgress.fromJson(value);
        case 'TermReviewProgressLanguagesInner':
          return TermReviewProgressLanguagesInner.fromJson(value);
        case 'TermSpan':
          return TermSpan.fromJson(value);
        case 'TermTranslation':
          return TermTranslation.fromJson(value);
        case 'Transcript':
          return Transcript.fromJson(value);
        case 'TranscriptEvent':
          return TranscriptEvent.fromJson(value);
        case 'TranscriptMessage':
          return TranscriptMessage.fromJson(value);
        case 'TranscriptTerms':
          return TranscriptTerms.fromJson(value);
        case 'Unavailable':
          return Unavailable.fromJson(value);
        case 'UpdateMeRequest':
          return UpdateMeRequest.fromJson(value);
        case 'VerifyRequest':
          return VerifyRequest.fromJson(value);
        default:
          dynamic match;
          if (value is List &&
              (match = _regList.firstMatch(targetType)?.group(1)) != null) {
            return value
                .map<dynamic>((dynamic v) => fromJson(
                      v,
                      match,
                      growable: growable,
                    ))
                .toList(growable: growable);
          }
          if (value is Set &&
              (match = _regSet.firstMatch(targetType)?.group(1)) != null) {
            return value
                .map<dynamic>((dynamic v) => fromJson(
                      v,
                      match,
                      growable: growable,
                    ))
                .toSet();
          }
          if (value is Map &&
              (match = _regMap.firstMatch(targetType)?.group(1)) != null) {
            return Map<String, dynamic>.fromIterables(
              value.keys.cast<String>(),
              value.values.map<dynamic>((dynamic v) => fromJson(
                    v,
                    match,
                    growable: growable,
                  )),
            );
          }
      }
    } on Exception catch (error, trace) {
      throw ApiException.withInner(
        HttpStatus.internalServerError,
        'Exception during deserialization.',
        error,
        trace,
      );
    }
    throw ApiException(
      HttpStatus.internalServerError,
      'Could not find a suitable class for deserialization',
    );
  }
}

/// Primarily intended for use in an isolate.
class DeserializationMessage {
  const DeserializationMessage({
    required this.json,
    required this.targetType,
    this.growable = false,
  });

  /// The JSON value to deserialize.
  final String json;

  /// Target type to deserialize to.
  final String targetType;

  /// Whether to make deserialized lists or maps growable.
  final bool growable;
}

/// Primarily intended for use in an isolate.
Future<dynamic> decodeAsync(DeserializationMessage message) async {
  // Remove all spaces. Necessary for regular expressions as well.
  final targetType = message.targetType.replaceAll(' ', '');

  // If the expected target type is String, nothing to do...
  return targetType == 'String' ? message.json : json.decode(message.json);
}

/// Primarily intended for use in an isolate.
Future<dynamic> deserializeAsync(DeserializationMessage message) async {
  // Remove all spaces. Necessary for regular expressions as well.
  final targetType = message.targetType.replaceAll(' ', '');

  // If the expected target type is String, nothing to do...
  return targetType == 'String'
      ? message.json
      : ApiClient.fromJson(
          json.decode(message.json),
          targetType,
          growable: message.growable,
        );
}

/// Primarily intended for use in an isolate.
Future<String> serializeAsync(Object? value) async =>
    value == null ? '' : json.encode(value);
