// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for French (`fr`).
class AppLocalizationsFr extends AppLocalizations {
  AppLocalizationsFr([String locale = 'fr']) : super(locale);

  @override
  String get tryAgain => 'Réessayer';

  @override
  String get language => 'Langue';

  @override
  String get useDeviceLanguage => 'Utiliser la langue de l’appareil';

  @override
  String get languageSaved => 'Langue enregistrée.';

  @override
  String get somethingWrong => 'Une erreur s’est produite. Réessayez.';

  @override
  String get unreachable =>
      'Impossible de joindre Salus. Vérifiez votre connexion et réessayez.';

  @override
  String get wrongCredentials =>
      'L’adresse e-mail ou le mot de passe est incorrect.';

  @override
  String get wrongCode =>
      'Le code est incorrect. Vérifiez l’heure de votre appareil et réessayez.';

  @override
  String get challengeEnded =>
      'La connexion a expiré. Saisissez à nouveau votre adresse e-mail et votre mot de passe.';

  @override
  String get emailTaken =>
      'Un compte existe déjà avec cette adresse e-mail. Connectez-vous plutôt.';

  @override
  String get invalidEmail => 'Saisissez une adresse e-mail valide.';

  @override
  String get invalidPassword =>
      'Le mot de passe doit comporter entre 12 et 128 caractères.';

  @override
  String get missingPassword => 'Saisissez votre mot de passe.';

  @override
  String get invalidCode => 'Le code comporte 6 chiffres.';

  @override
  String get staffOnly =>
      'Cette application est destinée aux patients. Le personnel des cliniques utilise l’application web Salus.';

  @override
  String get accountReady =>
      'Votre compte est prêt. Connectez-vous pour continuer.';

  @override
  String get signOutFailed =>
      'Impossible de vous déconnecter. Vérifiez votre connexion et réessayez.';

  @override
  String get tooManyAttemptsLater => 'Trop de tentatives. Réessayez plus tard.';

  @override
  String tooManyAttempts(int minutes) {
    String _temp0 = intl.Intl.pluralLogic(
      minutes,
      locale: localeName,
      other: 'Trop de tentatives. Réessayez dans $minutes minutes.',
      one: 'Trop de tentatives. Réessayez dans $minutes minute.',
    );
    return '$_temp0';
  }

  @override
  String get signInTitle => 'Se connecter à Salus';

  @override
  String get email => 'Adresse e-mail';

  @override
  String get password => 'Mot de passe';

  @override
  String get signIn => 'Se connecter';

  @override
  String get createAnAccount => 'Créer un compte';

  @override
  String get codePrompt =>
      'Saisissez le code à 6 chiffres de votre application d’authentification.';

  @override
  String get code => 'Code';

  @override
  String get useAnotherAccount => 'Utiliser un autre compte';

  @override
  String get passwordHint => '12 à 128 caractères';

  @override
  String get createAccount => 'Créer le compte';

  @override
  String get haveAccount => 'J’ai déjà un compte';

  @override
  String signedInAs(String email) {
    return 'Compte connecté : $email';
  }

  @override
  String get completeProfile => 'Compléter votre profil';

  @override
  String get completeProfileFirst =>
      'Complétez votre profil avant de prendre rendez-vous.';

  @override
  String get bookAppointment => 'Prendre rendez-vous';

  @override
  String get myAppointments => 'Mes rendez-vous';

  @override
  String get myConsents => 'Mes consentements';

  @override
  String get myProfile => 'Mon profil';

  @override
  String get signOut => 'Se déconnecter';

  @override
  String get profileIntro =>
      'Les cliniques auprès desquelles vous prenez rendez-vous voient ces informations. Le numéro de téléphone est facultatif ; le praticien ne l’utilise que pour vous joindre au sujet d’un rendez-vous.';

  @override
  String get givenName => 'Prénom';

  @override
  String get familyName => 'Nom';

