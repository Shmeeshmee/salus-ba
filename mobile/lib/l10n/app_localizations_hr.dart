// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for Croatian (`hr`).
class AppLocalizationsHr extends AppLocalizations {
  AppLocalizationsHr([String locale = 'hr']) : super(locale);

  @override
  String get tryAgain => 'Pokušajte ponovno';

  @override
  String get language => 'Jezik';

  @override
  String get useDeviceLanguage => 'Koristi jezik uređaja';

  @override
  String get languageSaved => 'Jezik je spremljen.';

  @override
  String get somethingWrong => 'Nešto nije u redu. Pokušajte ponovno.';

  @override
  String get unreachable => 'Salus trenutačno nije dostupan. Provjerite internetsku vezu i pokušajte ponovno.';

  @override
  String get wrongCredentials => 'Adresa e-pošte ili lozinka nisu točni.';

  @override
  String get wrongCode => 'Kod nije točan. Provjerite vrijeme na uređaju i pokušajte ponovno.';

  @override
  String get challengeEnded => 'Prijava je istekla. Ponovno unesite adresu e-pošte i lozinku.';

  @override
  String get emailTaken => 'Za ovu adresu e-pošte već postoji račun. Prijavite se.';

  @override
  String get invalidEmail => 'Unesite ispravnu adresu e-pošte.';

  @override
  String get invalidPassword => 'Lozinka mora imati od 12 do 128 znakova.';

  @override
  String get missingPassword => 'Unesite lozinku.';

  @override
  String get invalidCode => 'Kod ima 6 znamenki.';

  @override
  String get staffOnly => 'Ova je aplikacija namijenjena pacijentima. Osoblje klinike koristi Salus web-aplikaciju.';

  @override
  String get accountReady => 'Vaš je račun spreman. Prijavite se kako biste nastavili.';

  @override
  String get signOutFailed => 'Odjava nije uspjela. Provjerite internetsku vezu i pokušajte ponovno.';

  @override
  String get tooManyAttemptsLater => 'Previše pokušaja. Pokušajte ponovno kasnije.';

  @override
  String tooManyAttempts(int minutes) {
    String _temp0 = intl.Intl.pluralLogic(
      minutes,
      locale: localeName,
      other: 'Previše pokušaja. Pokušajte ponovno za $minutes minuta.',
      few: 'Previše pokušaja. Pokušajte ponovno za $minutes minute.',
      one: 'Previše pokušaja. Pokušajte ponovno za $minutes minutu.',
    );
    return '$_temp0';
  }

  @override
  String get signInTitle => 'Prijava u Salus';

  @override
  String get email => 'E-pošta';

  @override
  String get password => 'Lozinka';

  @override
  String get signIn => 'Prijavi se';

  @override
  String get createAnAccount => 'Izradite račun';

  @override
  String get codePrompt => 'Unesite šesteroznamenkasti kod iz aplikacije za autentifikaciju.';

  @override
  String get code => 'Kod';

  @override
  String get useAnotherAccount => 'Koristi drugi račun';

  @override
  String get passwordHint => 'Od 12 do 128 znakova';

  @override
  String get createAccount => 'Izradi račun';

  @override
  String get haveAccount => 'Već imam račun';

  @override
  String signedInAs(String email) {
    return 'Prijavljeni ste kao $email';
  }

  @override
  String get completeProfile => 'Dopunite profil';

  @override
  String get completeProfileFirst => 'Dopunite profil prije rezervacije termina.';

  @override
  String get bookAppointment => 'Rezerviraj termin';

  @override
  String get myAppointments => 'Moji termini';

  @override
  String get myConsents => 'Moje privole';

  @override
  String get myProfile => 'Moj profil';

  @override
  String get signOut => 'Odjavi se';

  @override
  String get profileIntro =>
      'Klinike u kojima rezervirate termin vide ove podatke. Broj telefona nije obvezan; liječnik ga koristi samo kako bi vas kontaktirao u vezi s terminom.';

  @override
  String get givenName => 'Ime';

  @override
  String get familyName => 'Prezime';

  @override
  String get dateOfBirth => 'Datum rođenja';

