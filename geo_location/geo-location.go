package geo_location

import (
	"log"
	"net"

	"github.com/oschwald/geoip2-golang"
)

func LookupGeoLocation(ip string) (string, string, string) {
	db, err := geoip2.Open("GeoLite2-City.mmdb")
	if err != nil {
		log.Fatalf("Could not open GeoIP database: %s\n", err)
	}
	defer db.Close()

	ipAddr := net.ParseIP(ip)
	record, err := db.City(ipAddr)
	if err != nil {
		log.Printf("Could not get GeoIP data for IP %s: %s\n", ip, err)
		return "Unknown", "Unknown", "Unknown"
	}
	continent := getContinent(record.Country.IsoCode)
	if len(record.Subdivisions) > 0 {
		return record.Subdivisions[0].Names["en"], record.Country.IsoCode, continent // Return English name of the region
	}
	return "Unknown", record.Country.IsoCode, continent // Return the country code
}

func getContinent(countryCode string) string {
	switch countryCode {
	case "AF", "AM", "AZ", "BH", "BD", "BT", "BN", "KH", "CN", "GE", "IN", "ID", "IR", "IQ", "IL", "JP", "JO", "KZ", "KW", "KG", "LA", "LB", "MY", "MV", "MN", "MM", "NP", "KP", "OM", "PK", "PS", "PH", "QA", "SA", "SG", "KR", "LK", "SY", "TW", "TJ", "TH", "TR", "TM", "AE", "UZ", "VN", "YE":
		return "Asia"
	case "AL", "AD", "AT", "BY", "BE", "BA", "BG", "HR", "CY", "CZ", "DK", "EE", "FO", "FI", "FR", "DE", "GI", "GR", "HU", "IS", "IE", "IM", "IT", "LV", "LI", "LT", "LU", "MK", "MT", "MD", "MC", "ME", "NL", "NO", "PL", "PT", "RO", "RU", "SM", "RS", "SK", "SI", "ES", "SJ", "SE", "CH", "UA", "GB", "VA":
		return "Europe"
	case "AI", "AG", "AR", "AW", "BS", "BB", "BZ", "BM", "BO", "BR", "VG", "CA", "KY", "CL", "CO", "CR", "CU", "CW", "DM", "DO", "EC", "SV", "FK", "GL", "GD", "GP", "GT", "GY", "HT", "HN", "JM", "MQ", "MX", "MS", "NI", "PA", "PY", "PE", "PR", "BL", "KN", "LC", "MF", "PM", "VC", "SX", "SR", "TT", "TC", "US", "UY", "VE", "VI":
		return "North America"
	case "DZ", "AO", "BJ", "BW", "BF", "BI", "CV", "CM", "CF", "TD", "KM", "CD", "CG", "CI", "DJ", "EG", "GQ", "ER", "ET", "GA", "GM", "GH", "GN", "GW", "KE", "LS", "LR", "LY", "MG", "MW", "ML", "MR", "MU", "YT", "MA", "MZ", "NA", "NE", "NG", "RW", "RE", "ST", "SN", "SC", "SL", "SO", "ZA", "SS", "SH", "SD", "SZ", "TZ", "TG", "TN", "UG", "EH", "ZM", "ZW":
		return "Africa"
	case "AU", "FJ", "KI", "MH", "FM", "NR", "NZ", "PW", "PG", "WS", "SB", "TO", "TV", "VU":
		return "Oceania"
	default:
		return "Unknown"
	}
}