  @override
  String get dateOfBirth => 'Date de naissance';

  @override
  String get dateFormatHint => 'AAAA-MM-JJ';

  @override
  String get phoneOptional => 'Téléphone (facultatif)';

  @override
  String phoneHint(String example) {
    return 'Avec l’indicatif du pays, par exemple $example';
  }

  @override
  String get save => 'Enregistrer';

  @override
  String get notNow => 'Pas maintenant';

  @override
  String get invalidGivenName =>
      'Saisissez votre prénom (100 caractères maximum).';

  @override
  String get invalidFamilyName =>
      'Saisissez votre nom (100 caractères maximum).';

  @override
  String get invalidName =>
      'Saisissez votre prénom et votre nom (100 caractères maximum chacun).';

  @override
  String get birthDateFormat =>
      'Saisissez votre date de naissance au format AAAA-MM-JJ.';

  @override
  String get birthDateRange =>
      'Saisissez une date de naissance passée, à partir de 1900.';

  @override
  String invalidPhone(String example) {
    return 'Saisissez le numéro de téléphone avec l’indicatif du pays, par exemple $example.';
  }

  @override
  String get profileChanged =>
      'Votre profil a été modifié entre-temps. Vérifiez-le et enregistrez à nouveau.';

  @override
  String get chooseClinic => 'Choisir une clinique';

  @override
  String get noClinics => 'Aucune clinique n’est encore répertoriée.';

  @override
  String get noPractitioners =>
      'Aucun praticien n’est encore répertorié pour cette clinique.';

  @override
  String get choosePractitioner => 'Choisissez un praticien.';

  @override
  String get noOpenTimes =>
      'Ce praticien n’a pas encore de créneaux disponibles.';

  @override
  String timesIn(String zone) {
    return 'Fuseau horaire des créneaux : $zone.';
  }

  @override
  String get previousDay => 'Jour précédent';

  @override
  String get nextDay => 'Jour suivant';

  @override
  String get noFreeTimes => 'Aucun créneau disponible ce jour-là.';

  @override
  String get bookThisTime => 'Réserver ce créneau';

  @override
  String get chooseAnotherTime => 'Choisir un autre créneau';

  @override
  String get book => 'Réserver';

  @override
  String timeRange(String start, String end, String zone) {
    return 'De $start à $end ($zone)';
  }

  @override
  String get slotTaken =>
      'Ce créneau n’est plus disponible. Choisissez-en un autre.';

  @override
  String get consentNeeded =>
      'Pour prendre rendez-vous dans cette clinique, votre consentement est nécessaire.';

  @override
  String get appointmentBooked => 'Votre rendez-vous est réservé.';

  @override
  String get unlistedClinic => 'Clinique qui n’est plus répertoriée';

  @override
  String get cancelTitle => 'Annuler ce rendez-vous ?';

  @override
  String cancelBody(String date, String time) {
    return 'Votre rendez-vous du $date à $time sera annulé. Vous recevrez un e-mail de confirmation.';
  }

  @override
  String get keepIt => 'Le conserver';

  @override
  String get cancelAppointment => 'Annuler le rendez-vous';

  @override
  String get noAppointments => 'Vous n’avez pas encore de rendez-vous.';

  @override
  String get upcoming => 'À venir';

  @override
  String get past => 'Passés';

  @override
  String get statusBooked => 'Réservé';

  @override
  String get statusCancelled => 'Annulé';

  @override
  String get statusAttended => 'Honoré';

  @override
  String get statusNotAttended => 'Non honoré';

  @override
  String get appointmentCancelled => 'Votre rendez-vous est annulé.';

  @override
  String get cancelWindowHint =>
      'Annulable dans l’application jusqu’à 24 heures avant le début.';

  @override
  String get cancelWindowClosed =>
      'Ce rendez-vous commence dans moins de 24 heures et ne peut plus être annulé dans l’application.';

