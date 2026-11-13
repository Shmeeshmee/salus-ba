// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Bosnian (`bs`).
class AppLocalizationsBs extends AppLocalizations {
  AppLocalizationsBs([String locale = 'bs']) : super(locale);

  @override
  String get tryAgain => 'Pokušajte ponovo';

  @override
  String get language => 'Jezik';

  @override
  String get useDeviceLanguage => 'Koristi jezik uređaja';

  @override
  String get languageSaved => 'Jezik je sačuvan.';

  @override
  String get somethingWrong => 'Nešto nije u redu. Pokušajte ponovo.';

  @override
  String get unreachable =>
      'Salus trenutno nije dostupan. Provjerite internet vezu i pokušajte ponovo.';

  @override
  String get wrongCredentials => 'E-mail ili lozinka nisu tačni.';

  @override
  String get wrongCode =>
      'Kod nije tačan. Provjerite vrijeme na uređaju i pokušajte ponovo.';

  @override
  String get challengeEnded =>
      'Prijava je istekla. Ponovo unesite e-mail i lozinku.';

  @override
  String get emailTaken => 'Za ovaj e-mail već postoji račun. Prijavite se.';

  @override
  String get invalidEmail => 'Unesite ispravnu e-mail adresu.';

  @override
  String get invalidPassword => 'Lozinka mora imati od 12 do 128 znakova.';

  @override
  String get missingPassword => 'Unesite lozinku.';

  @override
  String get invalidCode => 'Kod ima 6 cifara.';

  @override
  String get staffOnly =>
      'Ova aplikacija je namijenjena pacijentima. Osoblje klinike koristi Salus web aplikaciju.';

  @override
  String get accountReady =>
      'Vaš račun je spreman. Prijavite se da biste nastavili.';

  @override
  String get signOutFailed =>
      'Odjava nije uspjela. Provjerite internet vezu i pokušajte ponovo.';

  @override
  String get tooManyAttemptsLater =>
      'Previše pokušaja. Pokušajte ponovo kasnije.';

  @override
  String tooManyAttempts(int minutes) {
    String _temp0 = intl.Intl.pluralLogic(
      minutes,
      locale: localeName,
      other: 'Previše pokušaja. Pokušajte ponovo za $minutes minuta.',
      few: 'Previše pokušaja. Pokušajte ponovo za $minutes minute.',
      one: 'Previše pokušaja. Pokušajte ponovo za $minutes minutu.',
    );
    return '$_temp0';
  }

  @override
  String get signInTitle => 'Prijava u Salus';

  @override
  String get email => 'E-mail';

  @override
  String get password => 'Lozinka';

  @override
  String get signIn => 'Prijavi se';

  @override
  String get createAnAccount => 'Kreirajte račun';

  @override
  String get codePrompt =>
      'Unesite šestocifreni kod iz aplikacije za autentifikaciju.';

  @override
  String get code => 'Kod';

  @override
  String get useAnotherAccount => 'Koristi drugi račun';

  @override
  String get passwordHint => 'Od 12 do 128 znakova';

  @override
  String get createAccount => 'Kreiraj račun';

  @override
  String get haveAccount => 'Već imam račun';

  @override
  String signedInAs(String email) {
    return 'Prijavljeni ste kao $email';
  }

  @override
  String get completeProfile => 'Dopunite profil';

  @override
  String get completeProfileFirst =>
      'Dopunite profil prije zakazivanja termina.';

  @override
  String get bookAppointment => 'Zakaži termin';

  @override
  String get myAppointments => 'Moji termini';

  @override
  String get myConsents => 'Moje saglasnosti';

  @override
  String get myProfile => 'Moj profil';

  @override
  String get signOut => 'Odjavi se';

  @override
  String get profileIntro =>
      'Klinike kod kojih zakažete termin vide ove podatke. Broj telefona nije obavezan; ljekar ga koristi samo da bi vas kontaktirao u vezi s terminom.';

  @override
  String get givenName => 'Ime';

  @override
  String get familyName => 'Prezime';

  @override
  String get dateOfBirth => 'Datum rođenja';

  @override
  String get dateFormatHint => 'GGGG-MM-DD';

  @override
  String get phoneOptional => 'Telefon (nije obavezno)';

  @override
  String phoneHint(String example) {
    return 'S pozivnim brojem države, na primjer $example';
  }

  @override
  String get save => 'Sačuvaj';

  @override
  String get notNow => 'Ne sada';

  @override
  String get invalidGivenName => 'Unesite ime, najviše 100 znakova.';

  @override
  String get invalidFamilyName => 'Unesite prezime, najviše 100 znakova.';

  @override
  String get invalidName => 'Unesite ime i prezime, svako najviše 100 znakova.';

  @override
  String get birthDateFormat => 'Unesite datum rođenja u obliku GGGG-MM-DD.';

  @override
  String get birthDateRange =>
      'Unesite datum rođenja u prošlosti, ne raniji od 1900. godine.';

  @override
  String invalidPhone(String example) {
    return 'Unesite broj telefona s pozivnim brojem države, na primjer $example.';
  }

  @override
  String get profileChanged =>
      'Vaš profil je u međuvremenu izmijenjen. Provjerite ga i ponovo sačuvajte.';

  @override
  String get chooseClinic => 'Odaberite kliniku';

  @override
  String get noClinics => 'Još nema dostupnih klinika.';

  @override
  String get noPractitioners => 'Ova klinika još nema dostupnih ljekara.';

  @override
  String get choosePractitioner => 'Odaberite ljekara.';

  @override
  String get noOpenTimes => 'Ovaj ljekar još nema slobodnih termina.';

  @override
  String timesIn(String zone) {
    return 'Vrijeme je prikazano u vremenskoj zoni $zone.';
  }

  @override
  String get previousDay => 'Prethodni dan';

  @override
  String get nextDay => 'Sljedeći dan';

  @override
  String get noFreeTimes => 'Za ovaj dan nema slobodnih termina.';

  @override
  String get bookThisTime => 'Zakazivanje termina';

  @override
  String get chooseAnotherTime => 'Odaberite drugi termin';

  @override
  String get book => 'Zakaži';

  @override
  String timeRange(String start, String end, String zone) {
    return '$start – $end ($zone)';
  }

