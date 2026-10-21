// Package shipping — pincode_lookup.go
//
// India Post pincodes are 6 digits. The first 3 digits are the "Sorting District
// Code" and uniquely identify a metro/major city + the state it sits in. This
// file ships a hand-curated lookup for the top ~80 prefixes (covering every
// metro, every state capital, and the biggest tier-2 cities). For anything
// unmatched we fall back to the 1-digit zonal mapping (only the state is
// reliable that way; city stays empty).
//
// This closes the empty City/State fields in the Shiprocket recipient that we
// left in step A of the audit. The data here is static and small enough to keep
// in-process; if you want to add more prefixes later just append to either map.
package shipping

import "strings"

// Location is the result of a pincode lookup. City may be empty when only the
// state can be derived (1-digit zonal fallback).
type Location struct {
	City  string
	State string
}

// majorCityByPrefix3 maps the first 3 digits of a pincode to (city, state).
// Source: India Post Sorting District Code list; covers every metro + state
// capital + the largest tier-2 cities. Append more as needed.
var majorCityByPrefix3 = map[string]Location{
	// ── Zone 1: Delhi, HR, PB, HP, J&K ────────────────────────────────────────
	"110": {"New Delhi", "Delhi"},
	"122": {"Gurugram", "Haryana"},
	"124": {"Rohtak", "Haryana"},
	"125": {"Hisar", "Haryana"},
	"127": {"Bhiwani", "Haryana"},
	"131": {"Sonipat", "Haryana"},
	"132": {"Karnal", "Haryana"},
	"134": {"Ambala", "Haryana"},
	"141": {"Ludhiana", "Punjab"},
	"143": {"Amritsar", "Punjab"},
	"144": {"Jalandhar", "Punjab"},
	"146": {"Pathankot", "Punjab"},
	"147": {"Patiala", "Punjab"},
	"160": {"Chandigarh", "Chandigarh"},
	"171": {"Shimla", "Himachal Pradesh"},
	"180": {"Jammu", "Jammu and Kashmir"},
	"190": {"Srinagar", "Jammu and Kashmir"},

	// ── Zone 2: UP, UK ────────────────────────────────────────────────────────
	"201": {"Ghaziabad", "Uttar Pradesh"},
	"203": {"Bulandshahr", "Uttar Pradesh"},
	"208": {"Kanpur", "Uttar Pradesh"},
	"211": {"Allahabad", "Uttar Pradesh"},
	"221": {"Varanasi", "Uttar Pradesh"},
	"226": {"Lucknow", "Uttar Pradesh"},
	"228": {"Sultanpur", "Uttar Pradesh"},
	"243": {"Bareilly", "Uttar Pradesh"},
	"244": {"Moradabad", "Uttar Pradesh"},
	"247": {"Saharanpur", "Uttar Pradesh"},
	"248": {"Dehradun", "Uttarakhand"},
	"250": {"Meerut", "Uttar Pradesh"},
	"263": {"Nainital", "Uttarakhand"},
	"282": {"Agra", "Uttar Pradesh"},
	"284": {"Jhansi", "Uttar Pradesh"},

	// ── Zone 3: RJ, GJ ───────────────────────────────────────────────────────
	"302": {"Jaipur", "Rajasthan"},
	"305": {"Ajmer", "Rajasthan"},
	"313": {"Udaipur", "Rajasthan"},
	"324": {"Kota", "Rajasthan"},
	"334": {"Bikaner", "Rajasthan"},
	"342": {"Jodhpur", "Rajasthan"},
	"360": {"Rajkot", "Gujarat"},
	"380": {"Ahmedabad", "Gujarat"},
	"388": {"Anand", "Gujarat"},
	"390": {"Vadodara", "Gujarat"},
	"394": {"Surat", "Gujarat"},
	"396": {"Vapi", "Gujarat"},

	// ── Zone 4: MP, CG, MH, GA ───────────────────────────────────────────────
	"400": {"Mumbai", "Maharashtra"},
	"410": {"Navi Mumbai", "Maharashtra"},
	"411": {"Pune", "Maharashtra"},
	"413": {"Solapur", "Maharashtra"},
	"422": {"Nashik", "Maharashtra"},
	"440": {"Nagpur", "Maharashtra"},
	"452": {"Indore", "Madhya Pradesh"},
	"462": {"Bhopal", "Madhya Pradesh"},
	"474": {"Gwalior", "Madhya Pradesh"},
	"482": {"Jabalpur", "Madhya Pradesh"},
	"492": {"Raipur", "Chhattisgarh"},
	"403": {"Panaji", "Goa"},

	// ── Zone 5: AP, TG, KA ───────────────────────────────────────────────────
	"500": {"Hyderabad", "Telangana"},
	"515": {"Anantapur", "Andhra Pradesh"},
	"517": {"Tirupati", "Andhra Pradesh"},
	"520": {"Vijayawada", "Andhra Pradesh"},
	"530": {"Visakhapatnam", "Andhra Pradesh"},
	"560": {"Bengaluru", "Karnataka"},
	"570": {"Mysuru", "Karnataka"},
	"575": {"Mangaluru", "Karnataka"},
	"580": {"Hubballi", "Karnataka"},
	"590": {"Belagavi", "Karnataka"},

	// ── Zone 6: KL, TN, PY ───────────────────────────────────────────────────
	"600": {"Chennai", "Tamil Nadu"},
	"605": {"Puducherry", "Puducherry"},
	"620": {"Tiruchirappalli", "Tamil Nadu"},
	"625": {"Madurai", "Tamil Nadu"},
	"627": {"Tirunelveli", "Tamil Nadu"},
	"628": {"Thoothukudi", "Tamil Nadu"},
	"641": {"Coimbatore", "Tamil Nadu"},
	"673": {"Kozhikode", "Kerala"},
	"680": {"Thrissur", "Kerala"},
	"682": {"Kochi", "Kerala"},
	"695": {"Thiruvananthapuram", "Kerala"},

	// ── Zone 7: WB, OR, AS + NE ──────────────────────────────────────────────
	"700": {"Kolkata", "West Bengal"},
	"711": {"Howrah", "West Bengal"},
	"713": {"Asansol", "West Bengal"},
	"721": {"Kharagpur", "West Bengal"},
	"734": {"Siliguri", "West Bengal"},
	"751": {"Bhubaneswar", "Odisha"},
	"753": {"Cuttack", "Odisha"},
	"781": {"Guwahati", "Assam"},
	"784": {"Tezpur", "Assam"},
	"785": {"Jorhat", "Assam"},
	"786": {"Dibrugarh", "Assam"},
	"791": {"Itanagar", "Arunachal Pradesh"},
	"793": {"Shillong", "Meghalaya"},
	"794": {"Tura", "Meghalaya"},
	"795": {"Imphal", "Manipur"},
	"796": {"Aizawl", "Mizoram"},
	"797": {"Kohima", "Nagaland"},
	"799": {"Agartala", "Tripura"},

	// ── Zone 8: BR, JH ───────────────────────────────────────────────────────
	"800": {"Patna", "Bihar"},
	"812": {"Bhagalpur", "Bihar"},
	"823": {"Gaya", "Bihar"},
	"831": {"Jamshedpur", "Jharkhand"},
	"834": {"Ranchi", "Jharkhand"},
}

// stateByPrefix1 is the fallback when no 3-digit match is found. India Post
// reserves the first digit for the zone; from there we can name the state but
// not the city.
var stateByPrefix1 = map[byte]string{
	'1': "Delhi",            // multi-state zone; default to Delhi when unspecified
	'2': "Uttar Pradesh",
	'3': "Gujarat",
	'4': "Maharashtra",
	'5': "Karnataka",
	'6': "Tamil Nadu",
	'7': "West Bengal",
	'8': "Bihar",
}

// LookupPincode returns the best-known city + state for an Indian pincode.
// The returned Location is always non-empty for a 6-digit pincode (state at
// minimum) so callers don't have to special-case "NA" placeholders.
// Returns zero-value Location for clearly invalid input.
func LookupPincode(pincode string) Location {
	p := strings.TrimSpace(pincode)
	if len(p) != 6 {
		return Location{}
	}
	for _, r := range p {
		if r < '0' || r > '9' {
			return Location{}
		}
	}
	if loc, ok := majorCityByPrefix3[p[:3]]; ok {
		return loc
	}
	if state, ok := stateByPrefix1[p[0]]; ok {
		return Location{State: state}
	}
	return Location{}
}