  @override
  String get appointmentClosed => 'Ce rendez-vous n’est plus réservé.';

  @override
  String consentTitle(String clinic) {
    return 'Partager vos informations avec $clinic';
  }

  @override
  String consentIntro(String clinic) {
    return 'Avant votre première réservation auprès de $clinic, nous avons besoin de votre consentement pour partager vos informations avec cette clinique.';
  }

  @override
  String get whatIsShared => 'Ce qui est partagé';

  @override
  String get whatIsSharedBody =>
      'Votre profil (vos nom et prénom, votre date de naissance et, si vous l’avez indiqué, votre numéro de téléphone) et vos rendez-vous dans cette clinique.';

  @override
  String get whoSeesIt => 'Qui y a accès';

  @override
  String whoSeesItBody(String clinic) {
    return '$clinic. Le praticien auprès duquel vous prenez rendez-vous voit votre profil ; les administrateurs de la clinique voient votre nom et les horaires de vos rendez-vous.';
  }

  @override
  String get whatFor => 'Dans quel but';

  @override
  String get whatForBody =>
      'Uniquement pour vos soins dans cette clinique : préparer et tenir vos rendez-vous.';

  @override
  String get whoIsResponsible => 'Qui est responsable';

  @override
  String whoIsResponsibleBody(String clinic) {
    return '$clinic est responsable de vos données. Salus les traite pour le compte de la clinique.';
  }

  @override
  String get readPrivacyNotice => 'Lire la politique de confidentialité';

  @override
  String get privacyNoticeFailed =>
      'Impossible d’ouvrir la politique de confidentialité.';

  @override
  String get howToRevoke => 'Comment retirer votre consentement';

  @override
  String get howToRevokeBody =>
      'Vous pouvez retirer ce consentement à tout moment dans Mes consentements. La clinique ne voit alors plus votre profil. Elle conserve votre nom et les détails des rendez-vous qui ont déjà eu lieu, dans ses dossiers de soins. Vos rendez-vous à venir dans cette clinique sont annulés.';

  @override
  String get agreeAndBook => 'Accepter et réserver';

  @override
  String get revokeTitle => 'Retirer votre consentement ?';

  @override
  String revokeBody(String clinic) {
    return '$clinic ne verra plus votre profil. La clinique conserve votre nom et les détails des rendez-vous qui ont déjà eu lieu, dans ses dossiers de soins. Vos rendez-vous à venir avec $clinic seront annulés.';
  }

  @override
  String get keepConsent => 'Conserver le consentement';

  @override
  String get revokeConsent => 'Retirer le consentement';

  @override
  String get noConsents => 'Vous n’avez encore donné aucun consentement.';

  @override
  String get consentsIntro =>
      'Vous partagez votre profil et vos rendez-vous avec ces cliniques pour vos soins.';

  @override
  String givenOn(String date) {
    return 'Donné le $date';
  }

  @override
  String get consentActive => 'Actif';

  @override
  String revokedOn(String date) {
    return 'Retiré le $date';
  }

  @override
  String get consentRevoked =>
      'Votre consentement est retiré. Vos rendez-vous à venir dans cette clinique seront annulés, et vous recevrez un e-mail pour chacun d’eux.';

  @override
  String get consentAlreadyRevoked => 'Ce consentement est déjà retiré.';

  @override
  String get joinConsultation => 'Rejoindre la consultation';

  @override
  String get joinFrom =>
      'Vous pouvez la rejoindre à partir de 10 minutes avant le début.';

  @override
  String get checkTitle => 'Vérifiez votre caméra et votre micro';

  @override
  String get checkIntro =>
      'Le praticien vous verra et vous entendra dès que vous aurez rejoint la consultation.';

  @override
  String get noSound =>
      'Aucun son détecté. Vérifiez que votre micro est activé et n’est pas coupé.';

  @override
  String get joinCameraOff => 'Rejoindre caméra désactivée';

