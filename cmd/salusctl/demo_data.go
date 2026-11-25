package main

import "fmt"

// The demo clinic's synthetic data (docs/demo/SCRIPT.md). Every name is plainly made up: the family names are
// Bosnian words for "example", "sample", "test" and "trial" turned into surnames, the e-mail addresses are under
// example.test (RFC 2606), the phone numbers start +387 000, which no operator has, and the addresses are on
// "Ulica Primjera" (Example Street). Nothing here is, or is taken from, a real person, clinic or record.
const (
	demoClinicName = "Poliklinika Primjer (demo)"
	demoDomain     = "demo.example.test"
	// demoPassword is every demo account's password: the accounts are synthetic and the seed runs only on a
	// developer's stack.
	demoPassword = "demo-lozinka-primjer"
	demoLanguage = "bs"
	demoCurrency = "BAM"
	// demoPlanPrice is the default plan's placeholder price per active practitioner and month (2.00 BAM), set only
	// when the owner has set no plan: the real price is the owner's ([PRICE]).
	demoPlanPrice = 200
)

// demoAdminEmail is the clinic admin's address: the clinic's first admin, invited by salusctl as create-clinic does.
const demoAdminEmail = "uprava@" + demoDomain

// demoReceptionEmail is the receptionist's address.
const demoReceptionEmail = "recepcija@" + demoDomain

// demoHours is one weekly rule of a practitioner's hours: ISO weekday (1 Monday … 7 Sunday), the local start and end,
// the visit mode, and for an in-person rule the location's and the room's index in demoLocations.
type demoHours struct {
	start, end, mode string
	day              int
	location, room   int
}

// demoPractitioner is one of the four practitioners: their account, name, specialty (SNOMED CT, from
// practitionerrole.Specialties), the service a visit with them is for, in person and by video, and their hours.
type demoPractitioner struct {
	email, given, family, specialty string
	inPerson, video                 string // service codes of demoServices
	hours                           []demoHours
}

// demoPractitioners are the clinic's four practitioners. The evening video rules are the "online hours" a doctor
// works from home after the clinic closes; every day of the week has someone's hours, so the desk's day is never
// empty whenever the demo is seeded.
var demoPractitioners = []demoPractitioner{
	{
		email: "emir.primjerovic@" + demoDomain, given: "Emir", family: "Primjerović", specialty: "394814009",
		inPerson: "OP-01", video: "VK-01",
		hours: append(weekdays(1, 5, "08:00", "14:00", "in-person", 0, 0),
			append(weekdays(1, 4, "18:00", "20:00", "video", -1, -1),
				demoHours{day: 7, start: "09:00", end: "12:00", mode: "in-person", location: 0, room: 0})...),
	},
	{
		email: "selma.uzorkovic@" + demoDomain, given: "Selma", family: "Uzorković", specialty: "394579002",
		inPerson: "KP-01", video: "VK-01",
		hours: []demoHours{
			{day: 1, start: "08:00", end: "14:00", mode: "in-person", location: 0, room: 1},
			{day: 3, start: "08:00", end: "14:00", mode: "in-person", location: 0, room: 1},
			{day: 5, start: "08:00", end: "14:00", mode: "in-person", location: 0, room: 1},
			{day: 2, start: "09:00", end: "15:00", mode: "in-person", location: 1, room: 0},
			{day: 4, start: "09:00", end: "15:00", mode: "in-person", location: 1, room: 0},
			{day: 2, start: "18:00", end: "20:00", mode: "video", location: -1, room: -1},
		},
	},
	{
		email: "tarik.testic@" + demoDomain, given: "Tarik", family: "Testić", specialty: "394582007",
		inPerson: "DP-01", video: "VK-01",
		hours: append(weekdays(1, 5, "12:00", "18:00", "in-person", 0, 2),
			demoHours{day: 3, start: "19:00", end: "21:00", mode: "video", location: -1, room: -1}),
	},
	{
		email: "maja.probic@" + demoDomain, given: "Maja", family: "Probić", specialty: "394537008",
		inPerson: "PP-01", video: "VK-01",
		hours: append(weekdays(1, 5, "08:00", "14:00", "in-person", 1, 1),
			demoHours{day: 6, start: "09:00", end: "12:00", mode: "in-person", location: 1, room: 1}),
	},
}

