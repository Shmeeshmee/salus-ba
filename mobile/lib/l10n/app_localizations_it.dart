// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Italian (`it`).
class AppLocalizationsIt extends AppLocalizations {
  AppLocalizationsIt([String locale = 'it']) : super(locale);

  @override
  String get tryAgain => 'Riprova';

  @override
  String get language => 'Lingua';

  @override
  String get useDeviceLanguage => 'Usa la lingua del dispositivo';

  @override
  String get languageSaved => 'Lingua salvata.';

  @override
  String get somethingWrong => 'Si è verificato un errore. Riprovi.';

  @override
  String get unreachable =>
      'Impossibile raggiungere Salus. Controlli la connessione e riprovi.';

  @override
  String get wrongCredentials => 'L\'e-mail o la password non sono corrette.';

  @override
  String get wrongCode =>
      'Il codice non è corretto. Controlli l\'ora del dispositivo e riprovi.';

  @override
  String get challengeEnded =>
      'L\'accesso è scaduto. Inserisca di nuovo e-mail e password.';

  @override
  String get emailTaken =>
      'Esiste già un account con questa e-mail. Effettui l\'accesso.';

  @override
  String get invalidEmail => 'Inserisca un indirizzo e-mail valido.';

  @override
  String get invalidPassword => 'La password deve avere da 12 a 128 caratteri.';

  @override
  String get missingPassword => 'Inserisca la password.';

  @override
  String get invalidCode => 'Il codice è composto da 6 cifre.';

  @override
  String get staffOnly =>
      'Questa app è riservata ai pazienti. Il personale della clinica usa l\'app web di Salus.';

  @override
  String get accountReady => 'Il Suo account è pronto. Acceda per continuare.';

  @override
  String get signOutFailed =>
      'Impossibile uscire. Controlli la connessione e riprovi.';

  @override
  String get tooManyAttemptsLater => 'Troppi tentativi. Riprovi più tardi.';

  @override
  String tooManyAttempts(int minutes) {
    String _temp0 = intl.Intl.pluralLogic(
      minutes,
      locale: localeName,
      other: 'Troppi tentativi. Riprovi tra $minutes minuti.',
      one: 'Troppi tentativi. Riprovi tra $minutes minuto.',
    );
    return '$_temp0';
  }

  @override
  String get signInTitle => 'Accedi a Salus';

  @override
  String get email => 'E-mail';

  @override
  String get password => 'Password';

  @override
  String get signIn => 'Accedi';

  @override
  String get createAnAccount => 'Crea un account';

  @override
  String get codePrompt =>
      'Inserisca il codice a 6 cifre generato dall\'app di autenticazione.';

  @override
  String get code => 'Codice';

  @override
  String get useAnotherAccount => 'Usa un altro account';

  @override
  String get passwordHint => 'Da 12 a 128 caratteri';

  @override
  String get createAccount => 'Crea account';

  @override
  String get haveAccount => 'Ho già un account';

  @override
  String signedInAs(String email) {
    return 'Accesso effettuato come $email';
  }

  @override
  String get completeProfile => 'Completa il profilo';

  @override
  String get completeProfileFirst =>
      'Completi il Suo profilo prima di prenotare.';

  @override
  String get bookAppointment => 'Prenota un appuntamento';

  @override
  String get myAppointments => 'I miei appuntamenti';

  @override
  String get myConsents => 'I miei consensi';

  @override
  String get myProfile => 'Il mio profilo';

  @override
  String get signOut => 'Esci';

  @override
  String get profileIntro =>
      'Le cliniche presso cui prenota vedono questi dati. Il numero di telefono è facoltativo; il medico lo usa solo per contattarLa in merito a un appuntamento.';

  @override
  String get givenName => 'Nome';

  @override
  String get familyName => 'Cognome';

  @override
  String get dateOfBirth => 'Data di nascita';

  @override
  String get dateFormatHint => 'AAAA-MM-GG';

  @override
  String get phoneOptional => 'Telefono (facoltativo)';

  @override
  String phoneHint(String example) {
    return 'Con il prefisso internazionale, ad esempio $example';
  }

  @override
  String get save => 'Salva';

  @override
  String get notNow => 'Non ora';

  @override
  String get invalidGivenName => 'Inserisca il nome, al massimo 100 caratteri.';

  @override
  String get invalidFamilyName =>
      'Inserisca il cognome, al massimo 100 caratteri.';

  @override
  String get invalidName =>
      'Inserisca nome e cognome, al massimo 100 caratteri ciascuno.';

  @override
  String get birthDateFormat =>
      'Inserisca la data di nascita nel formato AAAA-MM-GG.';