  @override
  String get joinMicrophoneOff => 'Rejoindre micro désactivé';

  @override
  String get joinNow => 'Rejoindre';

  @override
  String get joining => 'Connexion…';

  @override
  String get cameraDenied =>
      'Salus n’a pas accès à votre caméra. Vous pouvez l’autoriser dans les réglages de votre appareil, ou rejoindre la consultation sans vidéo.';

  @override
  String get microphoneDenied =>
      'Salus n’a pas accès à votre micro. Vous pouvez l’autoriser dans les réglages de votre appareil. Sans micro, le praticien ne peut pas vous entendre.';

  @override
  String get openSettings => 'Ouvrir les réglages';

  @override
  String get noCamera => 'Aucune caméra trouvée.';

  @override
  String get noMicrophone => 'Aucun micro trouvé.';

  @override
  String get waiting => 'En attente du praticien';

  @override
  String joinsYouHere(String name) {
    return '$name vous rejoindra ici. Gardez cet écran ouvert.';
  }

  @override
  String get mute => 'Couper le micro';

  @override
  String get unmute => 'Réactiver le micro';

  @override
  String get cameraOff => 'Désactiver la caméra';

  @override
  String get cameraOn => 'Activer la caméra';

  @override
  String get leave => 'Quitter';

  @override
  String peerCameraOff(String name) {
    return 'La caméra de $name est désactivée';
  }

  @override
  String peerMuted(String name) {
    return 'Le micro de $name est coupé';
  }

  @override
  String peerLeft(String name) {
    return '$name a quitté la consultation et peut la rejoindre à nouveau tant qu’elle n’est pas terminée.';
  }

  @override
  String get reconnecting => 'Connexion perdue. Reconnexion…';

  @override
  String get notRestored => 'La connexion n’a pas pu être rétablie.';

  @override
  String get reconnected => 'Connexion rétablie';

  @override
  String get consultationOver => 'La consultation est terminée';

  @override
  String get closeScreen => 'Vous pouvez fermer cet écran.';

  @override
  String get timeLimit =>
      'La consultation a atteint sa durée maximale et s’est terminée.';

  @override
  String get practitionerAbsent =>
      'Le praticien n’a pas rejoint la consultation';

  @override
  String get timePassed =>
      'Le délai pour rejoindre cette consultation est dépassé. La clinique peut voir que vous l’avez rejointe. Elle pourra vous contacter, ou vous pouvez prendre un nouveau rendez-vous.';

  @override
  String get withdrewConsent =>
      'Vous avez retiré votre consentement pour cette clinique ; vous avez donc quitté la consultation.';

  @override
  String get appointmentWasCancelled =>
      'Ce rendez-vous a été annulé ; la consultation est donc terminée.';

  @override
  String get windowNotOpen =>
      'Cette consultation s’ouvre 10 minutes avant le début.';

  @override
  String get consultationEnded => 'Cette consultation est terminée.';

  @override
  String get joinNeedsConsent =>
      'Pour rejoindre la consultation, votre consentement pour cette clinique est nécessaire.';

  @override
  String get consultationNotAvailable =>
      'Cette consultation n’est pas disponible.';

  @override
  String get cannotConnect =>
      'La connexion vidéo n’a pas pu être établie. Vérifiez votre connexion et réessayez.';

  @override
  String get whyCameraAndMicrophone =>
      'Pour rejoindre la consultation vidéo, Salus a besoin de votre caméra et de votre micro. La consultation n’est enregistrée ni en audio ni en vidéo. Si vous l’acceptez, ce qui est dit est transcrit et traduit.';

  @override
  String get transcriptionOn => 'Transcription activée';

  @override
  String get transcriptionTitle => 'Transcription et traduction';

  @override
  String transcriptionIntro(String clinic) {
    return 'Lors de vos consultations vidéo avec $clinic, ce que vous et le praticien dites peut être transcrit au fil de la conversation et traduit, afin que le praticien puisse vous suivre dans votre langue.';
  }

