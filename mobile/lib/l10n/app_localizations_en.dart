// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class AppLocalizationsEn extends AppLocalizations {
  AppLocalizationsEn([String locale = 'en']) : super(locale);

  @override
  String get tryAgain => 'Try again';

  @override
  String get language => 'Language';

  @override
  String get useDeviceLanguage => 'Use the device\'s language';

  @override
  String get languageSaved => 'Language saved.';

  @override
  String get somethingWrong => 'Something went wrong. Try again.';

  @override
  String get unreachable => 'Salus could not be reached. Check your connection and try again.';

  @override
  String get wrongCredentials => 'The e-mail or password is wrong.';

  @override
  String get wrongCode => 'The code is wrong. Check the time on your device and try again.';

  @override
  String get challengeEnded => 'The sign-in has ended. Enter your e-mail and password again.';

  @override
  String get emailTaken => 'This e-mail already has an account. Sign in instead.';

  @override
  String get invalidEmail => 'Enter a valid e-mail address.';

  @override
  String get invalidPassword => 'Use 12 to 128 characters for the password.';

  @override
  String get missingPassword => 'Enter your password.';

  @override
  String get invalidCode => 'The code has 6 digits.';

  @override
  String get staffOnly => 'This app is for patients. Clinic staff use the Salus web app.';

  @override
  String get accountReady => 'Your account is ready. Sign in to continue.';

  @override
  String get signOutFailed => 'Could not sign out. Check your connection and try again.';

  @override
  String get tooManyAttemptsLater => 'Too many attempts. Try again later.';

  @override
  String tooManyAttempts(int minutes) {
    String _temp0 = intl.Intl.pluralLogic(
      minutes,
      locale: localeName,
      other: 'Too many attempts. Try again in $minutes minutes.',
      one: 'Too many attempts. Try again in $minutes minute.',
    );
    return '$_temp0';
  }

  @override
  String get signInTitle => 'Sign in to Salus';

  @override
  String get email => 'E-mail';

  @override
  String get password => 'Password';

  @override
  String get signIn => 'Sign in';

  @override
  String get createAnAccount => 'Create an account';

  @override
  String get codePrompt => 'Enter the 6-digit code from your authenticator app.';

  @override
  String get code => 'Code';

  @override
  String get useAnotherAccount => 'Use another account';

  @override
  String get passwordHint => '12 to 128 characters';

  @override
  String get createAccount => 'Create account';

  @override
  String get haveAccount => 'I already have an account';

  @override
  String signedInAs(String email) {
    return 'Signed in as $email';
  }

  @override
  String get completeProfile => 'Complete your profile';

  @override
  String get completeProfileFirst => 'Complete your profile before you book.';

  @override
  String get bookAppointment => 'Book an appointment';

  @override
  String get myAppointments => 'My appointments';

  @override
  String get myConsents => 'My consents';

  @override
  String get myProfile => 'My profile';

  @override
  String get signOut => 'Sign out';

  @override
  String get profileIntro =>
      'Clinics you book with see these details. The phone number is optional; the practitioner uses it only to reach you about an appointment.';

  @override
  String get givenName => 'Given name';

  @override
  String get familyName => 'Family name';

  @override
  String get dateOfBirth => 'Date of birth';

  @override
  String get dateFormatHint => 'YYYY-MM-DD';

  @override
  String get phoneOptional => 'Phone (optional)';

  @override
  String phoneHint(String example) {
    return 'With the country code, for example $example';
  }

  @override
  String get save => 'Save';

  @override
  String get notNow => 'Not now';

  @override
  String get invalidGivenName => 'Enter your given name, up to 100 characters.';

  @override
  String get invalidFamilyName => 'Enter your family name, up to 100 characters.';

  @override
  String get invalidName => 'Enter your given and family name, up to 100 characters each.';

  @override
  String get birthDateFormat => 'Enter your date of birth as YYYY-MM-DD.';

  @override
  String get birthDateRange => 'Enter a date of birth in the past, from 1900 on.';

  @override
  String invalidPhone(String example) {
    return 'Enter the phone number with its country code, for example $example.';
  }

  @override
  String get profileChanged => 'Your profile was changed in the meantime. Check it and save again.';

  @override
  String get chooseClinic => 'Choose a clinic';

  @override
  String get noClinics => 'No clinics are listed yet.';

  @override
  String get noPractitioners => 'This clinic has no practitioners listed yet.';

  @override
  String get choosePractitioner => 'Choose a practitioner.';

  @override
  String get noOpenTimes => 'This practitioner has no open times yet.';

  @override
  String timesIn(String zone) {
    return 'Times are in $zone.';
  }

  @override
  String get previousDay => 'Previous day';

  @override
  String get nextDay => 'Next day';

  @override
  String get noFreeTimes => 'No free times on this day.';

  @override
  String get bookThisTime => 'Book this time';

  @override
  String get chooseAnotherTime => 'Choose another time';

  @override
  String get book => 'Book';

  @override
  String timeRange(String start, String end, String zone) {
    return '$start to $end ($zone)';
  }

  @override
  String get slotTaken => 'This time is no longer free. Choose another time.';

  @override
  String get consentNeeded => 'Booking with this clinic needs your consent.';

  @override
  String get appointmentBooked => 'Your appointment is booked.';

  @override
  String get unlistedClinic => 'A clinic no longer listed';

  @override
  String get cancelTitle => 'Cancel this appointment?';

  @override
  String cancelBody(String date, String time) {
    return 'Your appointment on $date at $time will be cancelled. You will get an e-mail to confirm it.';
  }

  @override
  String get keepIt => 'Keep it';

  @override
  String get cancelAppointment => 'Cancel appointment';

  @override
  String get noAppointments => 'You have no appointments yet.';

  @override
  String get upcoming => 'Upcoming';

  @override
  String get past => 'Past';

  @override
  String get statusBooked => 'Booked';

  @override
  String get statusCancelled => 'Cancelled';

  @override
  String get statusAttended => 'Attended';

  @override
  String get statusNotAttended => 'Not attended';

  @override
  String get appointmentCancelled => 'Your appointment is cancelled.';

  @override
  String get cancelWindowHint => 'Can be cancelled in the app until 24 hours before the start.';

  @override
  String get cancelWindowClosed =>
      'This appointment starts in less than 24 hours and can no longer be cancelled in the app.';

  @override
  String get appointmentClosed => 'This appointment is no longer booked.';

  @override
  String consentTitle(String clinic) {
    return 'Share your details with $clinic';
  }

  @override
  String consentIntro(String clinic) {
    return 'Before your first booking with $clinic, we need your consent to share your details with it.';
  }

  @override
  String get whatIsShared => 'What is shared';

  @override
  String get whatIsSharedBody =>
      'Your profile (your name, date of birth and, if you gave one, your phone number) and your appointments with this clinic.';

  @override
  String get whoSeesIt => 'Who sees it';

  @override
  String whoSeesItBody(String clinic) {
    return '$clinic. The practitioner you book with sees your profile; the clinic\'s administrators see your name and appointment times.';
  }

  @override
  String get whatFor => 'What for';

  @override
  String get whatForBody => 'Only for your care at this clinic: to prepare for and hold your appointments.';

  @override
  String get whoIsResponsible => 'Who is responsible';

  @override
  String whoIsResponsibleBody(String clinic) {
    return '$clinic is responsible for your data. Salus processes it on the clinic\'s behalf.';
  }

  @override
  String get readPrivacyNotice => 'Read the privacy notice';

  @override
  String get privacyNoticeFailed => 'The privacy notice could not be opened.';

  @override
  String get howToRevoke => 'How to revoke';

  @override
  String get howToRevokeBody =>
      'You can revoke this consent at any time under My consents. The clinic then no longer sees your profile. It keeps your name and the details of appointments that have already taken place, as part of its care records. Your future appointments there are cancelled.';

  @override
  String get agreeAndBook => 'Agree and book';

  @override
  String get revokeTitle => 'Revoke your consent?';

  @override
  String revokeBody(String clinic) {
    return '$clinic will no longer see your profile. It keeps your name and the details of appointments that have already taken place, as part of its care records. Your future appointments with $clinic will be cancelled.';
  }

  @override
  String get keepConsent => 'Keep consent';

  @override
  String get revokeConsent => 'Revoke consent';

  @override
  String get noConsents => 'You have not given any consents yet.';

  @override
  String get consentsIntro => 'You share your profile and appointments with these clinics for your care.';

  @override
  String givenOn(String date) {
    return 'Given on $date';
  }

  @override
  String get consentActive => 'Active';

  @override
  String revokedOn(String date) {
    return 'Revoked on $date';
  }

  @override
  String get consentRevoked =>
      'Your consent is revoked. Your future appointments with this clinic will be cancelled, and you will get an e-mail for each.';

  @override
  String get consentAlreadyRevoked => 'This consent is already revoked.';

  @override
  String get joinConsultation => 'Join consultation';

  @override
  String get joinFrom => 'You can join from 10 minutes before the start.';

  @override
  String get checkTitle => 'Check your camera and microphone';

  @override
  String get checkIntro => 'The practitioner will see and hear you once you join.';

  @override
  String get noSound => 'No sound detected. Check that your microphone is on and not muted.';

  @override
  String get joinCameraOff => 'Join with camera off';

  @override
  String get joinMicrophoneOff => 'Join with microphone off';

  @override
  String get joinNow => 'Join';

  @override
  String get joining => 'Joining…';

  @override
  String get cameraDenied =>
      'Salus does not have access to your camera. You can allow it in your device settings, or join without video.';

  @override
  String get microphoneDenied =>
      'Salus does not have access to your microphone. You can allow it in your device settings. Without it, the practitioner cannot hear you.';

  @override
  String get openSettings => 'Open settings';

  @override
  String get noCamera => 'No camera found.';

  @override
  String get noMicrophone => 'No microphone found.';

  @override
  String get waiting => 'Waiting for the practitioner';

  @override
  String joinsYouHere(String name) {
    return '$name will join you here. Keep this screen open.';
  }

  @override
  String get mute => 'Mute microphone';

  @override
  String get unmute => 'Unmute microphone';

  @override
  String get cameraOff => 'Turn camera off';

  @override
  String get cameraOn => 'Turn camera on';

  @override
  String get leave => 'Leave';

  @override
  String peerCameraOff(String name) {
    return '$name\'s camera is off';
  }

  @override
  String peerMuted(String name) {
    return '$name is muted';
  }

  @override
  String peerLeft(String name) {
    return '$name has left the consultation. They can join again until the consultation ends.';
  }

  @override
  String get reconnecting => 'Connection lost. Reconnecting…';

  @override
  String get notRestored => 'The connection could not be restored.';

  @override
  String get reconnected => 'Reconnected';

  @override
  String get consultationOver => 'The consultation has ended';

  @override
  String get closeScreen => 'You can close this screen.';

  @override
  String get timeLimit => 'The consultation reached its time limit and has ended.';

  @override
  String get practitionerAbsent => 'The practitioner did not join';

  @override
  String get timePassed =>
      'The time to join this consultation has passed. The clinic can see that you joined. The clinic may contact you, or you can book a new appointment.';

  @override
  String get withdrewConsent => 'You withdrew your consent for this clinic, so you have left the consultation.';

  @override
  String get appointmentWasCancelled => 'This appointment was cancelled, so the consultation has ended.';

  @override
  String get windowNotOpen => 'This consultation opens 10 minutes before the start.';

  @override
  String get consultationEnded => 'This consultation has ended.';

  @override
  String get joinNeedsConsent => 'Joining needs your consent for this clinic.';

  @override
  String get consultationNotAvailable => 'This consultation is not available.';

  @override
  String get cannotConnect => 'The video connection could not be started. Check your connection and try again.';

  @override
  String get whyCameraAndMicrophone =>
      'To join the video consultation, Salus needs your camera and microphone. The consultation is not recorded as audio or video. If you agree, what is said is transcribed and translated.';

  @override
  String get transcriptionOn => 'Transcription is on';

  @override
  String get transcriptionTitle => 'Transcription and translation';

  @override
  String transcriptionIntro(String clinic) {
    return 'In your video consultations with $clinic, what you and the practitioner say can be transcribed as you speak and translated, so the practitioner can follow you in your language.';
  }

  @override
  String get transcriptionAssistance =>
      'From the transcript, the practitioner also gets AI assistance: suggestions during the consultation and drafts of documents. These are suggestions only; the practitioner checks them and makes every decision.';

  @override
  String transcriptionKept(String clinic) {
    return 'The transcript is kept with your medical record at $clinic. The consultation is not recorded as audio or video.';
  }

  @override
  String get transcriptionDecline =>
      'If you do not agree, the consultation goes ahead as usual, without transcription, translation or AI assistance.';

  @override
  String get transcriptionWithdraw =>
      'You can withdraw your agreement at any time, also during the consultation: tap “Transcription is on”, or open My consents. Transcription stops at once; what was transcribed until then stays in your record.';

  @override
  String get transcriptionAgree => 'Agree and continue';

  @override
  String get transcriptionWithout => 'Continue without transcription';

  @override
  String get stopTranscriptionTitle => 'Stop transcription?';

  @override
  String stopTranscriptionBody(String clinic) {
    return 'Transcription, translation and AI assistance stop at once in your consultations with $clinic. A consultation in progress goes on without them, and your appointments stay as they are. What was transcribed until now stays in your record.';
  }

  @override
  String get keepTranscription => 'Keep transcription';

  @override
  String get stopTranscription => 'Stop transcription';

  @override
  String get transcriptionStopped =>
      'Transcription is off. Your consultations with this clinic go ahead without it.';

  @override
  String get consentForCare => 'Sharing for care';

  @override
  String get consentForTranscription => 'Transcription and AI assistance';

  @override
  String get toFillIn => 'To fill in';

  @override
  String get inProgress => 'In progress';

  @override
  String get submittedStatus => 'Submitted';

  @override
  String get closedStatus => 'Closed';

  @override
  String get fillIn => 'Fill in';

  @override
  String get continueFilling => 'Continue';

  @override
  String get viewAnswers => 'View answers';

  @override
  String get submitBeforeStart => 'Please submit it before your appointment starts.';

  @override
  String get questionnaireClosed => 'The appointment has started, so this questionnaire can no longer be filled in.';

  @override
  String savedAsYouGo(String clinic) {
    return 'Your answers are saved as you go. $clinic sees them only after you submit.';
  }

  @override
  String get saved => 'Saved';

  @override
  String get saving => 'Saving…';

  @override
  String get notSaved => 'Not saved. Check your connection. Saving will try again.';

  @override
  String get requiredQuestion => 'Required';

  @override
  String get moreInformation => 'More information';

  @override
  String get yourAnswer => 'Your answer';

  @override
  String get chooseDate => 'Choose a date';

  @override
  String get selectOne => 'Select one.';

  @override
  String get selectAll => 'Select all that apply.';

  @override
  String get yes => 'Yes';

  @override
  String get no => 'No';

  @override
  String get questionRequired => 'This question is required.';

  @override
  String get enterWholeNumber => 'Enter a whole number.';

  @override
  String get enterNumber => 'Enter a number.';

  @override
  String get numberTooLarge => 'This number is too large.';

  @override
  String upToCharacters(int max) {
    return 'Enter up to $max characters.';
  }

  @override
  String get enterValidDate => 'Enter a valid date.';

  @override
  String get submit => 'Submit';

  @override
  String get submitTitle => 'Submit your answers?';

  @override
  String submitBody(String clinic) {
    return 'You cannot change them after you submit. Your practitioner at $clinic will see them.';
  }

  @override
  String get cancel => 'Cancel';

  @override
  String get requiredMissing => 'Some required questions are not answered.';

  @override
  String get answersSubmitted => 'Answers submitted';

  @override
  String get answersSubmittedBody =>
      'Your practitioner can see them before your appointment. You can view them from My appointments.';

  @override
  String get yourAnswers => 'Your answers';

  @override
  String submittedAt(String when) {
    return 'Submitted $when';
  }

  @override
  String dateAndTime(String date, String time) {
    return '$date, $time';
  }

  @override
  String get notAnswered => 'Not answered';

  @override
  String get leaveUnsaved => 'Your last answer is not saved yet. Leave anyway?';

  @override
  String get stay => 'Stay';

  @override
  String get fillingNeedsConsent => 'Filling in this questionnaire needs your consent for this clinic.';

  @override
  String get pastDue => 'The appointment has started, so this questionnaire can no longer be changed.';

  @override
  String get withdrawnByClinic => 'The clinic withdrew this questionnaire. You do not need to fill it in.';

  @override
  String get alreadySubmitted => 'You have already submitted this questionnaire.';

  @override
  String get changedElsewhere => 'These answers were changed on another device. The latest answers are shown.';

  @override
  String get questionnaireNotAvailable => 'This questionnaire is not available.';

  @override
  String get done => 'Done';

  @override
  String get addFile => 'Add file';

  @override
  String get photosUpTo => 'Photos up to 20 MB.';

  @override
  String get documentsUpTo => 'Documents (PDF) up to 20 MB.';

  @override
  String get videosUpTo => 'Videos up to 200 MB.';

  @override
  String get oneFileHint => '1 file.';

  @override
  String upToFilesHint(int n) {
    String _temp0 = intl.Intl.pluralLogic(n, locale: localeName, other: 'Up to $n files.', one: 'Up to $n file.');
    return '$_temp0';
  }

  @override
  String get takePhoto => 'Take a photo';

  @override
  String get recordVideo => 'Record a video';

  @override
  String get chooseFromPhotos => 'Choose from photos';

  @override
  String get chooseDocument => 'Choose a document';

  @override
  String get preparing => 'Preparing…';

  @override
  String uploadingPercent(int n) {
    return 'Uploading… $n%';
  }

  @override
  String get checkingFile => 'Checking the file…';

  @override
  String get added => 'Added';

  @override
  String get notUploaded => 'Not uploaded. Check your connection.';

  @override
  String get remove => 'Remove';

  @override
  String get removeFile => 'Remove file';

  @override
  String get removeFileTitle => 'Remove this file?';

  @override
  String get removeFileBody => 'It is deleted from your answers.';

  @override
  String get fileRejected => 'This file did not pass the safety check and was deleted. Choose another file.';

  @override
  String fileTooLarge(int max) {
    return 'This file is larger than $max MB.';
  }

  @override
  String get fileTypeRefused => 'This kind of file cannot be added to this question.';

  @override
  String tooManyFiles(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'You can add up to $n files to this question.',
      one: 'You can add up to $n file to this question.',
    );
    return '$_temp0';
  }

  @override
  String get filesFull => 'The files in this questionnaire have reached 500 MB. Remove a file to add another.';

  @override
  String get fileNotUploaded => 'The file could not be uploaded. Try again.';

  @override
  String get cannotPrepareFile => 'This file could not be prepared for upload. Choose another file.';

  @override
  String get filesBeingChecked => 'Some files are still being checked. You can submit once they are ready.';

  @override
  String get leaveUploading => 'A file is still uploading. Leave anyway? The upload stops.';

  @override
  String fileCount(int n) {
    String _temp0 = intl.Intl.pluralLogic(n, locale: localeName, other: '$n files', one: '1 file');
    return '$_temp0';
  }

  @override
  String get photoKind => 'Photo';

  @override
  String get documentKind => 'Document';

  @override
  String get videoKind => 'Video';

  @override
  String kilobytes(int n) {
    return '$n KB';
  }

  @override
  String megabytes(String mb) {
    return '$mb MB';
  }

  @override
  String kindAndSize(String kind, String size) {
    return '$kind, $size';
  }

  @override
  String uploadingLabel(String title, int n) {
    return 'Uploading $title, $n percent';
  }

  @override
  String fileLabel(String kind, String title, String status) {
    return '$kind, $title, $status';
  }

  @override
  String get allowCamera => 'Allow camera access';

  @override
  String get whyCameraForFiles => 'Salus needs the camera to take the photo or video you add to this questionnaire.';

  @override
  String get continueLabel => 'Continue';

  @override
  String get cameraDeniedForFiles =>
      'Salus does not have access to your camera. You can allow it in your device settings, or choose a file instead.';

  @override
  String get photosDenied => 'Salus does not have access to your photos. You can allow it in your device settings.';

  @override
  String cameraPhotoTitle(String date, String time) {
    return 'Photo, $date, $time';
  }

  @override
  String cameraVideoTitle(String date, String time) {
    return 'Video, $date, $time';
  }

  @override
  String get specialty394814009 => 'General practice';

  @override
  String get specialty419192003 => 'Internal medicine';

  @override
  String get specialty394579002 => 'Cardiology';

  @override
  String get specialty394582007 => 'Dermatology';

  @override
  String get specialty394586005 => 'Gynaecology';

  @override
  String get specialty394537008 => 'Paediatrics';

  @override
  String get specialty394587001 => 'Psychiatry';

  @override
  String get specialty394591006 => 'Neurology';

  @override
  String get specialty418960008 => 'ENT';

  @override
  String get specialty394594003 => 'Ophthalmology';

  @override
  String get specialty394801008 => 'Orthopaedics';

  @override
  String get specialty394583002 => 'Endocrinology';

  @override
  String questionnaireNotInLanguage(String language) {
    return 'This questionnaire is not available in $language.';
  }

  @override
  String questionnaireLanguages(String languages) {
    return 'It is available in: $languages. Choose the one you understand best.';
  }

  @override
  String continueInLanguage(String language) {
    return 'Continue in $language';
  }

  @override
  String get languageUnavailable => 'This language is no longer available for this questionnaire. Choose another one.';

  @override
  String specialtyAtClinic(String specialty, String clinic) {
    return '$specialty, $clinic';
  }

  @override
  String get documents => 'Documents';

  @override
  String get noDocuments =>
      'There are no signed documents for this appointment yet. A document appears here once your practitioner signs it.';

  @override
  String documentSignedOn(String date) {
    return 'Signed on $date';
  }

  @override
  String documentLanguage(String language) {
    return 'Language: $language';
  }

  @override
  String get sharePdf => 'Save or share the PDF';

  @override
  String get transcript => 'Transcript';

  @override
  String get transcriptNote =>
      'The consultation as it was transcribed and translated automatically. It may contain mistakes.';

  @override
  String get transcriptEmpty => 'Nothing was transcribed in this consultation.';

  @override
  String get speakerYou => 'You';

  @override
  String get speakerPractitioner => 'Practitioner';

  @override
  String get shareTranscript => 'Share the transcript';

  @override
  String transcriptTranslation(String text) {
    return 'Translation: $text';
  }
}