  @override
  String get dateFormatHint => 'GGGG-MM-DD';

  @override
  String get phoneOptional => 'Telefon (nije obvezno)';

  @override
  String phoneHint(String example) {
    return 'S pozivnim brojem države, na primjer $example';
  }

  @override
  String get save => 'Spremi';

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
  String get birthDateRange => 'Unesite datum rođenja u prošlosti, ne raniji od 1900. godine.';

  @override
  String invalidPhone(String example) {
    return 'Unesite broj telefona s pozivnim brojem države, na primjer $example.';
  }

  @override
  String get profileChanged => 'Vaš je profil u međuvremenu promijenjen. Provjerite ga i ponovno spremite.';

  @override
  String get chooseClinic => 'Odaberite kliniku';

  @override
  String get noClinics => 'Još nema dostupnih klinika.';

  @override
  String get noPractitioners => 'Ova klinika još nema dostupnih liječnika.';

  @override
  String get choosePractitioner => 'Odaberite liječnika.';

  @override
  String get noOpenTimes => 'Ovaj liječnik još nema slobodnih termina.';

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
  String get bookThisTime => 'Rezervacija termina';

  @override
  String get chooseAnotherTime => 'Odaberite drugi termin';

  @override
  String get book => 'Rezerviraj';

  @override
  String timeRange(String start, String end, String zone) {
    return '$start – $end ($zone)';
  }

  @override
  String get slotTaken => 'Ovaj termin više nije slobodan. Odaberite drugi termin.';

  @override
  String get consentNeeded => 'Za rezervaciju termina u ovoj klinici potrebna je vaša privola.';

  @override
  String get appointmentBooked => 'Vaš je termin rezerviran.';

  @override
  String get unlistedClinic => 'Klinika koja više nije na popisu';

  @override
  String get cancelTitle => 'Otkazati ovaj termin?';

  @override
  String cancelBody(String date, String time) {
    return 'Vaš termin $date u $time bit će otkazan. Primit ćete e-poruku kojom se otkazivanje potvrđuje.';
  }

  @override
  String get keepIt => 'Zadrži termin';

  @override
  String get cancelAppointment => 'Otkaži termin';

  @override
  String get noAppointments => 'Još nemate rezerviranih termina.';

  @override
  String get upcoming => 'Nadolazeći';

  @override
  String get past => 'Prošli';

  @override
  String get statusBooked => 'Rezerviran';

  @override
  String get statusCancelled => 'Otkazan';

  @override
  String get statusAttended => 'Održan';

  @override
  String get statusNotAttended => 'Propušten';

  @override
  String get appointmentCancelled => 'Vaš je termin otkazan.';

  @override
  String get cancelWindowHint => 'Termin možete otkazati u aplikaciji najkasnije 24 sata prije početka.';

  @override
  String get cancelWindowClosed =>
      'Ovaj termin počinje za manje od 24 sata i više ga nije moguće otkazati u aplikaciji.';

  @override
  String get appointmentClosed => 'Ovaj termin više nije rezerviran.';

  @override
  String consentTitle(String clinic) {
    return 'Dijeljenje vaših podataka s klinikom $clinic';
  }

  @override
  String consentIntro(String clinic) {
    return 'Prije prve rezervacije termina u klinici $clinic potrebna nam je vaša privola da s njom podijelimo vaše podatke.';
  }

  @override
  String get whatIsShared => 'Što se dijeli';

  @override
  String get whatIsSharedBody =>
      'Vaš profil (ime, datum rođenja i broj telefona, ako ste ga naveli) i vaši termini u ovoj klinici.';

  @override
  String get whoSeesIt => 'Tko vidi podatke';

  @override
  String whoSeesItBody(String clinic) {
    return '$clinic. Liječnik kod kojeg rezervirate termin vidi vaš profil; administratori klinike vide vaše ime i vrijeme vaših termina.';
  }

  @override
  String get whatFor => 'U koju svrhu';

  @override
  String get whatForBody => 'Isključivo za vaše liječenje u ovoj klinici: za pripremu i održavanje vaših termina.';

  @override
  String get whoIsResponsible => 'Tko je odgovoran';