  @override
  String get birthDateRange =>
      'Inserisca una data di nascita passata, non anteriore al 1900.';

  @override
  String invalidPhone(String example) {
    return 'Inserisca il numero di telefono con il prefisso internazionale, ad esempio $example.';
  }

  @override
  String get profileChanged =>
      'Il Suo profilo è stato modificato nel frattempo. Lo controlli e salvi di nuovo.';

  @override
  String get chooseClinic => 'Scegli una clinica';

  @override
  String get noClinics => 'Non ci sono ancora cliniche disponibili.';

  @override
  String get noPractitioners =>
      'Per questa clinica non sono ancora elencati medici.';

  @override
  String get choosePractitioner => 'Scelga un medico.';

  @override
  String get noOpenTimes => 'Questo medico non ha ancora orari disponibili.';

  @override
  String timesIn(String zone) {
    return 'Gli orari sono indicati nel fuso orario $zone.';
  }

  @override
  String get previousDay => 'Giorno precedente';

  @override
  String get nextDay => 'Giorno successivo';

  @override
  String get noFreeTimes => 'Nessun orario libero in questo giorno.';

  @override
  String get bookThisTime => 'Prenota questo orario';

  @override
  String get chooseAnotherTime => 'Scegli un altro orario';

  @override
  String get book => 'Prenota';

  @override
  String timeRange(String start, String end, String zone) {
    return '$start–$end ($zone)';
  }

  @override
  String get slotTaken => 'Questo orario non è più libero. Ne scelga un altro.';

  @override
  String get consentNeeded =>
      'Per prenotare presso questa clinica è necessario il Suo consenso.';

  @override
  String get appointmentBooked => 'L\'appuntamento è stato prenotato.';

  @override
  String get unlistedClinic => 'Una clinica non più in elenco';

  @override
  String get cancelTitle => 'Annullare questo appuntamento?';

  @override
  String cancelBody(String date, String time) {
    return 'L\'appuntamento previsto per il giorno $date alle ore $time sarà annullato. Riceverà un\'e-mail di conferma.';
  }

  @override
  String get keepIt => 'Mantieni';

  @override
  String get cancelAppointment => 'Annulla appuntamento';

  @override
  String get noAppointments => 'Non ha ancora appuntamenti.';

  @override
  String get upcoming => 'Prossimi';

  @override
  String get past => 'Passati';

  @override
  String get statusBooked => 'Prenotato';

  @override
  String get statusCancelled => 'Annullato';

  @override
  String get statusAttended => 'Effettuato';

  @override
  String get statusNotAttended => 'Non effettuato';

  @override
  String get appointmentCancelled => 'L\'appuntamento è stato annullato.';

  @override
  String get cancelWindowHint =>
      'Annullabile nell\'app fino a 24 ore prima dell\'inizio.';

  @override
  String get cancelWindowClosed =>
      'Questo appuntamento inizia tra meno di 24 ore e non può più essere annullato nell\'app.';

  @override
  String get appointmentClosed => 'Questo appuntamento non è più prenotato.';

  @override
  String consentTitle(String clinic) {
    return 'Condivisione dei Suoi dati con $clinic';
  }

  @override
  String consentIntro(String clinic) {
    return 'Prima della Sua prima prenotazione presso $clinic, ci serve il Suo consenso per condividere i Suoi dati con questa clinica.';
  }

  @override
  String get whatIsShared => 'Cosa viene condiviso';

  @override
  String get whatIsSharedBody =>
      'Il Suo profilo (nome, data di nascita e, se lo ha indicato, numero di telefono) e i Suoi appuntamenti presso questa clinica.';

  @override
  String get whoSeesIt => 'Chi vede i dati';

  @override
  String whoSeesItBody(String clinic) {
    return '$clinic. Il medico con cui prenota vede il Suo profilo; gli amministratori della clinica vedono il Suo nome e gli orari degli appuntamenti.';
  }

  @override
  String get whatFor => 'A quale scopo';

  @override
  String get whatForBody =>
      'Solo per le Sue cure presso questa clinica: per preparare e svolgere i Suoi appuntamenti.';

  @override
  String get whoIsResponsible => 'Chi è responsabile';

  @override
  String whoIsResponsibleBody(String clinic) {
    return '$clinic è titolare del trattamento dei Suoi dati. Salus li tratta per conto della clinica.';
  }

  @override
  String get readPrivacyNotice => 'Leggi l\'informativa sulla privacy';

  @override
  String get privacyNoticeFailed =>
      'Impossibile aprire l\'informativa sulla privacy.';