  @override
  String get slotTaken =>
      'Ovaj termin više nije slobodan. Odaberite drugi termin.';

  @override
  String get consentNeeded =>
      'Za zakazivanje kod ove klinike potrebna je vaša saglasnost.';

  @override
  String get appointmentBooked => 'Vaš termin je zakazan.';

  @override
  String get unlistedClinic => 'Klinika koja više nije na listi';

  @override
  String get cancelTitle => 'Otkazati ovaj termin?';

  @override
  String cancelBody(String date, String time) {
    return 'Vaš termin $date u $time bit će otkazan. Dobit ćete e-mail kojim se otkazivanje potvrđuje.';
  }

  @override
  String get keepIt => 'Zadrži termin';

  @override
  String get cancelAppointment => 'Otkaži termin';

  @override
  String get noAppointments => 'Još nemate zakazanih termina.';

  @override
  String get upcoming => 'Predstojeći';

  @override
  String get past => 'Prošli';

  @override
  String get statusBooked => 'Zakazan';

  @override
  String get statusCancelled => 'Otkazan';

  @override
  String get statusAttended => 'Održan';

  @override
  String get statusNotAttended => 'Propušten';

  @override
  String get appointmentCancelled => 'Vaš termin je otkazan.';

  @override
  String get cancelWindowHint =>
      'Termin možete otkazati u aplikaciji najkasnije 24 sata prije početka.';

  @override
  String get cancelWindowClosed =>
      'Ovaj termin počinje za manje od 24 sata i više ga nije moguće otkazati u aplikaciji.';

  @override
  String get appointmentClosed => 'Ovaj termin više nije zakazan.';

  @override
  String consentTitle(String clinic) {
    return 'Dijeljenje vaših podataka s klinikom $clinic';
  }

  @override
  String consentIntro(String clinic) {
    return 'Prije prvog zakazivanja termina kod klinike $clinic potrebna nam je vaša saglasnost da s njom podijelimo vaše podatke.';
  }

  @override
  String get whatIsShared => 'Šta se dijeli';

  @override
  String get whatIsSharedBody =>
      'Vaš profil (ime, datum rođenja i broj telefona, ako ste ga naveli) i vaši termini kod ove klinike.';

  @override
  String get whoSeesIt => 'Ko vidi podatke';

  @override
  String whoSeesItBody(String clinic) {
    return '$clinic. Ljekar kod kojeg zakažete termin vidi vaš profil; administratori klinike vide vaše ime i vrijeme vaših termina.';
  }

  @override
  String get whatFor => 'U koju svrhu';

  @override
  String get whatForBody =>
      'Isključivo za vaše liječenje u ovoj klinici: za pripremu i održavanje vaših termina.';

  @override
  String get whoIsResponsible => 'Ko je odgovoran';

  @override
  String whoIsResponsibleBody(String clinic) {
    return 'Za vaše podatke odgovara $clinic. Salus ih obrađuje u ime klinike.';
  }

  @override
  String get readPrivacyNotice => 'Pročitajte obavještenje o privatnosti';

  @override
  String get privacyNoticeFailed =>
      'Obavještenje o privatnosti nije moguće otvoriti.';

  @override
  String get howToRevoke => 'Kako povući saglasnost';

  @override
  String get howToRevokeBody =>
      'Ovu saglasnost možete povući u bilo kojem trenutku u odjeljku Moje saglasnosti. Klinika tada više ne vidi vaš profil. Vaše ime i podatke o terminima koji su već održani zadržava kao dio svoje medicinske dokumentacije. Vaši budući termini kod te klinike se otkazuju.';

  @override
  String get agreeAndBook => 'Pristajem i zakazujem';

  @override
  String get revokeTitle => 'Povući saglasnost?';

  @override
  String revokeBody(String clinic) {
    return '$clinic više neće vidjeti vaš profil. Vaše ime i podatke o terminima koji su već održani zadržat će kao dio svoje medicinske dokumentacije. Vaši budući termini kod klinike $clinic bit će otkazani. Prestaje i dijeljenje vaše historije iz drugih klinika s klinikom $clinic; ponovni pristanak ga ne vraća.';
  }

  @override
  String get keepConsent => 'Zadrži saglasnost';

  @override
  String get revokeConsent => 'Povuci saglasnost';

  @override
  String get noConsents => 'Još niste dali nijednu saglasnost.';

  @override
  String get consentsIntro =>
      'S ovim klinikama dijelite svoj profil i termine radi svog liječenja.';

  @override
  String givenOn(String date) {
    return 'Data $date';
  }

  @override
  String get consentActive => 'Aktivna';

  @override
  String revokedOn(String date) {
    return 'Povučena $date';
  }

  @override
  String get consentRevoked =>
      'Vaša saglasnost je povučena. Vaši budući termini kod ove klinike bit će otkazani i za svaki ćete dobiti e-mail.';

  @override
  String get consentAlreadyRevoked => 'Ova saglasnost je već povučena.';

  @override
  String get joinConsultation => 'Pridruži se konsultaciji';

  @override
  String get joinFrom => 'Možete se pridružiti od 10 minuta prije početka.';

  @override
  String get checkTitle => 'Provjerite kameru i mikrofon';

  @override
  String get checkIntro => 'Ljekar će vas vidjeti i čuti kada se pridružite.';

  @override
  String get noSound =>
      'Zvuk se ne čuje. Provjerite da je mikrofon uključen i da nije utišan.';

  @override
  String get joinCameraOff => 'Pridruži se s isključenom kamerom';

  @override
  String get joinMicrophoneOff => 'Pridruži se s isključenim mikrofonom';

  @override
  String get joinNow => 'Pridruži se';

  @override
  String get joining => 'Pridruživanje…';

  @override
  String get cameraDenied =>
      'Salus nema pristup vašoj kameri. Pristup možete dozvoliti u postavkama uređaja ili se pridružiti bez videa.';

  @override
  String get microphoneDenied =>
      'Salus nema pristup vašem mikrofonu. Pristup možete dozvoliti u postavkama uređaja. Bez mikrofona vas ljekar ne može čuti.';

  @override
  String get openSettings => 'Otvori postavke';

  @override
  String get noCamera => 'Kamera nije pronađena.';

  @override
  String get noMicrophone => 'Mikrofon nije pronađen.';