  @override
  String whoIsResponsibleBody(String clinic) {
    return 'Za vaše podatke odgovara $clinic. Salus ih obrađuje u ime klinike.';
  }

  @override
  String get readPrivacyNotice => 'Pročitajte obavijest o privatnosti';

  @override
  String get privacyNoticeFailed => 'Obavijest o privatnosti nije moguće otvoriti.';

  @override
  String get howToRevoke => 'Kako povući privolu';

  @override
  String get howToRevokeBody =>
      'Ovu privolu možete povući u bilo kojem trenutku u odjeljku Moje privole. Klinika tada više ne vidi vaš profil. Vaše ime i podatke o terminima koji su već održani zadržava kao dio svoje medicinske dokumentacije. Vaši budući termini u toj klinici otkazuju se.';

  @override
  String get agreeAndBook => 'Pristajem i rezerviram';

  @override
  String get revokeTitle => 'Povući privolu?';

  @override
  String revokeBody(String clinic) {
    return '$clinic više neće vidjeti vaš profil. Vaše ime i podatke o terminima koji su već održani zadržat će kao dio svoje medicinske dokumentacije. Vaši budući termini u klinici $clinic bit će otkazani.';
  }

  @override
  String get keepConsent => 'Zadrži privolu';

  @override
  String get revokeConsent => 'Povuci privolu';

  @override
  String get noConsents => 'Još niste dali nijednu privolu.';

  @override
  String get consentsIntro => 'S ovim klinikama dijelite svoj profil i termine radi svojeg liječenja.';

  @override
  String givenOn(String date) {
    return 'Dana $date';
  }

  @override
  String get consentActive => 'Aktivna';

  @override
  String revokedOn(String date) {
    return 'Povučena $date';
  }

  @override
  String get consentRevoked =>
      'Vaša je privola povučena. Vaši budući termini u ovoj klinici bit će otkazani i za svaki ćete primiti e-poruku.';

  @override
  String get consentAlreadyRevoked => 'Ova je privola već povučena.';

  @override
  String get joinConsultation => 'Pridruži se konzultaciji';

  @override
  String get joinFrom => 'Možete se pridružiti od 10 minuta prije početka.';

  @override
  String get checkTitle => 'Provjerite kameru i mikrofon';

  @override
  String get checkIntro => 'Liječnik će vas vidjeti i čuti kada se pridružite.';

  @override
  String get noSound => 'Zvuk se ne čuje. Provjerite je li mikrofon uključen i nije li utišan.';

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
      'Salus nema pristup vašoj kameri. Pristup možete dopustiti u postavkama uređaja ili se pridružiti bez videa.';

  @override
  String get microphoneDenied =>
      'Salus nema pristup vašem mikrofonu. Pristup možete dopustiti u postavkama uređaja. Bez mikrofona vas liječnik ne može čuti.';

  @override
  String get openSettings => 'Otvori postavke';

  @override
  String get noCamera => 'Kamera nije pronađena.';

  @override
  String get noMicrophone => 'Mikrofon nije pronađen.';

  @override
  String get waiting => 'Čeka se liječnik';

  @override
  String joinsYouHere(String name) {
    return '$name će vam se pridružiti ovdje. Ne zatvarajte ovaj zaslon.';
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
    return '$name više nije u konzultaciji. Može se ponovno pridružiti dok konzultacija ne završi.';
  }

  @override
  String get reconnecting => 'Veza je prekinuta. Ponovno povezivanje…';

  @override
  String get notRestored => 'Vezu nije bilo moguće ponovno uspostaviti.';

  @override
  String get reconnected => 'Veza je ponovno uspostavljena';

  @override
  String get consultationOver => 'Konzultacija je završena';

  @override
  String get closeScreen => 'Možete zatvoriti ovaj zaslon.';

  @override
  String get timeLimit => 'Vrijeme predviđeno za konzultaciju je isteklo i konzultacija je završena.';

  @override
  String get practitionerAbsent => 'Liječnik se nije pridružio';

  @override
  String get timePassed =>
      'Vrijeme za pridruživanje ovoj konzultaciji je isteklo. Klinika vidi da ste se pridružili. Klinika vas može kontaktirati, a možete i rezervirati novi termin.';