  @override
  String get howToRevoke => 'Come revocare';

  @override
  String get howToRevokeBody =>
      'Può revocare questo consenso in qualsiasi momento nella sezione «I miei consensi». La clinica non vedrà più il Suo profilo. Conserverà il Suo nome e i dati degli appuntamenti già avvenuti, come parte della propria documentazione sanitaria. I Suoi appuntamenti futuri presso la clinica saranno annullati.';

  @override
  String get agreeAndBook => 'Acconsenti e prenota';

  @override
  String get revokeTitle => 'Revocare il consenso?';

  @override
  String revokeBody(String clinic) {
    return '$clinic non vedrà più il Suo profilo. Conserverà il Suo nome e i dati degli appuntamenti già avvenuti, come parte della propria documentazione sanitaria. I Suoi appuntamenti futuri presso $clinic saranno annullati.';
  }

  @override
  String get keepConsent => 'Mantieni il consenso';

  @override
  String get revokeConsent => 'Revoca il consenso';

  @override
  String get noConsents => 'Non ha ancora dato alcun consenso.';

  @override
  String get consentsIntro =>
      'Condivide il Suo profilo e i Suoi appuntamenti con queste cliniche per le Sue cure.';

  @override
  String givenOn(String date) {
    return 'Dato in data $date';
  }

  @override
  String get consentActive => 'Attivo';

  @override
  String revokedOn(String date) {
    return 'Revocato in data $date';
  }

  @override
  String get consentRevoked =>
      'Il consenso è stato revocato. I Suoi appuntamenti futuri presso questa clinica saranno annullati e riceverà un\'e-mail per ciascuno.';

  @override
  String get consentAlreadyRevoked => 'Questo consenso è già stato revocato.';

  @override
  String get joinConsultation => 'Partecipa alla visita';

  @override
  String get joinFrom => 'Può partecipare da 10 minuti prima dell\'inizio.';

  @override
  String get checkTitle => 'Verifica fotocamera e microfono';

  @override
  String get checkIntro => 'Quando entra, il medico potrà vederLa e sentirLa.';

  @override
  String get noSound =>
      'Nessun suono rilevato. Controlli che il microfono sia acceso e non disattivato.';

  @override
  String get joinCameraOff => 'Partecipa con la fotocamera spenta';

  @override
  String get joinMicrophoneOff => 'Partecipa con il microfono spento';

  @override
  String get joinNow => 'Partecipa';

  @override
  String get joining => 'Accesso in corso…';

  @override
  String get cameraDenied =>
      'Salus non ha accesso alla fotocamera. Può consentirlo nelle impostazioni del dispositivo oppure partecipare senza video.';

  @override
  String get microphoneDenied =>
      'Salus non ha accesso al microfono. Può consentirlo nelle impostazioni del dispositivo. Senza microfono il medico non può sentirLa.';

  @override
  String get openSettings => 'Apri impostazioni';

  @override
  String get noCamera => 'Nessuna fotocamera trovata.';

  @override
  String get noMicrophone => 'Nessun microfono trovato.';

  @override
  String get waiting => 'In attesa del medico';

  @override
  String joinsYouHere(String name) {
    return '$name La raggiungerà qui. Tenga aperta questa schermata.';
  }

  @override
  String get mute => 'Disattiva il microfono';

  @override
  String get unmute => 'Attiva il microfono';

  @override
  String get cameraOff => 'Spegni la fotocamera';

  @override
  String get cameraOn => 'Accendi la fotocamera';

  @override
  String get leave => 'Esci';

  @override
  String peerCameraOff(String name) {
    return 'La fotocamera di $name è spenta';
  }

  @override
  String peerMuted(String name) {
    return 'Il microfono di $name è disattivato';
  }

  @override
  String peerLeft(String name) {
    return '$name ha lasciato la visita. Può rientrare fino al termine della visita.';
  }

  @override
  String get reconnecting => 'Connessione persa. Riconnessione in corso…';

  @override
  String get notRestored => 'Impossibile ripristinare la connessione.';

  @override
  String get reconnected => 'Connessione ripristinata';

  @override
  String get consultationOver => 'La visita è terminata';

  @override
  String get closeScreen => 'Può chiudere questa schermata.';

  @override
  String get timeLimit =>
      'La visita ha raggiunto il limite di tempo ed è terminata.';

  @override
  String get practitionerAbsent => 'Il medico non si è collegato';

  @override
  String get timePassed =>
      'Il tempo per partecipare a questa visita è scaduto. La clinica può vedere che Lei ha effettuato l\'accesso. La clinica potrebbe contattarLa, oppure può prenotare un nuovo appuntamento.';