  @override
  String get transcriptionAssistance =>
      'À partir de la transcription, le praticien reçoit aussi une aide de l’intelligence artificielle : des suggestions pendant la consultation et des brouillons de documents. Ce ne sont que des suggestions ; le praticien les vérifie et prend chaque décision.';

  @override
  String transcriptionKept(String clinic) {
    return 'La transcription est conservée avec votre dossier médical chez $clinic. La consultation n’est enregistrée ni en audio ni en vidéo.';
  }

  @override
  String get transcriptionDecline =>
      'Si vous refusez, la consultation se déroule normalement, sans transcription, traduction ni aide de l’intelligence artificielle.';

  @override
  String get transcriptionWithdraw =>
      'Vous pouvez retirer votre accord à tout moment, y compris pendant la consultation : touchez « Transcription activée » ou ouvrez Mes consentements. La transcription s’arrête aussitôt ; ce qui a été transcrit jusque-là reste dans votre dossier.';

  @override
  String get transcriptionAgree => 'Accepter et continuer';

  @override
  String get transcriptionWithout => 'Continuer sans transcription';

  @override
  String get stopTranscriptionTitle => 'Arrêter la transcription ?';

  @override
  String stopTranscriptionBody(String clinic) {
    return 'La transcription, la traduction et l’aide de l’intelligence artificielle s’arrêtent aussitôt dans vos consultations avec $clinic. Une consultation en cours se poursuit sans elles, et vos rendez-vous restent inchangés. Ce qui a été transcrit jusqu’ici reste dans votre dossier.';
  }

  @override
  String get keepTranscription => 'Garder la transcription';

  @override
  String get stopTranscription => 'Arrêter la transcription';

  @override
  String get transcriptionStopped =>
      'La transcription est désactivée. Vos consultations avec cette clinique se déroulent sans elle.';

  @override
  String get consentForCare => 'Partage pour les soins';

  @override
  String get consentForTranscription => 'Transcription et aide de l’IA';

  @override
  String get toFillIn => 'À remplir';

  @override
  String get inProgress => 'En cours';

  @override
  String get submittedStatus => 'Envoyé';

  @override
  String get closedStatus => 'Clôturé';

  @override
  String get fillIn => 'Remplir';

  @override
  String get continueFilling => 'Continuer';

  @override
  String get viewAnswers => 'Voir les réponses';

  @override
  String get submitBeforeStart =>
      'Merci de l’envoyer avant le début de votre rendez-vous.';

  @override
  String get questionnaireClosed =>
      'Le rendez-vous a commencé ; ce questionnaire ne peut donc plus être rempli.';

  @override
  String savedAsYouGo(String clinic) {
    return 'Vos réponses sont enregistrées au fur et à mesure. $clinic ne les voit qu’après leur envoi.';
  }

  @override
  String get saved => 'Enregistré';

  @override
  String get saving => 'Enregistrement…';

  @override
  String get notSaved =>
      'Non enregistré. Vérifiez votre connexion. L’enregistrement sera relancé.';

  @override
  String get requiredQuestion => 'Obligatoire';

  @override
  String get moreInformation => 'Plus d’informations';

  @override
  String get yourAnswer => 'Votre réponse';

  @override
  String get chooseDate => 'Choisir une date';

  @override
  String get selectOne => 'Choisissez une réponse.';

  @override
  String get selectAll => 'Choisissez toutes les réponses qui s’appliquent.';

  @override
  String get yes => 'Oui';

  @override
  String get no => 'Non';

  @override
  String get questionRequired => 'Cette question est obligatoire.';

  @override
  String get enterWholeNumber => 'Saisissez un nombre entier.';

  @override
  String get enterNumber => 'Saisissez un nombre.';

  @override
  String get numberTooLarge => 'Ce nombre est trop grand.';

