package server

// gplcPlace is one entry of the built-in gazetteer that answers every
// geo-places query. Positions are [longitude, latitude].
type gplcPlace struct {
	id, placeType, title string
	pos                  [2]float64
	country              [3]string // Code2, Code3, Name
	region               [2]string // Code, Name
	subRegion, locality  string
	district, postal     string
	street, number       string
	categories           [][2]string // Id, Name; the first is primary
	timeZone             string
	tzOffset             string
	website              string
	aliases              []string
}

// gplcGazetteer is deliberately small: a handful of cities and landmarks on
// several continents, enough for deterministic search, geocoding and
// reverse geocoding in tests.
var gplcGazetteer = []gplcPlace{
	{id: "seattle", placeType: "Locality", title: "Seattle, WA, United States", pos: [2]float64{-122.33207, 47.60621},
		country: [3]string{"US", "USA", "United States"}, region: [2]string{"WA", "Washington"}, subRegion: "King", locality: "Seattle",
		timeZone: "America/Los_Angeles", tzOffset: "-08:00"},
	{id: "space-needle", placeType: "PointOfInterest", title: "Space Needle", pos: [2]float64{-122.34930, 47.62051},
		country: [3]string{"US", "USA", "United States"}, region: [2]string{"WA", "Washington"}, subRegion: "King", locality: "Seattle",
		district: "Lower Queen Anne", postal: "98109", street: "Broad St", number: "400",
		categories: [][2]string{{"landmark-attraction", "Landmark-Attraction"}, {"observation-deck", "Observation Deck"}},
		timeZone:   "America/Los_Angeles", tzOffset: "-08:00", website: "https://www.spaceneedle.com", aliases: []string{"needle", "tower"}},
	{id: "pike-place-market", placeType: "PointOfInterest", title: "Pike Place Market", pos: [2]float64{-122.34211, 47.60970},
		country: [3]string{"US", "USA", "United States"}, region: [2]string{"WA", "Washington"}, subRegion: "King", locality: "Seattle",
		district: "Belltown", postal: "98101", street: "Pike Pl", number: "85",
		categories: [][2]string{{"market", "Market"}, {"landmark-attraction", "Landmark-Attraction"}},
		timeZone:   "America/Los_Angeles", tzOffset: "-08:00", website: "https://www.pikeplacemarket.org"},
	{id: "2111-7th-ave", placeType: "PointAddress", title: "2111 7th Ave, Seattle, WA 98121, United States", pos: [2]float64{-122.33856, 47.61552},
		country: [3]string{"US", "USA", "United States"}, region: [2]string{"WA", "Washington"}, subRegion: "King", locality: "Seattle",
		district: "Denny Triangle", postal: "98121", street: "7th Ave", number: "2111",
		timeZone: "America/Los_Angeles", tzOffset: "-08:00"},
	{id: "vancouver", placeType: "Locality", title: "Vancouver, BC, Canada", pos: [2]float64{-123.11934, 49.24966},
		country: [3]string{"CA", "CAN", "Canada"}, region: [2]string{"BC", "British Columbia"}, subRegion: "Greater Vancouver", locality: "Vancouver",
		timeZone: "America/Vancouver", tzOffset: "-08:00"},
	{id: "stanley-park", placeType: "PointOfInterest", title: "Stanley Park", pos: [2]float64{-123.14427, 49.30425},
		country: [3]string{"CA", "CAN", "Canada"}, region: [2]string{"BC", "British Columbia"}, subRegion: "Greater Vancouver", locality: "Vancouver",
		postal: "V6G 1Z4", street: "Stanley Park Dr",
		categories: [][2]string{{"park-recreation-area", "Park-Recreation Area"}},
		timeZone:   "America/Vancouver", tzOffset: "-08:00"},
	{id: "new-york", placeType: "Locality", title: "New York, NY, United States", pos: [2]float64{-74.00597, 40.71427},
		country: [3]string{"US", "USA", "United States"}, region: [2]string{"NY", "New York"}, subRegion: "New York", locality: "New York",
		timeZone: "America/New_York", tzOffset: "-05:00", aliases: []string{"nyc", "manhattan"}},
	{id: "empire-state-building", placeType: "PointOfInterest", title: "Empire State Building", pos: [2]float64{-73.98566, 40.74844},
		country: [3]string{"US", "USA", "United States"}, region: [2]string{"NY", "New York"}, subRegion: "New York", locality: "New York",
		district: "Midtown", postal: "10118", street: "5th Ave", number: "350",
		categories: [][2]string{{"landmark-attraction", "Landmark-Attraction"}, {"office-building", "Office Building"}},
		timeZone:   "America/New_York", tzOffset: "-05:00", website: "https://www.esbnyc.com"},
	{id: "statue-of-liberty", placeType: "PointOfInterest", title: "Statue of Liberty", pos: [2]float64{-74.04454, 40.68925},
		country: [3]string{"US", "USA", "United States"}, region: [2]string{"NY", "New York"}, subRegion: "New York", locality: "New York",
		postal: "10004", street: "Liberty Island",
		categories: [][2]string{{"historical-monument", "Historical Monument"}, {"landmark-attraction", "Landmark-Attraction"}},
		timeZone:   "America/New_York", tzOffset: "-05:00"},
	{id: "london", placeType: "Locality", title: "London, England, United Kingdom", pos: [2]float64{-0.12574, 51.50853},
		country: [3]string{"GB", "GBR", "United Kingdom"}, region: [2]string{"ENG", "England"}, subRegion: "Greater London", locality: "London",
		timeZone: "Europe/London", tzOffset: "+00:00"},
	{id: "big-ben", placeType: "PointOfInterest", title: "Big Ben", pos: [2]float64{-0.12462, 51.50073},
		country: [3]string{"GB", "GBR", "United Kingdom"}, region: [2]string{"ENG", "England"}, subRegion: "Greater London", locality: "London",
		district: "Westminster", postal: "SW1A 0AA", street: "Bridge St",
		categories: [][2]string{{"landmark-attraction", "Landmark-Attraction"}, {"historical-monument", "Historical Monument"}},
		timeZone:   "Europe/London", tzOffset: "+00:00", aliases: []string{"elizabeth tower", "tower"}},
	{id: "paris", placeType: "Locality", title: "Paris, Île-de-France, France", pos: [2]float64{2.35222, 48.85661},
		country: [3]string{"FR", "FRA", "France"}, region: [2]string{"IDF", "Île-de-France"}, subRegion: "Paris", locality: "Paris",
		timeZone: "Europe/Paris", tzOffset: "+01:00"},
	{id: "eiffel-tower", placeType: "PointOfInterest", title: "Eiffel Tower", pos: [2]float64{2.29448, 48.85826},
		country: [3]string{"FR", "FRA", "France"}, region: [2]string{"IDF", "Île-de-France"}, subRegion: "Paris", locality: "Paris",
		district: "7e Arrondissement", postal: "75007", street: "Avenue Gustave Eiffel", number: "5",
		categories: [][2]string{{"landmark-attraction", "Landmark-Attraction"}, {"observation-deck", "Observation Deck"}},
		timeZone:   "Europe/Paris", tzOffset: "+01:00", website: "https://www.toureiffel.paris", aliases: []string{"tour eiffel", "tower"}},
	{id: "louvre", placeType: "PointOfInterest", title: "Musée du Louvre", pos: [2]float64{2.33764, 48.86061},
		country: [3]string{"FR", "FRA", "France"}, region: [2]string{"IDF", "Île-de-France"}, subRegion: "Paris", locality: "Paris",
		district: "1er Arrondissement", postal: "75001", street: "Rue de Rivoli",
		categories: [][2]string{{"museum", "Museum"}},
		timeZone:   "Europe/Paris", tzOffset: "+01:00", aliases: []string{"louvre museum"}},
	{id: "berlin", placeType: "Locality", title: "Berlin, Germany", pos: [2]float64{13.40495, 52.52000},
		country: [3]string{"DE", "DEU", "Germany"}, region: [2]string{"BE", "Berlin"}, locality: "Berlin",
		timeZone: "Europe/Berlin", tzOffset: "+01:00"},
	{id: "brandenburg-gate", placeType: "PointOfInterest", title: "Brandenburger Tor", pos: [2]float64{13.37770, 52.51628},
		country: [3]string{"DE", "DEU", "Germany"}, region: [2]string{"BE", "Berlin"}, locality: "Berlin",
		district: "Mitte", postal: "10117", street: "Pariser Platz",
		categories: [][2]string{{"historical-monument", "Historical Monument"}},
		timeZone:   "Europe/Berlin", tzOffset: "+01:00", aliases: []string{"brandenburg gate"}},
	{id: "tokyo", placeType: "Locality", title: "Tokyo, Japan", pos: [2]float64{139.69171, 35.68950},
		country: [3]string{"JP", "JPN", "Japan"}, region: [2]string{"13", "Tokyo"}, locality: "Tokyo",
		timeZone: "Asia/Tokyo", tzOffset: "+09:00"},
	{id: "tokyo-tower", placeType: "PointOfInterest", title: "Tokyo Tower", pos: [2]float64{139.74543, 35.65858},
		country: [3]string{"JP", "JPN", "Japan"}, region: [2]string{"13", "Tokyo"}, locality: "Tokyo",
		district: "Minato", postal: "105-0011", street: "Shibakoen", number: "4-2-8",
		categories: [][2]string{{"landmark-attraction", "Landmark-Attraction"}, {"observation-deck", "Observation Deck"}},
		timeZone:   "Asia/Tokyo", tzOffset: "+09:00", aliases: []string{"tower"}},
	{id: "sydney", placeType: "Locality", title: "Sydney, NSW, Australia", pos: [2]float64{151.20732, -33.86785},
		country: [3]string{"AU", "AUS", "Australia"}, region: [2]string{"NSW", "New South Wales"}, locality: "Sydney",
		timeZone: "Australia/Sydney", tzOffset: "+10:00"},
	{id: "sydney-opera-house", placeType: "PointOfInterest", title: "Sydney Opera House", pos: [2]float64{151.21528, -33.85678},
		country: [3]string{"AU", "AUS", "Australia"}, region: [2]string{"NSW", "New South Wales"}, locality: "Sydney",
		district: "Bennelong Point", postal: "2000", street: "Bennelong Point",
		categories: [][2]string{{"performing-arts", "Performing Arts"}, {"landmark-attraction", "Landmark-Attraction"}},
		timeZone:   "Australia/Sydney", tzOffset: "+10:00", website: "https://www.sydneyoperahouse.com"},
}