  @override
  String get withdrewConsent =>
      'Ha revocato il consenso per questa clinica, quindi la Sua partecipazione alla visita è terminata.';

  @override
  String get appointmentWasCancelled =>
      'Questo appuntamento è stato annullato, quindi la visita è terminata.';

  @override
  String get windowNotOpen =>
      'Questa visita è accessibile da 10 minuti prima dell\'inizio.';

  @override
  String get consultationEnded => 'Questa visita è terminata.';

  @override
  String get joinNeedsConsent =>
      'Per partecipare è necessario il Suo consenso per questa clinica.';

  @override
  String get consultationNotAvailable => 'Questa visita non è disponibile.';

  @override
  String get cannotConnect =>
      'Impossibile avviare la connessione video. Controlli la connessione e riprovi.';

  @override
  String get whyCameraAndMicrophone =>
      'Per partecipare alla videovisita, Salus ha bisogno di accedere alla fotocamera e al microfono. La visita non viene registrata né in audio né in video. Se acconsente, ciò che viene detto viene trascritto e tradotto.';

  @override
  String get transcriptionOn => 'Trascrizione attiva';

  @override
  String get transcriptionTitle => 'Trascrizione e traduzione';

  @override
  String transcriptionIntro(String clinic) {
    return 'Nelle Sue videovisite con $clinic, ciò che Lei e il medico dite può essere trascritto mentre parlate e tradotto, così il medico può seguirLa nella Sua lingua.';
  }

  @override
  String get transcriptionAssistance =>
      'Dalla trascrizione il medico riceve anche un supporto dell\'intelligenza artificiale: suggerimenti durante la visita e bozze di documenti. Sono solo suggerimenti; il medico li verifica e prende ogni decisione.';

  @override
  String transcriptionKept(String clinic) {
    return 'La trascrizione viene conservata con la Sua documentazione sanitaria presso $clinic. La visita non viene registrata né in audio né in video.';
  }

  @override
  String get transcriptionDecline =>
      'Se non acconsente, la visita si svolge come di consueto, senza trascrizione, traduzione né supporto dell\'intelligenza artificiale.';

  @override
  String get transcriptionWithdraw =>
      'Può revocare il consenso in qualsiasi momento, anche durante la visita: tocchi «Trascrizione attiva» oppure apra I miei consensi. La trascrizione si interrompe subito; ciò che è stato trascritto fino a quel momento resta nella Sua documentazione.';

  @override
  String get transcriptionAgree => 'Acconsenti e continua';

  @override
  String get transcriptionWithout => 'Continua senza trascrizione';

  @override
  String get stopTranscriptionTitle => 'Interrompere la trascrizione?';

  @override
  String stopTranscriptionBody(String clinic) {
    return 'Trascrizione, traduzione e supporto dell\'intelligenza artificiale si interrompono subito nelle Sue visite con $clinic. Una visita in corso prosegue senza di essi e i Suoi appuntamenti restano invariati. Ciò che è stato trascritto finora resta nella Sua documentazione.';
  }

  @override
  String get keepTranscription => 'Mantieni la trascrizione';

  @override
  String get stopTranscription => 'Interrompi la trascrizione';

  @override
  String get transcriptionStopped =>
      'La trascrizione è disattivata. Le Sue visite con questa clinica si svolgono senza.';

  @override
  String get consentForCare => 'Condivisione per le cure';

  @override
  String get consentForTranscription =>
      'Trascrizione e supporto dell\'intelligenza artificiale';

  @override
  String get toFillIn => 'Da compilare';

  @override
  String get inProgress => 'In corso';

  @override
  String get submittedStatus => 'Inviato';

  @override
  String get closedStatus => 'Chiuso';

  @override
  String get fillIn => 'Compila';

  @override
  String get continueFilling => 'Continua';

  @override
  String get viewAnswers => 'Vedi le risposte';

  @override
  String get submitBeforeStart =>
      'Lo invii prima dell\'inizio dell\'appuntamento.';

  @override
  String get questionnaireClosed =>
      'L\'appuntamento è iniziato, quindi questo questionario non può più essere compilato.';

  @override
  String savedAsYouGo(String clinic) {
    return 'Le risposte vengono salvate man mano. $clinic le vede solo dopo l\'invio.';
  }

  @override
  String get saved => 'Salvato';

  @override
  String get saving => 'Salvataggio…';

  @override
  String get notSaved =>
      'Non salvato. Controlli la connessione. Il salvataggio verrà ritentato.';

  @override
  String get requiredQuestion => 'Obbligatoria';

  @override
  String get moreInformation => 'Maggiori informazioni';

