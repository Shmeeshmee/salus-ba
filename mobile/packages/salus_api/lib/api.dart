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
part 'api/chat_api.dart';
part 'api/consents_api.dart';
part 'api/consultation_api.dart';
part 'api/directory_api.dart';
part 'api/documents_api.dart';
part 'api/health_api.dart';
part 'api/history_api.dart';
part 'api/invites_api.dart';
part 'api/patient_api.dart';
part 'api/questionnaires_api.dart';
part 'api/scheduling_api.dart';
part 'api/terminology_api.dart';

part 'model/accept_invite_request.dart';
part 'model/acceptance.dart';
part 'model/account.dart';
part 'model/allergy_category.dart';
part 'model/allergy_clinical_status.dart';
part 'model/allergy_intolerance_input.dart';
part 'model/allergy_severity.dart';
part 'model/answer.dart';
part 'model/answer_value.dart';
part 'model/appointment.dart';
part 'model/availability.dart';
part 'model/availability_rule.dart';
part 'model/binary.dart';
part 'model/book_appointment_request.dart';
part 'model/chat_command.dart';
part 'model/chat_command_error.dart';
part 'model/chat_command_error_frame.dart';
part 'model/chat_frame.dart';
part 'model/chat_message.dart';
part 'model/chat_message_frame.dart';
part 'model/command_error.dart';
part 'model/command_error_message.dart';
part 'model/confirm_request.dart';
part 'model/consent.dart';
part 'model/consent_kind.dart';
part 'model/consent_list.dart';
part 'model/consult_command.dart';
part 'model/consult_frame.dart';
part 'model/consult_message.dart';
part 'model/consultation_join.dart';
part 'model/consultation_status.dart';
part 'model/conversation.dart';
part 'model/conversation_list.dart';
part 'model/conversation_participant.dart';
part 'model/cursor.dart';
part 'model/cursor_message.dart';
part 'model/diagnosis_group.dart';
part 'model/diagnosis_group_list.dart';
part 'model/diagnosis_group_name.dart';
part 'model/diagnosis_group_name_list.dart';
part 'model/document_field_texts.dart';
part 'model/document_template.dart';
part 'model/document_template_field.dart';
part 'model/document_template_translation.dart';
part 'model/document_templates.dart';
part 'model/document_terms.dart';
part 'model/document_terms_section.dart';
part 'model/dosage.dart';
part 'model/dose_unit.dart';
part 'model/draft_event.dart';
part 'model/draft_field.dart';
part 'model/draft_message.dart';
part 'model/draft_value.dart';
part 'model/drafts.dart';
part 'model/edit_diagnosis_group_name_request.dart';
part 'model/edit_message_request.dart';
part 'model/enrol_request.dart';
part 'model/enrolment.dart';
part 'model/entry_written.dart';
part 'model/error.dart';
part 'model/explained_term.dart';
part 'model/explained_term_list.dart';
part 'model/generate_draft_request.dart';
part 'model/grant_consent_request.dart';
part 'model/history_clinic.dart';
part 'model/history_item.dart';
part 'model/history_item_type.dart';
part 'model/history_page.dart';
part 'model/history_sharing.dart';
part 'model/history_sharing_list.dart';
part 'model/history_summary.dart';
part 'model/invite.dart';
part 'model/invite_list.dart';
part 'model/invite_request.dart';
part 'model/keyed_error.dart';
part 'model/language.dart';
part 'model/live_kit_webhook_event.dart';
part 'model/login_request.dart';
part 'model/login_response.dart';
part 'model/medication_statement_input.dart';
part 'model/medication_status.dart';
part 'model/message_page.dart';
part 'model/message_revision.dart';
part 'model/message_revisions.dart';
part 'model/open_conversation_request.dart';
part 'model/open_item.dart';
part 'model/patient.dart';
part 'model/period_unit.dart';
part 'model/place.dart';
part 'model/place_list.dart';
part 'model/questionnaire.dart';
part 'model/questionnaire_default.dart';
part 'model/questionnaire_item.dart';
part 'model/questionnaire_item_answer_option_inner.dart';
part 'model/questionnaire_item_enable_when.dart';
part 'model/questionnaire_item_translation.dart';
part 'model/questionnaire_response.dart';
part 'model/questionnaire_translation.dart';
part 'model/read_position.dart';
part 'model/register_request.dart';
part 'model/request_questionnaire_request.dart';
part 'model/review.dart';
part 'model/review_decision.dart';
part 'model/review_status.dart';
part 'model/role_specialty.dart';
part 'model/save_answers_request.dart';
part 'model/save_availability_request.dart';
part 'model/save_document_template_request.dart';
part 'model/save_patient_request.dart';
part 'model/save_questionnaire_request.dart';
part 'model/send_ack.dart';
part 'model/send_ack_frame.dart';
part 'model/send_message_request.dart';
part 'model/set_appointment_status_request.dart';
part 'model/set_history_sharing_request.dart';
part 'model/set_role_questionnaire_request.dart';
part 'model/set_specialty_request.dart';
part 'model/sharing_candidate.dart';
part 'model/sharing_candidate_list.dart';
part 'model/sign_draft_request.dart';
part 'model/specialty.dart';
part 'model/specialty_list.dart';
part 'model/start_questionnaire_request.dart';
part 'model/status.dart';
part 'model/suggestion_event.dart';
part 'model/suggestion_message.dart';
part 'model/suggestion_status.dart';
part 'model/suggestion_status_message.dart';
part 'model/summary_allergy.dart';
part 'model/summary_medication.dart';
part 'model/system_message.dart';
part 'model/term.dart';
part 'model/term_category.dart';
part 'model/term_codes.dart';
part 'model/term_explanation.dart';
part 'model/term_explanation_edit.dart';
part 'model/term_labels.dart';
part 'model/term_labels_edit.dart';
part 'model/term_language.dart';
part 'model/term_list.dart';
part 'model/term_review_entry.dart';
part 'model/term_review_page.dart';
part 'model/term_review_progress.dart';
part 'model/term_review_progress_languages_inner.dart';
part 'model/term_span.dart';
part 'model/term_translation.dart';
part 'model/transcript.dart';
part 'model/transcript_event.dart';
part 'model/transcript_message.dart';
part 'model/transcript_terms.dart';
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