  @override
  String get waiting => 'Čeka se ljekar';

  @override
  String joinsYouHere(String name) {
    return '$name će vam se pridružiti ovdje. Ne zatvarajte ovaj ekran.';
  }

  @override
  String get mute => 'Isključi mikrofon';

  @override
  String get unmute => 'Uključi mikrofon';

  @override
  String get cameraOff => 'Isključi kameru';

  @override
  String get cameraOn => 'Uključi kameru';

  @override
  String get leave => 'Napusti';

  @override
  String peerCameraOff(String name) {
    return '$name ima isključenu kameru';
  }

  @override
  String peerMuted(String name) {
    return '$name ima isključen mikrofon';
  }

  @override
  String peerLeft(String name) {
    return '$name više nije u konsultaciji. Može se ponovo pridružiti dok konsultacija ne završi.';
  }

  @override
  String get reconnecting => 'Veza je prekinuta. Ponovno povezivanje…';

  @override
  String get notRestored => 'Vezu nije bilo moguće ponovo uspostaviti.';

  @override
  String get reconnected => 'Veza je ponovo uspostavljena';

  @override
  String get consultationOver => 'Konsultacija je završena';

  @override
  String get closeScreen => 'Možete zatvoriti ovaj ekran.';

  @override
  String get timeLimit =>
      'Vrijeme predviđeno za konsultaciju je isteklo i konsultacija je završena.';

  @override
  String get practitionerAbsent => 'Ljekar se nije pridružio';

  @override
  String get timePassed =>
      'Vrijeme za pridruživanje ovoj konsultaciji je isteklo. Klinika vidi da ste se pridružili. Klinika vas može kontaktirati, a možete i zakazati novi termin.';

  @override
  String get withdrewConsent =>
      'Povukli ste saglasnost za ovu kliniku, pa ste izašli iz konsultacije.';

  @override
  String get appointmentWasCancelled =>
      'Ovaj termin je otkazan, pa je konsultacija završena.';

  @override
  String get windowNotOpen =>
      'Ova konsultacija se otvara 10 minuta prije početka.';

  @override
  String get consultationEnded => 'Ova konsultacija je završena.';

  @override
  String get joinNeedsConsent =>
      'Za pridruživanje je potrebna vaša saglasnost za ovu kliniku.';

  @override
  String get consultationNotAvailable => 'Ova konsultacija nije dostupna.';

  @override
  String get cannotConnect =>
      'Video vezu nije bilo moguće pokrenuti. Provjerite internet vezu i pokušajte ponovo.';

  @override
  String get whyCameraAndMicrophone =>
      'Da biste se pridružili video konsultaciji, Salus treba pristup vašoj kameri i mikrofonu. Konsultacija se ne snima ni kao zvuk ni kao video. Ako se složite, ono što se kaže transkribuje se i prevodi.';

  @override
  String get transcriptionOn => 'Transkripcija je uključena';

  @override
  String get transcriptionTitle => 'Transkripcija i prijevod';

  @override
  String transcriptionIntro(String clinic) {
    return 'U vašim video konsultacijama s klinikom $clinic ono što vi i ljekar kažete može se transkribovati dok govorite i prevesti, kako bi vas ljekar mogao pratiti na vašem jeziku.';
  }

  @override
  String get transcriptionAssistance =>
      'Na osnovu transkripta ljekar dobija i pomoć vještačke inteligencije (AI): prijedloge tokom konsultacije i nacrte dokumenata. To su samo prijedlozi; ljekar ih provjerava i donosi svaku odluku.';

  @override
  String transcriptionKept(String clinic) {
    return 'Transkript se čuva uz vašu medicinsku dokumentaciju u klinici $clinic. Konsultacija se ne snima ni kao zvuk ni kao video.';
  }

  @override
  String get transcriptionDecline =>
      'Ako se ne složite, konsultacija se odvija kao i obično, bez transkripcije, prijevoda i pomoći vještačke inteligencije.';

  @override
  String get transcriptionWithdraw =>
      'Saglasnost možete povući u bilo kojem trenutku, i tokom konsultacije: dodirnite „Transkripcija je uključena” ili otvorite Moje saglasnosti. Transkripcija prestaje odmah; ono što je do tada transkribovano ostaje u vašoj dokumentaciji.';

  @override
  String get transcriptionAgree => 'Prihvati i nastavi';

  @override
  String get transcriptionWithout => 'Nastavi bez transkripcije';

  @override
  String get stopTranscriptionTitle => 'Zaustaviti transkripciju?';

  @override
  String stopTranscriptionBody(String clinic) {
    return 'Transkripcija, prijevod i pomoć vještačke inteligencije odmah prestaju u vašim konsultacijama s klinikom $clinic. Konsultacija koja je u toku nastavlja se bez njih, a vaši termini ostaju kakvi jesu. Ono što je do sada transkribovano ostaje u vašoj dokumentaciji.';
  }

  @override
  String get keepTranscription => 'Zadrži transkripciju';

  @override
  String get stopTranscription => 'Zaustavi transkripciju';

  @override
  String get transcriptionStopped =>
      'Transkripcija je isključena. Vaše konsultacije s ovom klinikom odvijaju se bez nje.';

  @override
  String get consentForCare => 'Dijeljenje radi liječenja';

  @override
  String get consentForTranscription =>
      'Transkripcija i pomoć vještačke inteligencije';

  @override
  String get toFillIn => 'Za popunjavanje';

  @override
  String get inProgress => 'U toku';

  @override
  String get submittedStatus => 'Poslan';

  @override
  String get closedStatus => 'Zatvoren';

  @override
  String get fillIn => 'Popuni';

  @override
  String get continueFilling => 'Nastavi';

  @override
  String get viewAnswers => 'Pogledaj odgovore';

  @override
  String get submitBeforeStart =>
      'Molimo vas da ga pošaljete prije početka termina.';

  @override
  String get questionnaireClosed =>
      'Termin je počeo, pa ovaj upitnik više nije moguće popuniti.';

  @override
  String savedAsYouGo(String clinic) {
    return 'Vaši odgovori se čuvaju dok popunjavate upitnik. $clinic ih vidi tek nakon što ih pošaljete.';
  }

  @override
  String get saved => 'Sačuvano';

  @override
  String get saving => 'Čuvanje…';