  @override
  String get yourAnswer => 'La Sua risposta';

  @override
  String get chooseDate => 'Scegli una data';

  @override
  String get selectOne => 'Selezioni un\'opzione.';

  @override
  String get selectAll => 'Selezioni tutte le opzioni pertinenti.';

  @override
  String get yes => 'Sì';

  @override
  String get no => 'No';

  @override
  String get questionRequired => 'Questa domanda è obbligatoria.';

  @override
  String get enterWholeNumber => 'Inserisca un numero intero.';

  @override
  String get enterNumber => 'Inserisca un numero.';

  @override
  String get numberTooLarge => 'Questo numero è troppo grande.';

  @override
  String upToCharacters(int max) {
    return 'Inserisca al massimo $max caratteri.';
  }

  @override
  String get enterValidDate => 'Inserisca una data valida.';

  @override
  String get submit => 'Invia';

  @override
  String get submitTitle => 'Inviare le risposte?';

  @override
  String submitBody(String clinic) {
    return 'Dopo l\'invio non potrà più modificarle. Il Suo medico presso $clinic le vedrà.';
  }

  @override
  String get cancel => 'Annulla';

  @override
  String get requiredMissing =>
      'Alcune domande obbligatorie non hanno risposta.';

  @override
  String get answersSubmitted => 'Risposte inviate';

  @override
  String get answersSubmittedBody =>
      'Il Suo medico può vederle prima dell\'appuntamento. Può consultarle da «I miei appuntamenti».';

  @override
  String get yourAnswers => 'Le Sue risposte';

  @override
  String submittedAt(String when) {
    return 'Inviato in data $when';
  }

  @override
  String dateAndTime(String date, String time) {
    return '$date, $time';
  }

  @override
  String get notAnswered => 'Nessuna risposta';

  @override
  String get leaveUnsaved =>
      'L\'ultima risposta non è ancora salvata. Uscire comunque?';

  @override
  String get stay => 'Resta';

  @override
  String get fillingNeedsConsent =>
      'Per compilare questo questionario è necessario il Suo consenso per questa clinica.';

  @override
  String get pastDue =>
      'L\'appuntamento è iniziato, quindi questo questionario non può più essere modificato.';

  @override
  String get withdrawnByClinic =>
      'La clinica ha ritirato questo questionario. Non è necessario compilarlo.';

  @override
  String get alreadySubmitted => 'Ha già inviato questo questionario.';

  @override
  String get changedElsewhere =>
      'Queste risposte sono state modificate su un altro dispositivo. Vengono mostrate le risposte più recenti.';

  @override
  String get questionnaireNotAvailable =>
      'Questo questionario non è disponibile.';

  @override
  String get done => 'Fine';

  @override
  String get addFile => 'Aggiungi file';

  @override
  String get photosUpTo => 'Foto fino a 20 MB.';

  @override
  String get documentsUpTo => 'Documenti (PDF) fino a 20 MB.';

  @override
  String get videosUpTo => 'Video fino a 200 MB.';

  @override
  String get oneFileHint => '1 file.';

