//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

library salus_api;

import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:collection/collection.dart';
import 'package:http/http.dart';
import 'package:intl/intl.dart';
import 'package:meta/meta.dart';

part 'api_client.dart';
part 'api_helper.dart';
part 'api_exception.dart';
part 'auth/authentication.dart';
part 'auth/api_key_auth.dart';
part 'auth/oauth.dart';
part 'auth/http_basic_auth.dart';
part 'auth/http_bearer_auth.dart';

part 'api/appointments_api.dart';
part 'api/attachments_api.dart';
part 'api/auth_api.dart';
part 'api/consents_api.dart';
part 'api/consultation_api.dart';
part 'api/directory_api.dart';
part 'api/health_api.dart';
part 'api/invites_api.dart';
part 'api/patient_api.dart';
part 'api/questionnaires_api.dart';
part 'api/scheduling_api.dart';

part 'model/accept_invite_request.dart';
part 'model/acceptance.dart';
part 'model/account.dart';
part 'model/answer.dart';
part 'model/answer_value.dart';
part 'model/appointment.dart';
part 'model/availability.dart';
part 'model/availability_rule.dart';
part 'model/binary.dart';
part 'model/book_appointment_request.dart';
part 'model/confirm_request.dart';
part 'model/consent.dart';
part 'model/consent_list.dart';
part 'model/consultation_join.dart';
part 'model/consultation_status.dart';
part 'model/enrol_request.dart';
part 'model/enrolment.dart';
part 'model/error.dart';
part 'model/grant_consent_request.dart';
part 'model/invite.dart';
part 'model/invite_list.dart';
part 'model/invite_request.dart';
part 'model/language.dart';
part 'model/live_kit_webhook_event.dart';
part 'model/login_request.dart';
part 'model/login_response.dart';
part 'model/patient.dart';
part 'model/questionnaire.dart';
part 'model/questionnaire_default.dart';
part 'model/questionnaire_item.dart';
part 'model/questionnaire_item_answer_option_inner.dart';
part 'model/questionnaire_item_enable_when.dart';
part 'model/questionnaire_item_translation.dart';
part 'model/questionnaire_response.dart';
part 'model/questionnaire_translation.dart';
part 'model/register_request.dart';
part 'model/request_questionnaire_request.dart';
part 'model/role_specialty.dart';
part 'model/save_answers_request.dart';
part 'model/save_availability_request.dart';
part 'model/save_patient_request.dart';
part 'model/save_questionnaire_request.dart';
part 'model/set_appointment_status_request.dart';
part 'model/set_role_questionnaire_request.dart';
part 'model/set_specialty_request.dart';
part 'model/specialty.dart';
part 'model/specialty_list.dart';
part 'model/start_questionnaire_request.dart';
part 'model/status.dart';
part 'model/unavailable.dart';
part 'model/update_me_request.dart';
part 'model/verify_request.dart';

/// An [ApiClient] instance that uses the default values obtained from
/// the OpenAPI specification file.
var defaultApiClient = ApiClient();

const _delimiters = {'csv': ',', 'ssv': ' ', 'tsv': '\t', 'pipes': '|'};
const _dateEpochMarker = 'epoch';
const _deepEquality = DeepCollectionEquality();
final _dateFormatter = DateFormat('yyyy-MM-dd');
final _regList = RegExp(r'^List<(.*)>$');
final _regSet = RegExp(r'^Set<(.*)>$');
final _regMap = RegExp(r'^Map<String,(.*)>$');

bool _isEpochMarker(String? pattern) =>
    pattern == _dateEpochMarker || pattern == '/$_dateEpochMarker/';