// weekdays is one rule for each day from first to last.
func weekdays(first, last int, start, end, mode string, location, room int) []demoHours {
	var out []demoHours
	for d := first; d <= last; d++ {
		out = append(out, demoHours{day: d, start: start, end: end, mode: mode, location: location, room: room})
	}
	return out
}

// demoLocation is one of the clinic's two places, with its rooms and devices.
type demoLocation struct {
	name, addressLine, postalCode, cityID, phone string
	rooms, devices                               []string
}

// demoLocations: the clinic's seat in Sarajevo (Centar) and a second place in Tuzla; the city ids are the
// places table's (Wikidata).
var demoLocations = []demoLocation{
	{
		name: "Primjer Centar", addressLine: "Ulica Primjera 1", postalCode: "71000", cityID: "Q1270190", phone: "+387000000001",
		rooms: []string{"Ordinacija 1", "Ordinacija 2", "Ordinacija 3"}, devices: []string{"EKG aparat"},
	},
	{
		name: "Primjer Tuzla", addressLine: "Ulica Primjera 2", postalCode: "75000", cityID: "Q19904705", phone: "+387000000002",
		rooms: []string{"Ordinacija 1", "Ordinacija 2"},
	},
}

// demoService is one line of the price list, in BAM's minor unit (feninga); device names the device it needs.
type demoService struct {
	code, name, device string
	modes              []string
	price              int64
	minutes            int
	active             bool
}

// demoServices is the clinic's price list. The retired line shows that a service leaves the list without leaving the
// receipts that name it.
var demoServices = []demoService{
	{code: "OP-01", name: "Opći pregled", price: 5000, minutes: 30, modes: []string{"in-person"}, active: true},
	{code: "KP-01", name: "Kardiološki pregled", price: 8000, minutes: 30, modes: []string{"in-person"}, active: true},
	{code: "DP-01", name: "Dermatološki pregled", price: 6000, minutes: 30, modes: []string{"in-person"}, active: true},
	{code: "PP-01", name: "Pedijatrijski pregled", price: 5000, minutes: 30, modes: []string{"in-person"}, active: true},
	{code: "VK-01", name: "Video konsultacija", price: 4000, minutes: 30, modes: []string{"video"}, active: true},
	{code: "KN-01", name: "Kontrolni pregled", price: 3000, minutes: 30, modes: []string{"in-person", "video"}, active: true},
	{code: "EKG-01", name: "EKG snimanje", price: 2500, minutes: 15, modes: []string{"in-person"}, device: "EKG aparat", active: true},
	{code: "LAB-01", name: "Krvna slika (laboratorij)", price: 1500, minutes: 15, modes: []string{"in-person"}, active: true},
	{code: "UZ-01", name: "Ultrazvuk abdomena", price: 7000, minutes: 30, modes: []string{"in-person"}, active: false},
}

// demoPatients is the number of patients; the first demoAppPatients of them use the patient app (an account of their
// own), the others were registered at the desk.
const (
	demoPatients    = 60
	demoAppPatients = 5
)

var (
	demoGivenNames = []string{
		"Amra", "Adnan", "Lejla", "Haris", "Selma", "Mirza", "Ena", "Kenan", "Dženana", "Edin",
		"Lana", "Damir", "Una", "Benjamin", "Iva", "Nedim", "Sara", "Ivan", "Mina", "Marko",
		"Hana", "Petar", "Nina", "Luka", "Ajla", "Davor", "Sanja", "Faruk", "Merima", "Igor",
	}
	demoFamilyNames = []string{"Primjerović", "Uzorković", "Testić", "Demić", "Probić", "Oglednić"}
)

// demoPerson is the n-th patient's name, birth date and phone: every pair of names is different.
func demoPerson(n int) (given, family, birthDate, phone string) {
	given = demoGivenNames[n%len(demoGivenNames)]
	family = demoFamilyNames[(n+n/len(demoGivenNames))%len(demoFamilyNames)]
	birthDate = fmt.Sprintf("%d-%02d-%02d", 1945+(n*13)%78, n%12+1, n%27+1)
	phone = fmt.Sprintf("+387000001%03d", n)
	return given, family, birthDate, phone
}

// demoPatientEmail is the n-th app patient's address.
func demoPatientEmail(n int) string { return fmt.Sprintf("pacijent%d@%s", n+1, demoDomain) }