  @override
  String upToCharacters(int max) {
    return 'Saisissez $max caractères au maximum.';
  }

  @override
  String get enterValidDate => 'Saisissez une date valide.';

  @override
  String get submit => 'Envoyer';

  @override
  String get submitTitle => 'Envoyer vos réponses ?';

  @override
  String submitBody(String clinic) {
    return 'Vous ne pourrez plus les modifier après l’envoi. Votre praticien chez $clinic les verra.';
  }

  @override
  String get cancel => 'Annuler';

  @override
  String get requiredMissing =>
      'Certaines questions obligatoires n’ont pas de réponse.';

  @override
  String get answersSubmitted => 'Réponses envoyées';

  @override
  String get answersSubmittedBody =>
      'Votre praticien peut les consulter avant votre rendez-vous. Vous pouvez les revoir depuis Mes rendez-vous.';

  @override
  String get yourAnswers => 'Vos réponses';

  @override
  String submittedAt(String when) {
    return 'Envoyé le $when';
  }

  @override
  String dateAndTime(String date, String time) {
    return '$date à $time';
  }

  @override
  String get notAnswered => 'Sans réponse';

  @override
  String get leaveUnsaved =>
      'Votre dernière réponse n’est pas encore enregistrée. Quitter quand même ?';

  @override
  String get stay => 'Rester';

  @override
  String get fillingNeedsConsent =>
      'Pour remplir ce questionnaire, votre consentement pour cette clinique est nécessaire.';

  @override
  String get pastDue =>
      'Le rendez-vous a commencé ; ce questionnaire ne peut donc plus être modifié.';

  @override
  String get withdrawnByClinic =>
      'La clinique a retiré ce questionnaire. Vous n’avez pas besoin de le remplir.';

  @override
  String get alreadySubmitted => 'Vous avez déjà envoyé ce questionnaire.';

  @override
  String get changedElsewhere =>
      'Ces réponses ont été modifiées sur un autre appareil. Les réponses les plus récentes sont affichées.';

  @override
  String get questionnaireNotAvailable =>
      'Ce questionnaire n’est pas disponible.';

  @override
  String get done => 'OK';

  @override
  String get addFile => 'Ajouter un fichier';

  @override
  String get photosUpTo => 'Photos jusqu’à 20 Mo.';

  @override
  String get documentsUpTo => 'Documents (PDF) jusqu’à 20 Mo.';

  @override
  String get videosUpTo => 'Vidéos jusqu’à 200 Mo.';

  @override
  String get oneFileHint => '1 fichier.';

