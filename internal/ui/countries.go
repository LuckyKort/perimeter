package ui

import "strings"

// isoCountries lists ISO 3166-1 alpha-2 codes for the panel's country picker.
const isoCountries = "AD AE AF AG AI AL AM AO AR AT AU AW AZ BA BB BD BE BF BG BH BI BJ BM BN BO BR BS BT BW BY BZ " +
	"CA CD CF CG CH CI CL CM CN CO CR CU CV CY CZ DE DJ DK DM DO DZ EC EE EG ER ES ET FI FJ FM FO FR GA GB GD GE GH GI " +
	"GL GM GN GQ GR GT GU GW GY HK HN HR HT HU ID IE IL IM IN IQ IR IS IT JE JM JO JP KE KG KH KI KM KN KP KR KW KY KZ " +
	"LA LB LC LI LK LR LS LT LU LV LY MA MC MD ME MG MH MK ML MM MN MO MR MT MU MV MW MX MY MZ NA NE NG NI NL NO NP NR " +
	"NZ OM PA PE PG PH PK PL PR PS PT PW PY QA RO RS RU RW SA SB SC SD SE SG SI SK SL SM SN SO SR SS ST SV SY SZ TD TG " +
	"TH TJ TL TM TN TO TR TT TV TW TZ UA UG US UY UZ VA VC VE VG VI VN VU WS XK YE ZA ZM ZW"

type countryItem struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Flag string `json:"flag"`
}

func countryList() []countryItem {
	codes := strings.Fields(isoCountries)
	out := make([]countryItem, 0, len(codes))
	for _, c := range codes {
		out = append(out, countryItem{Code: c, Name: CountryName(c), Flag: Flag(c)})
	}
	return out
}