  @override
  String get withdrewConsent => 'Povukli ste privolu za ovu kliniku pa ste izašli iz konzultacije.';

  @override
  String get appointmentWasCancelled => 'Ovaj je termin otkazan pa je konzultacija završena.';

  @override
  String get windowNotOpen => 'Ova se konzultacija otvara 10 minuta prije početka.';

  @override
  String get consultationEnded => 'Ova je konzultacija završena.';

  @override
  String get joinNeedsConsent => 'Za pridruživanje je potrebna vaša privola za ovu kliniku.';

  @override
  String get consultationNotAvailable => 'Ova konzultacija nije dostupna.';

  @override
  String get cannotConnect => 'Videovezu nije bilo moguće pokrenuti. Provjerite internetsku vezu i pokušajte ponovno.';

  @override
  String get whyCameraAndMicrophone =>
      'Da biste se pridružili videokonzultaciji, Salus treba pristup vašoj kameri i mikrofonu. Konzultacija se ne snima.';

  @override
  String get transcriptionOn => 'Transkripcija je uključena';

  @override
  String get toFillIn => 'Za ispunjavanje';

  @override
  String get inProgress => 'U tijeku';

  @override
  String get submittedStatus => 'Poslan';

  @override
  String get closedStatus => 'Zatvoren';

  @override
  String get fillIn => 'Ispuni';

  @override
  String get continueFilling => 'Nastavi';

  @override
  String get viewAnswers => 'Pogledaj odgovore';

  @override
  String get submitBeforeStart => 'Molimo vas da ga pošaljete prije početka termina.';

  @override
  String get questionnaireClosed => 'Termin je počeo pa ovaj upitnik više nije moguće ispuniti.';

  @override
  String savedAsYouGo(String clinic) {
    return 'Vaši se odgovori spremaju dok ispunjavate upitnik. $clinic ih vidi tek nakon što ih pošaljete.';
  }

  @override
  String get saved => 'Spremljeno';

  @override
  String get saving => 'Spremanje…';

  @override
  String get notSaved => 'Nije spremljeno. Provjerite internetsku vezu. Aplikacija će pokušati ponovno.';

  @override
  String get requiredQuestion => 'Obvezno';

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
  String get questionRequired => 'Ovo je pitanje obvezno.';

  @override
  String get enterWholeNumber => 'Unesite cijeli broj.';

  @override
  String get enterNumber => 'Unesite broj.';

