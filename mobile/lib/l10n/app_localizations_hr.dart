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
  String get unreachable =>
      'Salus trenutačno nije dostupan. Provjerite internetsku vezu i pokušajte ponovno.';

  @override
  String get wrongCredentials => 'Adresa e-pošte ili lozinka nisu točni.';

  @override
  String get wrongCode =>
      'Kod nije točan. Provjerite vrijeme na uređaju i pokušajte ponovno.';

  @override
  String get challengeEnded =>
      'Prijava je istekla. Ponovno unesite adresu e-pošte i lozinku.';

  @override
  String get emailTaken =>
      'Za ovu adresu e-pošte već postoji račun. Prijavite se.';

  @override
  String get invalidEmail => 'Unesite ispravnu adresu e-pošte.';

  @override
  String get invalidPassword => 'Lozinka mora imati od 12 do 128 znakova.';

  @override
  String get missingPassword => 'Unesite lozinku.';

  @override
  String get invalidCode => 'Kod ima 6 znamenki.';

  @override
  String get staffOnly =>
      'Ova je aplikacija namijenjena pacijentima. Osoblje klinike koristi Salus web-aplikaciju.';

  @override
  String get accountReady =>
      'Vaš je račun spreman. Prijavite se kako biste nastavili.';

  @override
  String get signOutFailed =>
      'Odjava nije uspjela. Provjerite internetsku vezu i pokušajte ponovno.';

  @override
  String get tooManyAttemptsLater =>
      'Previše pokušaja. Pokušajte ponovno kasnije.';

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
  String get codePrompt =>
      'Unesite šesteroznamenkasti kod iz aplikacije za autentifikaciju.';

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
  String get completeProfileFirst =>
      'Dopunite profil prije rezervacije termina.';

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
  String get birthDateRange =>
      'Unesite datum rođenja u prošlosti, ne raniji od 1900. godine.';

  @override
  String invalidPhone(String example) {
    return 'Unesite broj telefona s pozivnim brojem države, na primjer $example.';
  }

  @override
  String get profileChanged =>
      'Vaš je profil u međuvremenu promijenjen. Provjerite ga i ponovno spremite.';

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
  String get slotTaken =>
      'Ovaj termin više nije slobodan. Odaberite drugi termin.';

  @override
  String get consentNeeded =>
      'Za rezervaciju termina u ovoj klinici potrebna je vaša privola.';

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
  String get cancelWindowHint =>
      'Termin možete otkazati u aplikaciji najkasnije 24 sata prije početka.';

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
  String get whatForBody =>
      'Isključivo za vaše liječenje u ovoj klinici: za pripremu i održavanje vaših termina.';

  @override
  String get whoIsResponsible => 'Tko je odgovoran';

  @override
  String whoIsResponsibleBody(String clinic) {
    return 'Za vaše podatke odgovara $clinic. Salus ih obrađuje u ime klinike.';
  }

  @override
  String get readPrivacyNotice => 'Pročitajte obavijest o privatnosti';

  @override
  String get privacyNoticeFailed =>
      'Obavijest o privatnosti nije moguće otvoriti.';

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
    return '$clinic više neće vidjeti vaš profil. Vaše ime i podatke o terminima koji su već održani zadržat će kao dio svoje medicinske dokumentacije. Vaši budući termini u klinici $clinic bit će otkazani. Prestaje i dijeljenje vaše povijesti iz drugih klinika s klinikom $clinic; ponovni pristanak ga ne vraća.';
  }

  @override
  String get keepConsent => 'Zadrži privolu';

  @override
  String get revokeConsent => 'Povuci privolu';

  @override
  String get noConsents => 'Još niste dali nijednu privolu.';

  @override
  String get consentsIntro =>
      'S ovim klinikama dijelite svoj profil i termine radi svojeg liječenja.';

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
  String get noSound =>
      'Zvuk se ne čuje. Provjerite je li mikrofon uključen i nije li utišan.';

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
  String get timeLimit =>
      'Vrijeme predviđeno za konzultaciju je isteklo i konzultacija je završena.';

  @override
  String get practitionerAbsent => 'Liječnik se nije pridružio';

  @override
  String get timePassed =>
      'Vrijeme za pridruživanje ovoj konzultaciji je isteklo. Klinika vidi da ste se pridružili. Klinika vas može kontaktirati, a možete i rezervirati novi termin.';

  @override
  String get withdrewConsent =>
      'Povukli ste privolu za ovu kliniku pa ste izašli iz konzultacije.';

  @override
  String get appointmentWasCancelled =>
      'Ovaj je termin otkazan pa je konzultacija završena.';

  @override
  String get windowNotOpen =>
      'Ova se konzultacija otvara 10 minuta prije početka.';

  @override
  String get consultationEnded => 'Ova je konzultacija završena.';

  @override
  String get joinNeedsConsent =>
      'Za pridruživanje je potrebna vaša privola za ovu kliniku.';

  @override
  String get consultationNotAvailable => 'Ova konzultacija nije dostupna.';

  @override
  String get cannotConnect =>
      'Videovezu nije bilo moguće pokrenuti. Provjerite internetsku vezu i pokušajte ponovno.';

  @override
  String get whyCameraAndMicrophone =>
      'Da biste se pridružili videokonzultaciji, Salus treba pristup vašoj kameri i mikrofonu. Konzultacija se ne snima ni kao zvuk ni kao video. Ako pristanete, ono što se izgovori transkribira se i prevodi.';

  @override
  String get transcriptionOn => 'Transkripcija je uključena';

  @override
  String get transcriptionTitle => 'Transkripcija i prijevod';

  @override
  String transcriptionIntro(String clinic) {
    return 'U vašim videokonzultacijama s klinikom $clinic ono što vi i liječnik kažete može se transkribirati dok govorite i prevesti, kako bi vas liječnik mogao pratiti na vašem jeziku.';
  }

  @override
  String get transcriptionAssistance =>
      'Na temelju transkripta liječnik dobiva i pomoć umjetne inteligencije (AI): prijedloge tijekom konzultacije i nacrte dokumenata. To su samo prijedlozi; liječnik ih provjerava i donosi svaku odluku.';

  @override
  String transcriptionKept(String clinic) {
    return 'Transkript se čuva uz vašu medicinsku dokumentaciju u klinici $clinic. Konzultacija se ne snima ni kao zvuk ni kao video.';
  }

  @override
  String get transcriptionDecline =>
      'Ako ne pristanete, konzultacija se odvija kao i obično, bez transkripcije, prijevoda i pomoći umjetne inteligencije.';

  @override
  String get transcriptionWithdraw =>
      'Privolu možete povući u bilo kojem trenutku, i tijekom konzultacije: dodirnite „Transkripcija je uključena” ili otvorite Moje privole. Transkripcija prestaje odmah; ono što je dotad transkribirano ostaje u vašoj dokumentaciji.';

  @override
  String get transcriptionAgree => 'Prihvati i nastavi';

  @override
  String get transcriptionWithout => 'Nastavi bez transkripcije';

  @override
  String get stopTranscriptionTitle => 'Zaustaviti transkripciju?';

  @override
  String stopTranscriptionBody(String clinic) {
    return 'Transkripcija, prijevod i pomoć umjetne inteligencije odmah prestaju u vašim konzultacijama s klinikom $clinic. Konzultacija koja je u tijeku nastavlja se bez njih, a vaši termini ostaju nepromijenjeni. Ono što je dosad transkribirano ostaje u vašoj dokumentaciji.';
  }

  @override
  String get keepTranscription => 'Zadrži transkripciju';

  @override
  String get stopTranscription => 'Zaustavi transkripciju';

  @override
  String get transcriptionStopped =>
      'Transkripcija je isključena. Vaše konzultacije s ovom klinikom odvijaju se bez nje.';

  @override
  String get consentForCare => 'Dijeljenje radi liječenja';

  @override
  String get consentForTranscription =>
      'Transkripcija i pomoć umjetne inteligencije';

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
  String get submitBeforeStart =>
      'Molimo vas da ga pošaljete prije početka termina.';

  @override
  String get questionnaireClosed =>
      'Termin je počeo pa ovaj upitnik više nije moguće ispuniti.';

  @override
  String savedAsYouGo(String clinic) {
    return 'Vaši se odgovori spremaju dok ispunjavate upitnik. $clinic ih vidi tek nakon što ih pošaljete.';
  }

  @override
  String get saved => 'Spremljeno';

  @override
  String get saving => 'Spremanje…';

  @override
  String get notSaved =>
      'Nije spremljeno. Provjerite internetsku vezu. Aplikacija će pokušati ponovno.';

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
  String get leaveUnsaved =>
      'Vaš posljednji odgovor još nije spremljen. Želite li ipak izaći?';

  @override
  String get stay => 'Ostani';

  @override
  String get fillingNeedsConsent =>
      'Za ispunjavanje ovog upitnika potrebna je vaša privola za ovu kliniku.';

  @override
  String get pastDue =>
      'Termin je počeo pa ovaj upitnik više nije moguće mijenjati.';

  @override
  String get withdrawnByClinic =>
      'Klinika je povukla ovaj upitnik. Ne morate ga ispunjavati.';

  @override
  String get alreadySubmitted => 'Već ste poslali ovaj upitnik.';

  @override
  String get changedElsewhere =>
      'Ovi su odgovori promijenjeni na drugom uređaju. Prikazani su najnoviji odgovori.';

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
  String get fileRejected =>
      'Ova datoteka nije prošla sigurnosnu provjeru i izbrisana je. Odaberite drugu datoteku.';

  @override
  String fileTooLarge(int max) {
    return 'Ova je datoteka veća od $max MB.';
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
      'Datoteke u ovom upitniku ukupno su dosegnule 500 MB. Uklonite neku datoteku kako biste dodali novu.';

  @override
  String get fileNotUploaded =>
      'Datoteku nije bilo moguće prenijeti. Pokušajte ponovno.';

  @override
  String get cannotPrepareFile =>
      'Ovu datoteku nije bilo moguće pripremiti za prijenos. Odaberite drugu datoteku.';

  @override
  String get filesBeingChecked =>
      'Neke se datoteke još provjeravaju. Upitnik možete poslati kada budu spremne.';

  @override
  String get leaveUploading =>
      'Datoteka se još prenosi. Želite li ipak izaći? Prijenos će se prekinuti.';

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
  String get photosDenied =>
      'Salus nema pristup vašim fotografijama. Pristup možete dopustiti u postavkama uređaja.';

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
  String get transcriptEmpty =>
      'U ovoj konzultaciji ništa nije transkribirano.';

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
  String get chatDoctorUnknown => 'Vaš liječnik';

  @override
  String get chatMessageHint => 'Napišite poruku';

  @override
  String get chatSend => 'Pošalji';

  @override
  String get chatAttach => 'Dodaj datoteku';

  @override
  String get chatSendFailed =>
      'Poruka nije poslana. Provjerite internetsku vezu i pokušajte ponovno.';

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
          'Već imate $count datoteka koje čekaju slanje. Neposlane datoteke uklanjaju se nakon 24 sata; pokušajte ponovno kasnije.',
      few:
          'Već imate $count datoteke koje čekaju slanje. Neposlane datoteke uklanjaju se nakon 24 sata; pokušajte ponovno kasnije.',
      one:
          'Već imate $count datoteku koja čeka slanje. Neposlane datoteke uklanjaju se nakon 24 sata; pokušajte ponovno kasnije.',
    );
    return '$_temp0';
  }

  @override
  String get chatOpenFailed =>
      'Datoteku nije moguće otvoriti. Pokušajte ponovno.';

  @override
  String get chatEndedSession =>
      'Vaša je sesija istekla. Ponovno se prijavite da biste vidjeli nove poruke.';

  @override
  String get chatEnded =>
      'Nove poruke u ovom razgovoru više se ne mogu prikazati.';

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
          'Klinika je trenutačno izvan radnog vremena za poruke. I dalje možete pisati; obično odgovara u roku od $hours sati.',
      few:
          'Klinika je trenutačno izvan radnog vremena za poruke. I dalje možete pisati; obično odgovara u roku od $hours sata.',
      one:
          'Klinika je trenutačno izvan radnog vremena za poruke. I dalje možete pisati; obično odgovara u roku od $hours sata.',
    );
    return '$_temp0';
  }

  @override
  String chatSystemInvited(String reason) {
    String _temp0 = intl.Intl.selectLogic(reason, {
      'second_opinion': 'Liječnik iz druge klinike pozvan je u ovaj razgovor radi drugog mišljenja.',
      'specialist_review': 'Liječnik iz druge klinike pozvan je u ovaj razgovor radi specijalističkog pregleda.',
      'handover': 'Liječnik iz druge klinike pozvan je u ovaj razgovor da preuzme liječenje.',
      'other': 'Liječnik iz druge klinike pozvan je u ovaj razgovor.',
    });
    return '$_temp0';
  }

  @override
  String chatSystemAdded(String reason) {
    String _temp0 = intl.Intl.selectLogic(reason, {
      'second_opinion':
          'Liječnik se pridružio razgovoru radi drugog mišljenja.',
      'specialist_review':
          'Liječnik se pridružio razgovoru radi specijalističkog pregleda.',
      'handover': 'Liječnik se pridružio razgovoru da preuzme liječenje.',
      'other': 'Liječnik se pridružio razgovoru.',
    });
    return '$_temp0';
  }

  @override
  String get chatSystemRemoved => 'Liječnik više ne sudjeluje u razgovoru.';

  @override
  String get chatSystemGranted =>
      'Dopustili ste liječniku iz druge klinike da se pridruži razgovoru.';

  @override
  String get chatSystemDeclined => 'Odbili ste liječnika iz druge klinike.';

  @override
  String get chatSystemOther => 'Razgovor je ažuriran.';

  @override
  String get chatInviteTitle =>
      'Liječnik iz druge klinike traži pristup razgovoru';

  @override
  String chatInviteBody(String reason) {
    String _temp0 = intl.Intl.selectLogic(reason, {
      'second_opinion': 'Vaša klinika traži da se liječnik iz druge klinike pridruži ovom razgovoru radi drugog mišljenja.',
      'specialist_review': 'Vaša klinika traži da se liječnik iz druge klinike pridruži ovom razgovoru radi specijalističkog pregleda.',
      'handover': 'Vaša klinika traži da se liječnik iz druge klinike pridruži ovom razgovoru da preuzme vaše liječenje.',
      'other': 'Vaša klinika traži da se liječnik iz druge klinike pridruži ovom razgovoru.',
    });
    return '$_temp0';
  }

  @override
  String chatInviteWho(String doctor, String clinic) {
    return 'Liječnik: $doctor, $clinic';
  }

  @override
  String get chatInviteUnresolved =>
      'Ime i klinika pozvanog liječnika nisu se mogli učitati. Pristup možete odobriti tek kada budu prikazani.';

  @override
  String chatInviteSees(String history) {
    String _temp0 = intl.Intl.selectLogic(history, {
      'full': 'Ako pristanete, liječnik vidi cijeli razgovor, uključujući ranije poruke, i može pisati u njemu.',
      'from': 'Ako pristanete, liječnik vidi razgovor od poruke koju je odabrala vaša klinika nadalje i može pisati u njemu.',
      'other': 'Ako pristanete, liječnik vidi razgovor i može pisati u njemu.',
    });
    return '$_temp0';
  }

  @override
  String get chatInviteChoice =>
      'Možete odbiti. Ako pristanete, pristup liječnika možete poslije opozvati u ovom razgovoru; prestaje odmah.';

  @override
  String get chatGrant => 'Odobri pristup';

  @override
  String get chatDecline => 'Odbij';

  @override
  String get chatGranted =>
      'Pristup je odobren. Možete ga opozvati u ovom razgovoru u bilo kojem trenutku.';

  @override
  String get chatDeclined =>
      'Odbijeno. Liječnik se ne pridružuje ovom razgovoru.';

  @override
  String get chatRevoke => 'Opozovi pristup liječnika iz druge klinike';

  @override
  String chatGrantedTo(String doctor, String clinic) {
    return 'Pristup je odobren: $doctor, $clinic.';
  }

  @override
  String get chatGrantedToUnknown =>
      'Pristup je odobren liječniku iz druge klinike.';

  @override
  String get chatRevokeBody =>
      'Liječnik iz druge klinike odmah gubi pristup ovom razgovoru. Već napisane poruke ostaju u njemu.';

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
      'Klinike kojima ste dali privolu vide alergije iz vašeg profila kada imate termin kod njih.';

  @override
  String get medicationsIntro =>
      'Klinike kojima ste dali privolu vide lijekove iz vašeg profila kada imate termin kod njih.';

  @override
  String get noAllergies => 'Niste dodali nijednu alergiju.';

  @override
  String get noMedications => 'Niste dodali nijedan lijek.';

  @override
  String get addAllergy => 'Dodaj alergiju';

  @override
  String get addMedication => 'Dodaj lijek';

  @override
  String get changeAllergy => 'Promjena alergije';

  @override
  String get changeMedication => 'Promjena lijeka';

  @override
  String get change => 'Promijeni';

  @override
  String get allergySubstance => 'Tvar ili namirnica';

  @override
  String get medicationName => 'Lijek';

  @override
  String get termSearchHint =>
      'Počnite tipkati za pretraživanje ili upišite svojim riječima.';

  @override
  String get allergyCategory => 'Vrsta (nije obvezno)';

  @override
  String get notSpecified => 'Nije navedeno';

  @override
  String get categoryFood => 'Hrana';

  @override
  String get categoryMedication => 'Lijek';

  @override
  String get categoryEnvironment => 'Okoliš';

  @override
  String get categoryBiologic => 'Biološki preparat';

  @override
  String get reactionOptional => 'Reakcija (nije obvezno)';

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
  String get confirmedByClinician => 'Potvrdio liječnik';

  @override
  String get recordedByClinic =>
      'Unijela klinika tijekom jednog od vaših posjeta';

  @override
  String get confirmedChangeNote =>
      'Ako promijenite ovaj unos, više neće biti potvrđen od liječnika.';

  @override
  String get nameRequired => 'Unesite naziv, najviše 200 znakova.';

  @override
  String get addToProfile => 'Dodaj u moj profil';

  @override
  String get addToProfileTitle => 'Dodati u vaš profil?';

  @override
  String get addToProfileBody =>
      'Tada će ga vidjeti svaka klinika kojoj ste dali privolu, bez naziva klinike koja ga je unijela.';

  @override
  String get addedToProfile => 'Sada je u vašem profilu.';

  @override
  String get removeEntryTitle => 'Ukloniti ovaj unos?';

  @override
  String get removeEntryBody =>
      'Više se neće prikazivati ni vama ni ijednoj klinici. Ako ga je klinika unijela ili potvrdila, ostaje u vašoj medicinskoj dokumentaciji, označen kao uklonjen.';

  @override
  String get entryRemoved => 'Unos je uklonjen.';

  @override
  String get entrySaved => 'Spremljeno u vaš profil.';

  @override
  String get entryChanged =>
      'Ovaj je unos u međuvremenu promijenjen. Prikazana je najnovija verzija.';

  @override
  String get entryInvalid =>
      'Neki podaci nisu ispravni. Provjerite ih i pokušajte ponovno.';

  @override
  String get entryNotAllowed => 'Ne možete promijeniti ovaj unos.';

  @override
  String get entryNotFound => 'Ovaj unos više nije dostupan.';

  @override
  String get doseHeading => 'Doza (nije obvezno)';

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
  String get unitIu => 'internacionalne jedinice (IU)';

  @override
  String get unitTablet => 'tablete';

  @override
  String get unitCapsule => 'kapsule';

  @override
  String get unitDrop => 'kapi';

  @override
  String get unitPuff => 'udasi (inhalator)';

  @override
  String get unitPatch => 'flasteri';

  @override
  String get doseBoth => 'Unesite i količinu i jedinicu ili ništa od toga.';

  @override
  String get numberAboveZero => 'Unesite broj veći od 0.';

  @override
  String get howOftenHeading => 'Koliko često (nije obvezno)';

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
  String get periodWeeks => 'tjedana';

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
    return '$value IU';
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
      other: '$value udaha',
      few: '$value udaha',
      one: '$value udah',
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
    return '$frequency× tjedno';
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
      other: '$frequency× svakih $every tjedana',
      few: '$frequency× svaka $every tjedna',
      one: '$frequency× svaki $every tjedan',
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
  String get myHistory => 'Moja povijest';

  @override
  String get historyEmpty =>
      'Vaša je povijest još prazna. Ovdje se pojavljuju vaši posjeti, dokumenti i odgovori.';

  @override
  String get historyNoMatch =>
      'Ništa u vašoj povijesti ne odgovara ovim filtrima.';

  @override
  String get historyClinic => 'Klinika';

  @override
  String get historyAllClinics => 'Sve klinike';

  @override
  String get historyVisits => 'Posjeti';

  @override
  String get historyVisit => 'Posjet';

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
      'Ostatak vaše povijesti nije se mogao učitati.';

  @override
  String get historyOpenFailed =>
      'Ova se stavka nije mogla otvoriti. Pokušajte ponovno.';

  @override
  String get sharingToggle =>
      'Podijeli moju povijest iz drugih klinika s ovom klinikom';

  @override
  String sharingIntro(String clinic) {
    return 'Označite ranije posjete koje $clinic smije vidjeti. Za svaku označenu posjetu liječnici klinike $clinic kod kojih imate termin vide ono što je ovdje prikazano, s dokumentima posjete, odgovorima na upitnike i datotekama, alergijama i lijekovima zabilježenima u njoj te svime što se posjeti naknadno doda. Posjete koje ne označite ostaju skrivene od $clinic.';
  }

  @override
  String get sharingChangeLater =>
      'Posjete možete dodati ili ukloniti, ili prestati dijeliti, u bilo kojem trenutku u odjeljku Podijeljena povijest. Svaka promjena vrijedi odmah.';

  @override
  String get sharingNoVisits =>
      'Nemate ranijih posjeta u drugim klinikama koje biste mogli podijeliti.';

  @override
  String get sharingSelectAll => 'Označi sve';

  @override
  String get sharingSelectNone => 'Poništi sve';

  @override
  String get sharedHistory => 'Podijeljena povijest';

  @override
  String get sharedHistoryIntro =>
      'Klinike kojima ste dali privolu i koliko vaših posjeta iz drugih klinika svaka od njih vidi.';

  @override
  String get sharedHistoryNone => 'Još niste dali privolu nijednoj klinici.';

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
  String get sharingSaved => 'Vaš je izbor spremljen i vrijedi odmah.';

  @override
  String get sharingStop => 'Prestani dijeliti';

  @override
  String get sharingStopTitle => 'Prestati dijeliti vašu povijest?';

  @override
  String sharingStopBody(String clinic) {
    return '$clinic više neće vidjeti nijednu posjetu iz drugih klinika koju ste s njom podijelili. To vrijedi odmah.';
  }

  @override
  String get sharingKeep => 'Nastavi dijeliti';

  @override
  String get sharingStopped =>
      'Više ne dijelite svoju povijest s ovom klinikom.';

  @override
  String get sharingChanged =>
      'Vaše su se podijeljene posjete u međuvremenu promijenile. Provjerite ih i ponovno spremite.';

  @override
  String get bookedNotShared =>
      'Vaš je termin rezerviran, ali vaša povijest nije mogla biti podijeljena. Možete je podijeliti u odjeljku Podijeljena povijest.';

  @override
  String get consentForHistory => 'Posjete podijeljene iz drugih klinika';

  @override
  String get reminders => 'Podsjetnici';

  @override
  String get remindersIntro =>
      'Odaberite kako vas klinike podsjećaju na termine.';

  @override
  String get remindersPush => 'Push obavijesti';

  @override
  String get remindersEmail => 'E-pošta';

  @override
  String get remindersSms => 'SMS';

  @override
  String get remindersSmsHint =>
      'Podsjetnici i ponude s liste čekanja SMS-om na broj telefona iz vašeg profila, samo u klinikama koje nude SMS. Nikada za oglašavanje.';

  @override
  String get quietHours => 'Tihi sati';

  @override
  String get quietHoursHint =>
      'Push obavijesti i SMS čekaju do kraja tihih sati. Ako bi tada stigli manje od 30 minuta prije termina, ne šalju se. E-pošta se ne zadržava.';

  @override
  String get quietHoursFrom => 'Od';

  @override
  String get quietHoursUntil => 'Do';

  @override
  String get remindersSaved => 'Vaš je izbor spremljen.';

  @override
  String get remindersPhoneRequired =>
      'Za SMS je potreban broj telefona u profilu. Dodajte ga u odjeljku Moj profil pa pokušajte ponovno.';

  @override
  String get remindersInvalid =>
      'Ove postavke nije bilo moguće spremiti. Provjerite ih i pokušajte ponovno.';

  @override
  String get confirmAttendance => 'Potvrdi dolazak';

  @override
  String get appointmentConfirmedStatus => 'Potvrđen';

  @override
  String get appointmentConfirmedNotice => 'Vaš je termin potvrđen.';

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
    return 'Sada rezervirano: $date u $time';
  }

  @override
  String get rescheduleConfirmTitle => 'Premjestiti termin?';

  @override
  String rescheduleConfirmBody(String date, String time) {
    return 'Vaš će termin biti premješten na $date u $time. Dosadašnje vrijeme bit će oslobođeno.';
  }

  @override
  String get moveAppointment => 'Premjesti termin';

  @override
  String get appointmentMoved => 'Vaš je termin premješten na novo vrijeme.';

  @override
  String get rescheduleWindowClosed =>
      'Vrijeme ovog termina više se ne može promijeniti u aplikaciji. Obratite se klinici.';

  @override
  String get rescheduleLimit =>
      'Vrijeme ovog termina promijenjeno je onoliko puta koliko klinika dopušta.';

  @override
  String get notReschedulable =>
      'Vrijeme ovog termina ne može se promijeniti u aplikaciji.';

  @override
  String get waitlistTitle => 'Moja lista čekanja';

  @override
  String get waitlistJoinButton =>
      'Nema odgovarajućeg termina? Prijavite se na listu čekanja';

  @override
  String get waitlistJoinTitle => 'Prijava na listu čekanja';

  @override
  String get waitlistExplained =>
      'Kad se oslobodi termin kod ovog liječnika, nudi se pacijentima na čekanju, jednom po jednom. Dobit ćete obavijest i termin možete naručiti u aplikaciji dok je rezerviran za vas.';

  @override
  String get waitlistHowLong => 'Koliko dugo?';

  @override
  String waitlistDays(int days) {
    String _temp0 = intl.Intl.pluralLogic(
      days,
      locale: localeName,
      other: '$days dana',
      few: '$days dana',
      one: '$days dan',
    );
    return '$_temp0';
  }

  @override
  String get waitlistWhen => 'U koje doba dana?';

  @override
  String get waitlistAnyTime => 'Bilo kada';

  @override
  String get waitlistMornings => 'Ujutro, od 08:00 do 12:00';

  @override
  String get waitlistAfternoons => 'Poslijepodne, od 12:00 do 17:00';

  @override
  String get waitlistJoin => 'Prijavi se na listu čekanja';

  @override
  String get waitlistJoined => 'Na listi čekanja ste.';

  @override
  String get waitlistEmpty => 'Niste ni na jednoj listi čekanja.';

  @override
  String get waitlistAPractitioner => 'Liječnik';

  @override
  String waitlistUntil(String date) {
    return 'Čekate do $date';
  }

  @override
  String get waitlistOfferTitle => 'Oslobodio se termin za vas';

  @override
  String waitlistHeldUntil(String time) {
    return 'Rezerviran za vas do $time';
  }

  @override
  String get waitlistAcceptEffect =>
      'Prihvaćanjem naručujete ovaj termin. Kad plaćanje bude dostupno u aplikaciji Salus, prihvaćanjem ćete ga i platiti.';

  @override
  String get waitlistAccept => 'Prihvati i naruči';

  @override
  String get waitlistDecline => 'Odbij';

  @override
  String get waitlistBooked =>
      'Vaš je termin naručen. Pronaći ćete ga u odjeljku Moji termini.';

  @override
  String get waitlistDeclined =>
      'Odbili ste termin. Ostajete na listi čekanja.';

  @override
  String get waitlistWithdraw => 'Napusti listu čekanja';

  @override
  String get waitlistWithdrawn => 'Napustili ste listu čekanja.';

  @override
  String get waitlistOff => 'Ova ustanova nema listu čekanja.';

  @override
  String get waitlistTooMany =>
      'Već ste pet puta na listi čekanja ove ustanove. Najprije napustite jednu od prijava.';

  @override
  String get waitlistOfferExpired =>
      'Ovaj termin više nije rezerviran za vas. Ostajete na listi čekanja.';
}