  @override
  String get notSaved =>
      'Nije sačuvano. Provjerite internet vezu. Aplikacija će pokušati ponovo.';

  @override
  String get requiredQuestion => 'Obavezno';

  @override
  String get moreInformation => 'Više informacija';

  @override
  String get yourAnswer => 'Vaš odgovor';

  @override
  String get chooseDate => 'Odaberite datum';

  @override
  String get selectOne => 'Odaberite jedan odgovor.';

  @override
  String get selectAll => 'Odaberite sve što odgovara.';

  @override
  String get yes => 'Da';

  @override
  String get no => 'Ne';

  @override
  String get questionRequired => 'Ovo pitanje je obavezno.';

  @override
  String get enterWholeNumber => 'Unesite cijeli broj.';

  @override
  String get enterNumber => 'Unesite broj.';

  @override
  String get numberTooLarge => 'Ovaj broj je prevelik.';

  @override
  String upToCharacters(int max) {
    return 'Unesite najviše $max znakova.';
  }

  @override
  String get enterValidDate => 'Unesite ispravan datum.';

  @override
  String get submit => 'Pošalji';

  @override
  String get submitTitle => 'Poslati odgovore?';

  @override
  String submitBody(String clinic) {
    return 'Nakon slanja ih više ne možete mijenjati. Vidjet će ih vaš ljekar u klinici $clinic.';
  }

  @override
  String get cancel => 'Odustani';

  @override
  String get requiredMissing => 'Na neka obavezna pitanja niste odgovorili.';

  @override
  String get answersSubmitted => 'Odgovori su poslani';

  @override
  String get answersSubmittedBody =>
      'Vaš ljekar ih može vidjeti prije termina. Možete ih pogledati u odjeljku Moji termini.';

  @override
  String get yourAnswers => 'Vaši odgovori';

  @override
  String submittedAt(String when) {
    return 'Poslano $when';
  }

  @override
  String dateAndTime(String date, String time) {
    return '$date, $time';
  }

  @override
  String get notAnswered => 'Bez odgovora';

  @override
  String get leaveUnsaved =>
      'Vaš posljednji odgovor još nije sačuvan. Želite li ipak izaći?';

  @override
  String get stay => 'Ostani';

  @override
  String get fillingNeedsConsent =>
      'Za popunjavanje ovog upitnika potrebna je vaša saglasnost za ovu kliniku.';

  @override
  String get pastDue =>
      'Termin je počeo, pa ovaj upitnik više nije moguće mijenjati.';

  @override
  String get withdrawnByClinic =>
      'Klinika je povukla ovaj upitnik. Ne morate ga popunjavati.';

  @override
  String get alreadySubmitted => 'Već ste poslali ovaj upitnik.';

  @override
  String get changedElsewhere =>
      'Ovi odgovori su izmijenjeni na drugom uređaju. Prikazani su najnoviji odgovori.';

  @override
  String get questionnaireNotAvailable => 'Ovaj upitnik nije dostupan.';

  @override
  String get done => 'Gotovo';

  @override
  String get addFile => 'Dodaj datoteku';

  @override
  String get photosUpTo => 'Fotografije do 20 MB.';

  @override
  String get documentsUpTo => 'Dokumenti (PDF) do 20 MB.';

  @override
  String get videosUpTo => 'Videozapisi do 200 MB.';

  @override
  String get oneFileHint => '1 datoteka.';