  @override
  String upToFilesHint(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Fino a $n file.',
      one: 'Fino a $n file.',
    );
    return '$_temp0';
  }

  @override
  String get takePhoto => 'Scatta una foto';

  @override
  String get recordVideo => 'Registra un video';

  @override
  String get chooseFromPhotos => 'Scegli dalle foto';

  @override
  String get chooseDocument => 'Scegli un documento';

  @override
  String get preparing => 'Preparazione…';

  @override
  String uploadingPercent(int n) {
    return 'Caricamento… $n%';
  }

  @override
  String get checkingFile => 'Verifica del file…';

  @override
  String get added => 'Aggiunto';

  @override
  String get notUploaded => 'Non caricato. Controlli la connessione.';

  @override
  String get remove => 'Rimuovi';

  @override
  String get removeFile => 'Rimuovi file';

  @override
  String get removeFileTitle => 'Rimuovere questo file?';

  @override
  String get removeFileBody => 'Verrà eliminato dalle Sue risposte.';

  @override
  String get fileRejected =>
      'Questo file non ha superato il controllo di sicurezza ed è stato eliminato. Scelga un altro file.';

  @override
  String fileTooLarge(int max) {
    return 'Questo file supera i $max MB.';
  }

  @override
  String get fileTypeRefused =>
      'Questo tipo di file non può essere aggiunto a questa domanda.';

  @override
  String tooManyFiles(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Può aggiungere al massimo $n file a questa domanda.',
      one: 'Può aggiungere al massimo $n file a questa domanda.',
    );
    return '$_temp0';
  }

  @override
  String get filesFull =>
      'I file di questo questionario hanno raggiunto i 500 MB. Rimuova un file per aggiungerne un altro.';

  @override
  String get fileNotUploaded => 'Impossibile caricare il file. Riprovi.';

  @override
  String get cannotPrepareFile =>
      'Impossibile preparare questo file per il caricamento. Scelga un altro file.';

  @override
  String get filesBeingChecked =>
      'Alcuni file sono ancora in verifica. Potrà inviare quando saranno pronti.';

  @override
  String get leaveUploading =>
      'Un file è ancora in caricamento. Uscire comunque? Il caricamento verrà interrotto.';

  @override
  String fileCount(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '$n file',
      one: '$n file',
    );
    return '$_temp0';
  }

  @override
  String get photoKind => 'Foto';

  @override
  String get documentKind => 'Documento';

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
    return 'Caricamento di $title, $n per cento';
  }

  @override
  String fileLabel(String kind, String title, String status) {
    return '$kind, $title, $status';
  }

  @override
  String get allowCamera => 'Consenti l\'accesso alla fotocamera';

  @override
  String get whyCameraForFiles =>
      'Salus ha bisogno della fotocamera per scattare la foto o registrare il video da aggiungere a questo questionario.';

  @override
  String get continueLabel => 'Continua';

  @override
  String get cameraDeniedForFiles =>
      'Salus non ha accesso alla fotocamera. Può consentirlo nelle impostazioni del dispositivo oppure scegliere un file.';

  @override
  String get photosDenied =>
      'Salus non ha accesso alle Sue foto. Può consentirlo nelle impostazioni del dispositivo.';

  @override
  String cameraPhotoTitle(String date, String time) {
    return 'Foto, $date, $time';
  }

  @override
  String cameraVideoTitle(String date, String time) {
    return 'Video, $date, $time';
  }

  @override
  String get specialty394814009 => 'Medicina generale';

  @override
  String get specialty419192003 => 'Medicina interna';

  @override
  String get specialty394579002 => 'Cardiologia';

  @override
  String get specialty394582007 => 'Dermatologia';

  @override
  String get specialty394586005 => 'Ginecologia';

  @override
  String get specialty394537008 => 'Pediatria';

  @override
  String get specialty394587001 => 'Psichiatria';

  @override
  String get specialty394591006 => 'Neurologia';

  @override
  String get specialty418960008 => 'Otorinolaringoiatria';

  @override
  String get specialty394594003 => 'Oftalmologia';

  @override
  String get specialty394801008 => 'Ortopedia';

  @override
  String get specialty394583002 => 'Endocrinologia';

  @override
  String questionnaireNotInLanguage(String language) {
    return 'Questo questionario non è disponibile in $language.';
  }

  @override
  String questionnaireLanguages(String languages) {
    return 'È disponibile in: $languages. Scelga la lingua che comprende meglio.';
  }

  @override
  String continueInLanguage(String language) {
    return 'Continua in $language';
  }

  @override
  String get languageUnavailable =>
      'Questa lingua non è più disponibile per questo questionario. Ne scelga un’altra.';

  @override
  String specialtyAtClinic(String specialty, String clinic) {
    return '$specialty, $clinic';
  }

  @override
  String get documents => 'Documenti';

  @override
  String get noDocuments =>
      'Per questo appuntamento non ci sono ancora documenti firmati. Un documento compare qui quando il medico lo firma.';

  @override
  String documentSignedOn(String date) {
    return 'Firmato il $date';
  }

  @override
  String documentLanguage(String language) {
    return 'Lingua: $language';
  }

  @override
  String get sharePdf => 'Salva o condividi il PDF';

  @override
  String get transcript => 'Trascrizione';

  @override
  String get transcriptNote =>
      'La visita come è stata trascritta e tradotta automaticamente. Può contenere errori.';

  @override
  String get transcriptEmpty =>
      'In questa visita non è stato trascritto nulla.';

  @override
  String get speakerYou => 'Lei';

  @override
  String get speakerPractitioner => 'Medico';

  @override
  String get shareTranscript => 'Condividi la trascrizione';

  @override
  String transcriptTranslation(String text) {
    return 'Traduzione: $text';
  }

  @override
  String get termExplanationNote => 'A titolo informativo, non è una diagnosi';

  @override
  String get termNotReviewed => 'Non ancora verificato';

  @override
  String get termClose => 'Chiudi';

  @override
  String get chatsTitle => 'Messaggi';

  @override
  String get chatsEmpty => 'Non ha ancora conversazioni.';

  @override
  String chatUnread(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count messaggi non letti',
      one: '$count messaggio non letto',
    );
    return '$_temp0';
  }

  @override
  String get chatDoctorUnknown => 'Il Suo medico';

  @override
  String get chatMessageHint => 'Scriva un messaggio';

  @override
  String get chatSend => 'Invia';

  @override
  String get chatAttach => 'Aggiungi un file';

  @override
  String get chatSendFailed =>
      'Il messaggio non è stato inviato. Controlli la connessione e riprovi.';

  @override
  String get chatFileRejected => 'Questo file non è stato accettato.';

  @override
  String get chatFileRemove => 'Rimuovi il file';

  @override
  String chatTooManyFiles(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other:
          'Ha già $count file in attesa di invio. I file non inviati vengono rimossi dopo 24 ore; riprovi più tardi.',
      one:
          'Ha già $count file in attesa di invio. I file non inviati vengono rimossi dopo 24 ore; riprovi più tardi.',
    );
    return '$_temp0';
  }

  @override
  String get chatOpenFailed => 'Non è stato possibile aprire il file. Riprovi.';

  @override
  String get chatEndedSession =>
      'La sessione è terminata. Acceda di nuovo per vedere i nuovi messaggi.';

  @override
  String get chatEnded =>
      'In questa conversazione non è più possibile mostrare nuovi messaggi.';

  @override
  String get chatHours => 'Orari dei messaggi';

  @override
  String chatHoursDay(String day, String start, String end) {
    return '$day: $start–$end';
  }

  @override
  String get chatHoursNone =>
      'La clinica non ha stabilito orari per i messaggi.';

  @override
  String chatHoursZone(String zone) {
    return 'Gli orari sono nel fuso orario della clinica, $zone.';
  }

  @override
  String chatResponseTime(int hours) {
    String _temp0 = intl.Intl.pluralLogic(
      hours,
      locale: localeName,
      other: 'Di solito la clinica risponde entro $hours ore.',
      one: 'Di solito la clinica risponde entro $hours ora.',
    );
    return '$_temp0';
  }

  @override
  String chatOutOfHours(int hours) {
    String _temp0 = intl.Intl.pluralLogic(
      hours,
      locale: localeName,
      other:
          'La clinica è fuori dagli orari dei messaggi. Può comunque scrivere; di solito risponde entro $hours ore.',
      one:
          'La clinica è fuori dagli orari dei messaggi. Può comunque scrivere; di solito risponde entro $hours ora.',
    );
    return '$_temp0';
  }

  @override
  String chatSystemInvited(String reason) {
    String _temp0 = intl.Intl.selectLogic(reason, {
      'second_opinion': 'Un medico di un’altra clinica è stato invitato a questa conversazione per un secondo parere.',
      'specialist_review': 'Un medico di un’altra clinica è stato invitato a questa conversazione per una valutazione specialistica.',
      'handover': 'Un medico di un’altra clinica è stato invitato a questa conversazione per prendere in carico le cure.',
      'other': 'Un medico di un’altra clinica è stato invitato a questa conversazione.',
    });
    return '$_temp0';
  }

  @override
  String chatSystemAdded(String reason) {
    String _temp0 = intl.Intl.selectLogic(reason, {
      'second_opinion':
          'Un medico si è unito alla conversazione per un secondo parere.',
      'specialist_review': 'Un medico si è unito alla conversazione per una valutazione specialistica.',
      'handover': 'Un medico si è unito alla conversazione per prendere in carico le cure.',
      'other': 'Un medico si è unito alla conversazione.',
    });
    return '$_temp0';
  }

  @override
  String get chatSystemRemoved =>
      'Un medico non partecipa più alla conversazione.';

  @override
  String get chatSystemGranted =>
      'Ha consentito a un medico di un’altra clinica di unirsi alla conversazione.';

  @override
  String get chatSystemDeclined =>
      'Ha rifiutato un medico di un’altra clinica.';

  @override
  String get chatSystemOther => 'La conversazione è stata aggiornata.';

  @override
  String get chatInviteTitle =>
      'Un medico di un’altra clinica chiede di partecipare';

  @override
  String chatInviteBody(String reason) {
    String _temp0 = intl.Intl.selectLogic(reason, {
      'second_opinion': 'La Sua clinica chiede che un medico di un’altra clinica si unisca a questa conversazione per un secondo parere.',
      'specialist_review': 'La Sua clinica chiede che un medico di un’altra clinica si unisca a questa conversazione per una valutazione specialistica.',
      'handover': 'La Sua clinica chiede che un medico di un’altra clinica si unisca a questa conversazione per prendere in carico le Sue cure.',
      'other': 'La Sua clinica chiede che un medico di un’altra clinica si unisca a questa conversazione.',
    });
    return '$_temp0';
  }

  @override
  String chatInviteWho(String doctor, String clinic) {
    return 'Medico: $doctor, $clinic';
  }

  @override
  String get chatInviteUnresolved =>
      'Non è stato possibile caricare il nome e la clinica del medico invitato. Potrà concedere l’accesso solo quando saranno visualizzati.';

  @override
  String chatInviteSees(String history) {
    String _temp0 = intl.Intl.selectLogic(history, {
      'full': 'Se acconsente, il medico vede l’intera conversazione, compresi i messaggi precedenti, e può scrivervi.',
      'from': 'Se acconsente, il medico vede la conversazione a partire da un messaggio scelto dalla Sua clinica e può scrivervi.',
      'other':
          'Se acconsente, il medico vede la conversazione e può scrivervi.',
    });
    return '$_temp0';
  }

  @override
  String get chatInviteChoice =>
      'Può rifiutare. Se acconsente, può revocare l’accesso del medico in seguito in questa conversazione; termina subito.';

  @override
  String get chatGrant => 'Concedi l’accesso';

  @override
  String get chatDecline => 'Rifiuta';

  @override
  String get chatGranted =>
      'Accesso concesso. Può revocarlo in qualsiasi momento in questa conversazione.';

  @override
  String get chatDeclined =>
      'Rifiutato. Il medico non si unisce a questa conversazione.';

  @override
  String get chatRevoke => 'Revoca l’accesso del medico dell’altra clinica';

  @override
  String chatGrantedTo(String doctor, String clinic) {
    return 'Accesso concesso: $doctor, $clinic.';
  }

  @override
  String get chatGrantedToUnknown =>
      'Accesso concesso a un medico di un’altra clinica.';

  @override
  String get chatRevokeBody =>
      'Il medico dell’altra clinica perde subito l’accesso a questa conversazione. I messaggi già scritti vi restano.';

  @override
  String get chatRevokeConfirm => 'Revoca l’accesso';

  @override
  String get chatRevoked => 'Accesso revocato.';

  @override
  String get sharingToggle =>
      'Condividi la mia storia clinica di altre cliniche con questa clinica';

  @override
  String sharingIntro(String clinic) {
    return 'Selezioni le visite precedenti che $clinic può vedere. Per ogni visita selezionata, $clinic vede ciò che è mostrato qui, con i relativi documenti, risposte ai questionari e file. Le visite che non seleziona restano nascoste a $clinic.';
  }

  @override
  String get sharingChangeLater =>
      'Può aggiungere o togliere visite, o interrompere la condivisione, in qualsiasi momento nella sezione «Storia condivisa». Ogni modifica vale subito.';

  @override
  String get sharingNoVisits =>
      'Non ha visite precedenti presso altre cliniche da condividere.';

  @override
  String get sharingSelectAll => 'Seleziona tutte';

  @override
  String get sharingSelectNone => 'Deseleziona tutte';

  @override
  String get sharedHistory => 'Storia condivisa';

  @override
  String get sharedHistoryIntro =>
      'Le cliniche a cui ha dato il consenso e quante delle Sue visite presso altre cliniche vede ciascuna.';

  @override
  String get sharedHistoryNone =>
      'Non ha ancora dato il consenso a nessuna clinica.';

  @override
  String sharedVisits(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count visite condivise',
      one: '$count visita condivisa',
      zero: 'Nulla di condiviso',
    );
    return '$_temp0';
  }

  @override
  String get sharingSaved => 'La Sua scelta è salvata e vale subito.';

  @override
  String get sharingStop => 'Interrompi la condivisione';

  @override
  String get sharingStopTitle =>
      'Interrompere la condivisione della Sua storia clinica?';

  @override
  String sharingStopBody(String clinic) {
    return '$clinic non vedrà più nessuna delle visite presso altre cliniche che ha condiviso con essa. Vale subito.';
  }

  @override
  String get sharingKeep => 'Continua a condividere';

  @override
  String get sharingStopped =>
      'Non condivide più la Sua storia clinica con questa clinica.';

  @override
  String get sharingChanged =>
      'Le Sue visite condivise sono cambiate nel frattempo. Le controlli e salvi di nuovo.';

  @override
  String get bookedNotShared =>
      'L\'appuntamento è prenotato, ma non è stato possibile condividere la Sua storia clinica. Può condividerla nella sezione «Storia condivisa».';

  @override
  String get consentForHistory => 'Visite condivise da altre cliniche';
}