  @override
  String get numberTooLarge => 'Ovaj je broj prevelik.';

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
    return 'Nakon slanja više ih ne možete mijenjati. Vidjet će ih vaš liječnik u klinici $clinic.';
  }

  @override
  String get cancel => 'Odustani';

  @override
  String get requiredMissing => 'Na neka obvezna pitanja niste odgovorili.';

  @override
  String get answersSubmitted => 'Odgovori su poslani';

  @override
  String get answersSubmittedBody =>
      'Vaš liječnik može ih vidjeti prije termina. Možete ih pogledati u odjeljku Moji termini.';

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
  String get leaveUnsaved => 'Vaš posljednji odgovor još nije spremljen. Želite li ipak izaći?';

  @override
  String get stay => 'Ostani';

  @override
  String get fillingNeedsConsent => 'Za ispunjavanje ovog upitnika potrebna je vaša privola za ovu kliniku.';

  @override
  String get pastDue => 'Termin je počeo pa ovaj upitnik više nije moguće mijenjati.';

  @override
  String get withdrawnByClinic => 'Klinika je povukla ovaj upitnik. Ne morate ga ispunjavati.';

  @override
  String get alreadySubmitted => 'Već ste poslali ovaj upitnik.';

  @override
  String get changedElsewhere => 'Ovi su odgovori promijenjeni na drugom uređaju. Prikazani su najnoviji odgovori.';

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
  String get recordVideo => 'Snimi videozapis';

  @override
  String get chooseFromPhotos => 'Odaberi iz galerije';

  @override
  String get chooseDocument => 'Odaberi dokument';

  @override
  String get preparing => 'Priprema…';

  @override
  String uploadingPercent(int n) {
    return 'Prijenos… $n %';
  }

  @override
  String get checkingFile => 'Provjera datoteke…';

  @override
  String get added => 'Dodano';

  @override
  String get notUploaded => 'Nije preneseno. Provjerite internetsku vezu.';

  @override
  String get remove => 'Ukloni';

  @override
  String get removeFile => 'Ukloni datoteku';

  @override
  String get removeFileTitle => 'Ukloniti ovu datoteku?';

  @override
  String get removeFileBody => 'Bit će izbrisana iz vaših odgovora.';

  @override
  String get fileRejected => 'Ova datoteka nije prošla sigurnosnu provjeru i izbrisana je. Odaberite drugu datoteku.';

  @override
  String fileTooLarge(int max) {
    return 'Ova je datoteka veća od $max MB.';
  }

  @override
  String get fileTypeRefused => 'Ovu vrstu datoteke nije moguće dodati uz ovo pitanje.';

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
      'Datoteke u ovom upitniku ukupno su dosegnule 500 MB. Uklonite neku datoteku kako biste dodali novu.';

  @override
  String get fileNotUploaded => 'Datoteku nije bilo moguće prenijeti. Pokušajte ponovno.';

  @override
  String get cannotPrepareFile => 'Ovu datoteku nije bilo moguće pripremiti za prijenos. Odaberite drugu datoteku.';

  @override
  String get filesBeingChecked => 'Neke se datoteke još provjeravaju. Upitnik možete poslati kada budu spremne.';

  @override
  String get leaveUploading => 'Datoteka se još prenosi. Želite li ipak izaći? Prijenos će se prekinuti.';

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
  String get videoKind => 'Videozapis';

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
    return 'Prijenos datoteke $title, $n posto';
  }

  @override
  String fileLabel(String kind, String title, String status) {
    return '$kind, $title, $status';
  }

  @override
  String get allowCamera => 'Dopustite pristup kameri';

  @override
  String get whyCameraForFiles =>
      'Salus treba pristup kameri kako biste snimili fotografiju ili videozapis koji dodajete u ovaj upitnik.';

  @override
  String get continueLabel => 'Nastavi';

  @override
  String get cameraDeniedForFiles =>
      'Salus nema pristup vašoj kameri. Pristup možete dopustiti u postavkama uređaja ili umjesto toga odabrati datoteku.';

  @override
  String get photosDenied => 'Salus nema pristup vašim fotografijama. Pristup možete dopustiti u postavkama uređaja.';

  @override
  String cameraPhotoTitle(String date, String time) {
    return 'Fotografija, $date, $time';
  }

  @override
  String cameraVideoTitle(String date, String time) {
    return 'Videozapis, $date, $time';
  }

  @override
  String get specialty394814009 => 'Obiteljska medicina';

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
  String get languageUnavailable => 'Ovaj jezik više nije dostupan za ovaj upitnik. Odaberite drugi.';

  @override
  String specialtyAtClinic(String specialty, String clinic) {
    return '$specialty, $clinic';
  }

  @override
  String get documents => 'Dokumenti';

  @override
  String get noDocuments =>
      'Za ovaj termin još nema potpisanih dokumenata. Dokument se ovdje pojavljuje kada ga vaš liječnik potpiše.';

  @override
  String documentSignedOn(String date) {
    return 'Potpisano $date';
  }

  @override
  String documentLanguage(String language) {
    return 'Jezik: $language';
  }

  @override
  String get sharePdf => 'Spremi ili podijeli PDF';

  @override
  String get transcript => 'Transkript';

  @override
  String get transcriptNote =>
      'Konzultacija onako kako je automatski transkribirana i prevedena. Može sadržavati pogreške.';

  @override
  String get transcriptEmpty => 'U ovoj konzultaciji ništa nije transkribirano.';

  @override
  String get speakerYou => 'Vi';

  @override
  String get speakerPractitioner => 'Liječnik';

  @override
  String get shareTranscript => 'Podijeli transkript';

  @override
  String transcriptTranslation(String text) {
    return 'Prijevod: $text';
  }
}