  @override
  String upToFilesHint(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Najviše $n datoteka.',
      few: 'Najviše $n datoteke.',
      one: 'Najviše $n datoteka.',
    );
    return '$_temp0';
  }

  @override
  String get takePhoto => 'Snimi fotografiju';

  @override
  String get recordVideo => 'Snimi video';

  @override
  String get chooseFromPhotos => 'Odaberi iz galerije';

  @override
  String get chooseDocument => 'Odaberi dokument';

  @override
  String get preparing => 'Priprema…';

  @override
  String uploadingPercent(int n) {
    return 'Otpremanje… $n %';
  }

  @override
  String get checkingFile => 'Provjera datoteke…';

  @override
  String get added => 'Dodano';

  @override
  String get notUploaded => 'Nije otpremljeno. Provjerite internet vezu.';

  @override
  String get remove => 'Ukloni';

  @override
  String get removeFile => 'Ukloni datoteku';

  @override
  String get removeFileTitle => 'Ukloniti ovu datoteku?';

  @override
  String get removeFileBody => 'Bit će obrisana iz vaših odgovora.';

  @override
  String get fileRejected =>
      'Ova datoteka nije prošla sigurnosnu provjeru i obrisana je. Odaberite drugu datoteku.';

  @override
  String fileTooLarge(int max) {
    return 'Ova datoteka je veća od $max MB.';
  }

  @override
  String get fileTypeRefused =>
      'Ovu vrstu datoteke nije moguće dodati uz ovo pitanje.';

  @override
  String tooManyFiles(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: 'Uz ovo pitanje možete dodati najviše $n datoteka.',
      few: 'Uz ovo pitanje možete dodati najviše $n datoteke.',
      one: 'Uz ovo pitanje možete dodati najviše $n datoteku.',
    );
    return '$_temp0';
  }

  @override
  String get filesFull =>
      'Datoteke u ovom upitniku ukupno su dostigle 500 MB. Uklonite neku datoteku da biste dodali novu.';

  @override
  String get fileNotUploaded =>
      'Datoteku nije bilo moguće otpremiti. Pokušajte ponovo.';

  @override
  String get cannotPrepareFile =>
      'Ovu datoteku nije bilo moguće pripremiti za otpremanje. Odaberite drugu datoteku.';

  @override
  String get filesBeingChecked =>
      'Neke datoteke se još provjeravaju. Upitnik možete poslati kada budu spremne.';

  @override
  String get leaveUploading =>
      'Datoteka se još otprema. Želite li ipak izaći? Otpremanje će se prekinuti.';

  @override
  String fileCount(int n) {
    String _temp0 = intl.Intl.pluralLogic(
      n,
      locale: localeName,
      other: '$n datoteka',
      few: '$n datoteke',
      one: '$n datoteka',
    );
    return '$_temp0';
  }

  @override
  String get photoKind => 'Fotografija';

  @override
  String get documentKind => 'Dokument';

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
    return 'Otpremanje datoteke $title, $n posto';
  }

  @override
  String fileLabel(String kind, String title, String status) {
    return '$kind, $title, $status';
  }

  @override
  String get allowCamera => 'Dozvolite pristup kameri';

  @override
  String get whyCameraForFiles =>
      'Salus treba pristup kameri da biste snimili fotografiju ili video koji dodajete u ovaj upitnik.';

  @override
  String get continueLabel => 'Nastavi';

  @override
  String get cameraDeniedForFiles =>
      'Salus nema pristup vašoj kameri. Pristup možete dozvoliti u postavkama uređaja ili umjesto toga odabrati datoteku.';

  @override
  String get photosDenied =>
      'Salus nema pristup vašim fotografijama. Pristup možete dozvoliti u postavkama uređaja.';

  @override
  String cameraPhotoTitle(String date, String time) {
    return 'Fotografija, $date, $time';
  }

  @override
  String cameraVideoTitle(String date, String time) {
    return 'Video, $date, $time';
  }

  @override
  String get specialty394814009 => 'Porodična medicina';

  @override
  String get specialty419192003 => 'Interna medicina';

  @override
  String get specialty394579002 => 'Kardiologija';

  @override
  String get specialty394582007 => 'Dermatologija';

  @override
  String get specialty394586005 => 'Ginekologija';

  @override
  String get specialty394537008 => 'Pedijatrija';

  @override
  String get specialty394587001 => 'Psihijatrija';

  @override
  String get specialty394591006 => 'Neurologija';

  @override
  String get specialty418960008 => 'Otorinolaringologija (ORL)';

  @override
  String get specialty394594003 => 'Oftalmologija';

  @override
  String get specialty394801008 => 'Ortopedija';

  @override
  String get specialty394583002 => 'Endokrinologija';

  @override
  String questionnaireNotInLanguage(String language) {
    return 'Ovaj upitnik nije dostupan na jeziku: $language.';
  }

  @override
  String questionnaireLanguages(String languages) {
    return 'Dostupan je na ovim jezicima: $languages. Odaberite onaj koji najbolje razumijete.';
  }

  @override
  String continueInLanguage(String language) {
    return 'Nastavi na jeziku: $language';
  }

  @override
  String get languageUnavailable =>
      'Ovaj jezik više nije dostupan za ovaj upitnik. Odaberite drugi.';

  @override
  String specialtyAtClinic(String specialty, String clinic) {
    return '$specialty, $clinic';
  }

  @override
  String get documents => 'Dokumenti';

  @override
  String get noDocuments =>
      'Za ovaj termin još nema potpisanih dokumenata. Dokument se pojavljuje ovdje kada ga vaš ljekar potpiše.';

  @override
  String documentSignedOn(String date) {
    return 'Potpisano $date';
  }

  @override
  String documentLanguage(String language) {
    return 'Jezik: $language';
  }

  @override
  String get sharePdf => 'Sačuvaj ili podijeli PDF';

  @override
  String get transcript => 'Transkript';

  @override
  String get transcriptNote =>
      'Konsultacija onako kako je automatski transkribovana i prevedena. Može sadržavati greške.';

  @override
  String get transcriptEmpty =>
      'U ovoj konsultaciji ništa nije transkribovano.';

  @override
  String get speakerYou => 'Vi';

  @override
  String get speakerPractitioner => 'Ljekar';

  @override
  String get shareTranscript => 'Podijeli transkript';

  @override
  String transcriptTranslation(String text) {
    return 'Prijevod: $text';
  }

  @override
  String get termExplanationNote => 'Informativno, nije dijagnoza';

  @override
  String get termNotReviewed => 'Još nije provjereno';

  @override
  String get termClose => 'Zatvori';

  @override
  String get chatsTitle => 'Poruke';

  @override
  String get chatsEmpty => 'Još nemate razgovora.';

  @override
  String chatUnread(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count nepročitanih poruka',
      few: '$count nepročitane poruke',
      one: '$count nepročitana poruka',
    );
    return '$_temp0';
  }

  @override
  String get chatDoctorUnknown => 'Vaš ljekar';

  @override
  String get chatMessageHint => 'Napišite poruku';

  @override
  String get chatSend => 'Pošalji';

  @override
  String get chatAttach => 'Dodaj datoteku';

  @override
  String get chatSendFailed =>
      'Poruka nije poslana. Provjerite internet vezu i pokušajte ponovo.';

  @override
  String get chatFileRejected => 'Ova datoteka nije prihvaćena.';

  @override
  String get chatFileRemove => 'Ukloni datoteku';

  @override
  String chatTooManyFiles(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other:
          'Već imate $count datoteka koje čekaju slanje. Neposlane datoteke se uklanjaju nakon 24 sata; pokušajte ponovo kasnije.',
      few:
          'Već imate $count datoteke koje čekaju slanje. Neposlane datoteke se uklanjaju nakon 24 sata; pokušajte ponovo kasnije.',
      one:
          'Već imate $count datoteku koja čeka slanje. Neposlane datoteke se uklanjaju nakon 24 sata; pokušajte ponovo kasnije.',
    );
    return '$_temp0';
  }

  @override
  String get chatOpenFailed =>
      'Datoteka se nije mogla otvoriti. Pokušajte ponovo.';

  @override
  String get chatEndedSession =>
      'Vaša sesija je istekla. Ponovo se prijavite da vidite nove poruke.';

  @override
  String get chatEnded =>
      'Nove poruke se u ovom razgovoru više ne mogu prikazati.';

  @override
  String get chatHours => 'Radno vrijeme za poruke';

  @override
  String chatHoursDay(String day, String start, String end) {
    return '$day: $start–$end';
  }

  @override
  String get chatHoursNone => 'Klinika nije odredila radno vrijeme za poruke.';

  @override
  String chatHoursZone(String zone) {
    return 'Vremena su u vremenskoj zoni klinike, $zone.';
  }

  @override
  String chatResponseTime(int hours) {
    String _temp0 = intl.Intl.pluralLogic(
      hours,
      locale: localeName,
      other: 'Klinika obično odgovara u roku od $hours sati.',
      few: 'Klinika obično odgovara u roku od $hours sata.',
      one: 'Klinika obično odgovara u roku od $hours sata.',
    );
    return '$_temp0';
  }

  @override
  String chatOutOfHours(int hours) {
    String _temp0 = intl.Intl.pluralLogic(
      hours,
      locale: localeName,
      other:
          'Klinika je trenutno izvan radnog vremena za poruke. I dalje možete pisati; obično odgovara u roku od $hours sati.',
      few:
          'Klinika je trenutno izvan radnog vremena za poruke. I dalje možete pisati; obično odgovara u roku od $hours sata.',
      one:
          'Klinika je trenutno izvan radnog vremena za poruke. I dalje možete pisati; obično odgovara u roku od $hours sata.',
    );
    return '$_temp0';
  }

  @override
  String chatSystemInvited(String reason) {
    String _temp0 = intl.Intl.selectLogic(reason, {
      'second_opinion': 'Ljekar iz druge klinike pozvan je u ovaj razgovor radi drugog mišljenja.',
      'specialist_review': 'Ljekar iz druge klinike pozvan je u ovaj razgovor radi specijalističkog pregleda.',
      'handover': 'Ljekar iz druge klinike pozvan je u ovaj razgovor da preuzme liječenje.',
      'other': 'Ljekar iz druge klinike pozvan je u ovaj razgovor.',
    });
    return '$_temp0';
  }

  @override
  String chatSystemAdded(String reason) {
    String _temp0 = intl.Intl.selectLogic(reason, {
      'second_opinion': 'Ljekar se pridružio razgovoru radi drugog mišljenja.',
      'specialist_review':
          'Ljekar se pridružio razgovoru radi specijalističkog pregleda.',
      'handover': 'Ljekar se pridružio razgovoru da preuzme liječenje.',
      'other': 'Ljekar se pridružio razgovoru.',
    });
    return '$_temp0';
  }

  @override
  String get chatSystemRemoved => 'Ljekar više ne učestvuje u razgovoru.';

  @override
  String get chatSystemGranted =>
      'Dozvolili ste ljekaru iz druge klinike da se pridruži razgovoru.';

  @override
  String get chatSystemDeclined => 'Odbili ste ljekara iz druge klinike.';

  @override
  String get chatSystemOther => 'Razgovor je ažuriran.';

  @override
  String get chatInviteTitle =>
      'Ljekar iz druge klinike traži pristup razgovoru';

  @override
  String chatInviteBody(String reason) {
    String _temp0 = intl.Intl.selectLogic(reason, {
      'second_opinion': 'Vaša klinika traži da se ljekar iz druge klinike pridruži ovom razgovoru radi drugog mišljenja.',
      'specialist_review': 'Vaša klinika traži da se ljekar iz druge klinike pridruži ovom razgovoru radi specijalističkog pregleda.',
      'handover': 'Vaša klinika traži da se ljekar iz druge klinike pridruži ovom razgovoru da preuzme vaše liječenje.',
      'other': 'Vaša klinika traži da se ljekar iz druge klinike pridruži ovom razgovoru.',
    });
    return '$_temp0';
  }

  @override
  String chatInviteWho(String doctor, String clinic) {
    return 'Ljekar: $doctor, $clinic';
  }

  @override
  String get chatInviteUnresolved =>
      'Ime i klinika pozvanog ljekara nisu mogli biti učitani. Pristup možete odobriti tek kada budu prikazani.';

  @override
  String chatInviteSees(String history) {
    String _temp0 = intl.Intl.selectLogic(history, {
      'full': 'Ako pristanete, ljekar vidi cijeli razgovor, uključujući ranije poruke, i može pisati u njemu.',
      'from': 'Ako pristanete, ljekar vidi razgovor od poruke koju je odabrala vaša klinika nadalje i može pisati u njemu.',
      'other': 'Ako pristanete, ljekar vidi razgovor i može pisati u njemu.',
    });
    return '$_temp0';
  }

  @override
  String get chatInviteChoice =>
      'Možete odbiti. Ako pristanete, pristup ljekara možete kasnije opozvati u ovom razgovoru; prestaje odmah.';

  @override
  String get chatGrant => 'Odobri pristup';

  @override
  String get chatDecline => 'Odbij';

  @override
  String get chatGranted =>
      'Pristup je odobren. Možete ga opozvati u ovom razgovoru u bilo kojem trenutku.';

  @override
  String get chatDeclined =>
      'Odbijeno. Ljekar se ne pridružuje ovom razgovoru.';

  @override
  String get chatRevoke => 'Opozovi pristup ljekara iz druge klinike';

  @override
  String chatGrantedTo(String doctor, String clinic) {
    return 'Pristup je odobren: $doctor, $clinic.';
  }

  @override
  String get chatGrantedToUnknown =>
      'Pristup je odobren ljekaru iz druge klinike.';

  @override
  String get chatRevokeBody =>
      'Ljekar iz druge klinike odmah gubi pristup ovom razgovoru. Već napisane poruke ostaju u njemu.';

  @override
  String get chatRevokeConfirm => 'Opozovi pristup';

  @override
  String get chatRevoked => 'Pristup je opozvan.';

  @override
  String get allergies => 'Alergije';

  @override
  String get currentMedications => 'Trenutni lijekovi';

  @override
  String get allergiesIntro =>
      'Klinike kojima ste dali saglasnost vide alergije iz vašeg profila kada imate termin kod njih.';

  @override
  String get medicationsIntro =>
      'Klinike kojima ste dali saglasnost vide lijekove iz vašeg profila kada imate termin kod njih.';

  @override
  String get noAllergies => 'Niste dodali nijednu alergiju.';

  @override
  String get noMedications => 'Niste dodali nijedan lijek.';

  @override
  String get addAllergy => 'Dodaj alergiju';

  @override
  String get addMedication => 'Dodaj lijek';

  @override
  String get changeAllergy => 'Izmjena alergije';

  @override
  String get changeMedication => 'Izmjena lijeka';

  @override
  String get change => 'Izmijeni';

  @override
  String get allergySubstance => 'Supstanca ili namirnica';

  @override
  String get medicationName => 'Lijek';

  @override
  String get termSearchHint =>
      'Počnite kucati za pretragu ili upišite svojim riječima.';

  @override
  String get allergyCategory => 'Vrsta (nije obavezno)';

  @override
  String get notSpecified => 'Nije navedeno';

  @override
  String get categoryFood => 'Hrana';

  @override
  String get categoryMedication => 'Lijek';

  @override
  String get categoryEnvironment => 'Okolina';

  @override
  String get categoryBiologic => 'Biološki preparat';

  @override
  String get reactionOptional => 'Reakcija (nije obavezno)';

  @override
  String get severity => 'Težina';

  @override
  String get severityMild => 'Blaga';

  @override
  String get severityModerate => 'Umjerena';

  @override
  String get severitySevere => 'Teška';

  @override
  String get entryStatus => 'Status';

  @override
  String get allergyActive => 'Aktivna';

  @override
  String get allergyInactive => 'Neaktivna';

  @override
  String get allergyResolved => 'Prošla';

  @override
  String reactionLine(String reaction) {
    return 'Reakcija: $reaction';
  }

  @override
  String reactionWithSeverity(String reaction, String severity) {
    return 'Reakcija: $reaction ($severity)';
  }

  @override
  String get confirmedByClinician => 'Potvrdio ljekar';

  @override
  String get recordedByClinic => 'Unijela klinika tokom jedne od vaših posjeta';

  @override
  String get confirmedChangeNote =>
      'Ako izmijenite ovaj unos, više neće biti potvrđen od ljekara.';

  @override
  String get nameRequired => 'Unesite naziv, najviše 200 znakova.';

  @override
  String get addToProfile => 'Dodaj u moj profil';

  @override
  String get addToProfileTitle => 'Dodati u vaš profil?';

  @override
  String get addToProfileBody =>
      'Tada će ga vidjeti svaka klinika kojoj ste dali saglasnost, bez naziva klinike koja ga je unijela.';

  @override
  String get addedToProfile => 'Sada je u vašem profilu.';

  @override
  String get removeEntryTitle => 'Ukloniti ovaj unos?';

  @override
  String get removeEntryBody =>
      'Više se neće prikazivati ni vama ni bilo kojoj klinici. Ako ga je klinika unijela ili potvrdila, ostaje u vašem medicinskom kartonu, označen kao uklonjen.';

  @override
  String get entryRemoved => 'Unos je uklonjen.';

  @override
  String get entrySaved => 'Sačuvano u vašem profilu.';

  @override
  String get entryChanged =>
      'Ovaj unos je u međuvremenu izmijenjen. Prikazana je najnovija verzija.';

  @override
  String get entryInvalid =>
      'Neki podaci nisu ispravni. Provjerite ih i pokušajte ponovo.';

  @override
  String get entryNotAllowed => 'Ne možete izmijeniti ovaj unos.';

  @override
  String get entryNotFound => 'Ovaj unos više nije dostupan.';

  @override
  String get doseHeading => 'Doza (nije obavezno)';

  @override
  String get doseAmount => 'Količina';

  @override
  String get doseUnit => 'Jedinica';

  @override
  String get unitMg => 'miligrami (mg)';

  @override
  String get unitG => 'grami (g)';

  @override
  String get unitUg => 'mikrogrami (µg)';

  @override
  String get unitMl => 'mililitri (ml)';

  @override
  String get unitIu => 'internacionalne jedinice (IJ)';

  @override
  String get unitTablet => 'tablete';

  @override
  String get unitCapsule => 'kapsule';

  @override
  String get unitDrop => 'kapi';

  @override
  String get unitPuff => 'udisaji (inhalator)';

  @override
  String get unitPatch => 'flasteri';

  @override
  String get doseBoth => 'Unesite i količinu i jedinicu ili nijedno.';

  @override
  String get numberAboveZero => 'Unesite broj veći od 0.';

  @override
  String get howOftenHeading => 'Koliko često (nije obavezno)';

  @override
  String get timesLabel => 'Puta';

  @override
  String get everyLabel => 'Svakih';

  @override
  String get periodUnitLabel => 'Jedinica vremena';

  @override
  String get periodHours => 'sati';

  @override
  String get periodDays => 'dana';

  @override
  String get periodWeeks => 'sedmica';

  @override
  String get periodMonths => 'mjeseci';

  @override
  String get howOftenAll => 'Popunite sva tri polja ili ih ostavite prazna.';

  @override
  String get asNeeded => 'Po potrebi';

  @override
  String get medicationActive => 'Trenutno uzimam';

  @override
  String get medicationStopped => 'Više ne uzimam';

  @override
  String doseMg(String value) {
    return '$value mg';
  }

  @override
  String doseG(String value) {
    return '$value g';
  }

  @override
  String doseUg(String value) {
    return '$value µg';
  }

  @override
  String doseMl(String value) {
    return '$value ml';
  }

  @override
  String doseIu(String value) {
    return '$value IJ';
  }

  @override
  String doseTablet(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value tableta',
      few: '$value tablete',
      one: '$value tableta',
    );
    return '$_temp0';
  }

  @override
  String doseCapsule(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value kapsula',
      few: '$value kapsule',
      one: '$value kapsula',
    );
    return '$_temp0';
  }

  @override
  String doseDrop(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value kapi',
      few: '$value kapi',
      one: '$value kap',
    );
    return '$_temp0';
  }

  @override
  String dosePuff(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value udisaja',
      few: '$value udisaja',
      one: '$value udisaj',
    );
    return '$_temp0';
  }

  @override
  String dosePatch(num count, String value) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$value flastera',
      few: '$value flastera',
      one: '$value flaster',
    );
    return '$_temp0';
  }

  @override
  String doseLine(String dose) {
    return 'Doza: $dose';
  }

  @override
  String howOftenPerHour(int frequency) {
    return '$frequency× na sat';
  }

  @override
  String howOftenPerDay(int frequency) {
    return '$frequency× dnevno';
  }

  @override
  String howOftenPerWeek(int frequency) {
    return '$frequency× sedmično';
  }

  @override
  String howOftenPerMonth(int frequency) {
    return '$frequency× mjesečno';
  }

  @override
  String howOftenEveryHours(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: '$frequency× svakih $every sati',
      few: '$frequency× svaka $every sata',
      one: '$frequency× svaki $every sat',
    );
    return '$_temp0';
  }

  @override
  String howOftenEveryDays(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: '$frequency× svakih $every dana',
      few: '$frequency× svaka $every dana',
      one: '$frequency× svaki $every dan',
    );
    return '$_temp0';
  }

  @override
  String howOftenEveryWeeks(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: '$frequency× svakih $every sedmica',
      few: '$frequency× svake $every sedmice',
      one: '$frequency× svake $every sedmice',
    );
    return '$_temp0';
  }

  @override
  String howOftenEveryMonths(num period, int frequency, String every) {
    String _temp0 = intl.Intl.pluralLogic(
      period,
      locale: localeName,
      other: '$frequency× svakih $every mjeseci',
      few: '$frequency× svaka $every mjeseca',
      one: '$frequency× svaki $every mjesec',
    );
    return '$_temp0';
  }

  @override
  String get myHistory => 'Moja historija';

  @override
  String get historyEmpty =>
      'Vaša historija je još prazna. Ovdje se pojavljuju vaše posjete, dokumenti i odgovori.';

  @override
  String get historyNoMatch =>
      'Ništa u vašoj historiji ne odgovara ovim filterima.';

  @override
  String get historyClinic => 'Klinika';

  @override
  String get historyAllClinics => 'Sve klinike';

  @override
  String get historyVisits => 'Posjete';

  @override
  String get historyVisit => 'Posjeta';

  @override
  String get historyAnswers => 'Odgovori na upitnik';

  @override
  String get historyFiles => 'Datoteke';

  @override
  String get historyFile => 'Datoteka';

  @override
  String get historyAnyDate => 'Bilo koji datum';

  @override
  String historyDateRange(String from, String to) {
    return 'od $from do $to';
  }

  @override
  String get historyClearDates => 'Poništi datume';

  @override
  String get historyShowMore => 'Prikaži više';

  @override
  String get historyMoreFailed =>
      'Ostatak vaše historije nije se mogao učitati.';

  @override
  String get historyOpenFailed =>
      'Ova stavka se nije mogla otvoriti. Pokušajte ponovo.';

  @override
  String get sharingToggle =>
      'Podijeli moju historiju iz drugih klinika s ovom klinikom';

  @override
  String sharingIntro(String clinic) {
    return 'Označite ranije posjete koje $clinic smije vidjeti. Za svaku označenu posjetu ljekari klinike $clinic kod kojih imate termin vide ono što je ovdje prikazano, s dokumentima posjete, odgovorima na upitnike i datotekama, alergijama i lijekovima evidentiranim u njoj te svime što se posjeti kasnije doda. Posjete koje ne označite ostaju skrivene od $clinic.';
  }

  @override
  String get sharingChangeLater =>
      'Posjete možete dodati ili ukloniti, ili prestati dijeliti, u bilo kojem trenutku u odjeljku Podijeljena historija. Svaka promjena važi odmah.';

  @override
  String get sharingNoVisits =>
      'Nemate ranijih posjeta u drugim klinikama koje biste mogli podijeliti.';

  @override
  String get sharingSelectAll => 'Označi sve';

  @override
  String get sharingSelectNone => 'Poništi sve';

  @override
  String get sharedHistory => 'Podijeljena historija';

  @override
  String get sharedHistoryIntro =>
      'Klinike kojima ste dali saglasnost i koliko vaših posjeta iz drugih klinika svaka od njih vidi.';

  @override
  String get sharedHistoryNone => 'Još niste dali saglasnost nijednoj klinici.';

  @override
  String sharedVisits(int count) {
    String _temp0 = intl.Intl.pluralLogic(
      count,
      locale: localeName,
      other: '$count podijeljenih posjeta',
      few: '$count podijeljene posjete',
      one: '$count podijeljena posjeta',
      zero: 'Ništa nije podijeljeno',
    );
    return '$_temp0';
  }

  @override
  String get sharingSaved => 'Vaš izbor je sačuvan i važi odmah.';

  @override
  String get sharingStop => 'Prestani dijeliti';

  @override
  String get sharingStopTitle => 'Prestati dijeliti vašu historiju?';

  @override
  String sharingStopBody(String clinic) {
    return '$clinic više neće vidjeti nijednu posjetu iz drugih klinika koju ste s njom podijelili. Ovo važi odmah.';
  }

  @override
  String get sharingKeep => 'Nastavi dijeliti';

  @override
  String get sharingStopped =>
      'Više ne dijelite svoju historiju s ovom klinikom.';

  @override
  String get sharingChanged =>
      'Vaše podijeljene posjete su se u međuvremenu promijenile. Provjerite ih i ponovo sačuvajte.';

  @override
  String get bookedNotShared =>
      'Vaš termin je zakazan, ali vaša historija nije mogla biti podijeljena. Možete je podijeliti u odjeljku Podijeljena historija.';

  @override
  String get consentForHistory => 'Posjete podijeljene iz drugih klinika';

  @override
  String get confirmAttendance => 'Potvrdi dolazak';

  @override
  String get appointmentConfirmedStatus => 'Potvrđen';

  @override
  String get appointmentConfirmedNotice => 'Vaš termin je potvrđen.';

  @override
  String get changeTime => 'Promijeni vrijeme';

  @override
  String rescheduleUntil(String date, String time) {
    return 'Vrijeme možete promijeniti do $date u $time.';
  }

  @override
  String reschedulesLeft(int count) {
    return 'Preostalo promjena vremena: $count';
  }

  @override
  String get rescheduleTitle => 'Odaberite novo vrijeme';

  @override
  String rescheduleCurrent(String date, String time) {
    return 'Sada zakazano: $date u $time';
  }

  @override
  String get rescheduleConfirmTitle => 'Premjestiti termin?';

  @override
  String rescheduleConfirmBody(String date, String time) {
    return 'Vaš termin bit će premješten na $date u $time. Dosadašnje vrijeme bit će oslobođeno.';
  }

  @override
  String get moveAppointment => 'Premjesti termin';

  @override
  String get appointmentMoved => 'Vaš termin je premješten na novo vrijeme.';

  @override
  String get rescheduleWindowClosed =>
      'Vrijeme ovog termina više se ne može promijeniti u aplikaciji. Obratite se klinici.';

  @override
  String get rescheduleLimit =>
      'Vrijeme ovog termina promijenjeno je onoliko puta koliko klinika dozvoljava.';

  @override
  String get notReschedulable =>
      'Vrijeme ovog termina ne može se promijeniti u aplikaciji.';
}