  @override
  String upToFilesHint(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Jusqu’à $n fichiers.',
      one: 'Jusqu’à $n fichier.',
    );
    return '$_temp0';
  }

  @override
  String get takePhoto => 'Prendre une photo';

  @override
  String get recordVideo => 'Enregistrer une vidéo';

  @override
  String get chooseFromPhotos => 'Choisir dans les photos';

  @override
  String get chooseDocument => 'Choisir un document';

  @override
  String get preparing => 'Préparation…';

  @override
  String uploadingPercent(int n) {
    return 'Envoi… $n %';
  }

  @override
  String get checkingFile => 'Vérification du fichier…';

  @override
  String get added => 'Ajouté';

  @override
  String get notUploaded => 'Non envoyé. Vérifiez votre connexion.';

  @override
  String get remove => 'Retirer';

  @override
  String get removeFile => 'Retirer le fichier';

  @override
  String get removeFileTitle => 'Retirer ce fichier ?';

  @override
  String get removeFileBody => 'Il sera supprimé de vos réponses.';

  @override
  String get fileRejected =>
      'Ce fichier n’a pas passé le contrôle de sécurité et a été supprimé. Choisissez un autre fichier.';

  @override
  String fileTooLarge(int max) {
    return 'Ce fichier dépasse $max Mo.';
  }

  @override
  String get fileTypeRefused =>
      'Ce type de fichier ne peut pas être ajouté à cette question.';

  @override
  String tooManyFiles(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Vous pouvez ajouter $n fichiers au maximum à cette question.',
      one: 'Vous pouvez ajouter $n fichier au maximum à cette question.',
    );
    return '$_temp0';
  }

  @override
  String get filesFull =>
      'Les fichiers de ce questionnaire ont atteint 500 Mo. Retirez un fichier pour en ajouter un autre.';

  @override
  String get fileNotUploaded => 'Le fichier n’a pas pu être envoyé. Réessayez.';

  @override
  String get cannotPrepareFile =>
      'Ce fichier n’a pas pu être préparé pour l’envoi. Choisissez un autre fichier.';

  @override
  String get filesBeingChecked =>
      'Certains fichiers sont encore en cours de vérification. Vous pourrez envoyer vos réponses une fois qu’ils seront prêts.';

  @override
  String get leaveUploading =>
      'Un fichier est encore en cours d’envoi. Quitter quand même ? L’envoi sera interrompu.';

  @override
  String fileCount(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '$n fichiers',
      one: '$n fichier',
    );
    return '$_temp0';
  }

  @override
  String get photoKind => 'Photo';

  @override
  String get documentKind => 'Document';

  @override
  String get videoKind => 'Vidéo';

  @override
  String kilobytes(int n) {
    return '$n Ko';
  }

  @override
  String megabytes(String mb) {
    return '$mb Mo';
  }

  @override
  String kindAndSize(String kind, String size) {
    return '$kind, $size';
  }

  @override
  String uploadingLabel(String title, int n) {
    return 'Envoi de $title, $n pour cent';
  }

  @override
  String fileLabel(String kind, String title, String status) {
    return '$kind, $title, $status';
  }

  @override
  String get allowCamera => 'Autoriser l’accès à la caméra';

  @override
  String get whyCameraForFiles =>
      'Salus a besoin de la caméra pour prendre la photo ou la vidéo que vous ajoutez à ce questionnaire.';

  @override
  String get continueLabel => 'Continuer';

  @override
  String get cameraDeniedForFiles =>
      'Salus n’a pas accès à votre caméra. Vous pouvez l’autoriser dans les réglages de votre appareil, ou choisir plutôt un fichier.';

  @override
  String get photosDenied =>
      'Salus n’a pas accès à vos photos. Vous pouvez l’autoriser dans les réglages de votre appareil.';

  @override
  String cameraPhotoTitle(String date, String time) {
    return 'Photo, $date, $time';
  }

  @override
  String cameraVideoTitle(String date, String time) {
    return 'Vidéo, $date, $time';
  }

  @override
  String get specialty394814009 => 'Médecine générale';

  @override
  String get specialty419192003 => 'Médecine interne';

  @override
  String get specialty394579002 => 'Cardiologie';

  @override
  String get specialty394582007 => 'Dermatologie';

  @override
  String get specialty394586005 => 'Gynécologie';

  @override
  String get specialty394537008 => 'Pédiatrie';

  @override
  String get specialty394587001 => 'Psychiatrie';

  @override
  String get specialty394591006 => 'Neurologie';

  @override
  String get specialty418960008 => 'ORL';

  @override
  String get specialty394594003 => 'Ophtalmologie';

  @override
  String get specialty394801008 => 'Orthopédie';

  @override
  String get specialty394583002 => 'Endocrinologie';

  @override
  String questionnaireNotInLanguage(String language) {
    return 'Ce questionnaire n’est pas disponible en $language.';
  }

  @override
  String questionnaireLanguages(String languages) {
    return 'Il est disponible en : $languages. Choisissez la langue que vous comprenez le mieux.';
  }

  @override
  String continueInLanguage(String language) {
    return 'Continuer en $language';
  }

  @override
  String get languageUnavailable =>
      'Cette langue n’est plus disponible pour ce questionnaire. Choisissez-en une autre.';

  @override
  String specialtyAtClinic(String specialty, String clinic) {
    return '$specialty, $clinic';
  }

  @override
  String get documents => 'Documents';

  @override
  String get noDocuments =>
      'Il n’y a pas encore de document signé pour ce rendez-vous. Un document apparaît ici dès que votre praticien l’a signé.';

  @override
  String documentSignedOn(String date) {
    return 'Signé le $date';
  }

  @override
  String documentLanguage(String language) {
    return 'Langue : $language';
  }

  @override
  String get sharePdf => 'Enregistrer ou partager le PDF';

  @override
  String get transcript => 'Transcription';

  @override
  String get transcriptNote =>
      'La consultation telle qu’elle a été transcrite et traduite automatiquement. Elle peut contenir des erreurs.';

  @override
  String get transcriptEmpty =>
      'Rien n’a été transcrit pendant cette consultation.';

  @override
  String get speakerYou => 'Vous';

  @override
  String get speakerPractitioner => 'Praticien';

  @override
  String get shareTranscript => 'Partager la transcription';

  @override
  String transcriptTranslation(String text) {
    return 'Traduction : $text';
  }

  @override
  String get termExplanationNote => 'À titre informatif, pas un diagnostic';

  @override
  String get termNotReviewed => 'Pas encore vérifié';

  @override
  String get termClose => 'Fermer';

  @override
  String get sharingToggle =>
      'Partager mon historique d’autres cliniques avec cette clinique';

  @override
  String sharingIntro(String clinic) {
    return 'Cochez les visites précédentes que $clinic peut voir. Pour chaque visite cochée, $clinic voit ce qui est affiché ici, avec ses documents, ses réponses aux questionnaires et ses fichiers. Les visites que vous ne cochez pas restent cachées à $clinic.';
  }

  @override
  String get sharingChangeLater =>
      'Vous pouvez ajouter ou retirer des visites, ou arrêter le partage, à tout moment dans Historique partagé. Chaque modification prend effet immédiatement.';

  @override
  String get sharingNoVisits =>
      'Vous n’avez aucune visite précédente dans d’autres cliniques à partager.';

  @override
  String get sharingSelectAll => 'Tout cocher';

  @override
  String get sharingSelectNone => 'Tout décocher';

  @override
  String get sharedHistory => 'Historique partagé';

  @override
  String get sharedHistoryIntro =>
      'Les cliniques auxquelles vous avez donné votre consentement, et combien de vos visites dans d’autres cliniques chacune voit.';

  @override
  String get sharedHistoryNone =>
      'Vous n’avez encore donné votre consentement à aucune clinique.';

  @override
  String sharedVisits(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count visites partagées',
      one: '$count visite partagée',
      zero: 'Rien n’est partagé',
    );
    return '$_temp0';
  }

  @override
  String get sharingSaved =>
      'Votre choix est enregistré et prend effet immédiatement.';

  @override
  String get sharingStop => 'Arrêter le partage';

  @override
  String get sharingStopTitle => 'Arrêter le partage de votre historique ?';

  @override
  String sharingStopBody(String clinic) {
    return '$clinic ne verra plus aucune des visites d’autres cliniques que vous avez partagées avec elle. Cela prend effet immédiatement.';
  }

  @override
  String get sharingKeep => 'Continuer le partage';

  @override
  String get sharingStopped =>
      'Vous ne partagez plus votre historique avec cette clinique.';

  @override
  String get sharingChanged =>
      'Vos visites partagées ont changé entre-temps. Vérifiez-les et enregistrez à nouveau.';

  @override
  String get bookedNotShared =>
      'Votre rendez-vous est réservé, mais votre historique n’a pas pu être partagé. Vous pouvez le partager dans Historique partagé.';

  @override
  String get consentForHistory => 'Visites partagées d’autres cliniques';
}
