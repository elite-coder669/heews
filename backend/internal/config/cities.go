package config

// CityInfo describes one supported municipality: display label, the state
// it's in, and a representative lat/lon used for the city-wide weather feed
// (individual ward centroids come from that city's ward GeoJSON at
// data/fixtures/wards/<slug>_wards.geojson).
type CityInfo struct {
	Label string
	State string
	Lat   float64
	Lon   float64
}

// Cities is the set of municipalities HEEWS ships ward-boundary data for.
// Sourced from the DataMeet Municipal_Spatial_Data project (25 Indian
// cities across many states — not full state-level coverage; that dataset
// doesn't exist publicly at ward granularity). Add more cities here as more
// ward GeoJSONs are dropped into data/fixtures/wards/.
var Cities = map[string]CityInfo{
	"ahmedabad": {Label: "Ahmedabad", State: "Gujarat", Lat: 23.0272, Lon: 72.5754},
	"bangalore": {Label: "Bangalore", State: "Karnataka", Lat: 12.9881, Lon: 77.6221},
	"bhopal": {Label: "Bhopal", State: "Madhya Pradesh", Lat: 23.2251, Lon: 77.3981},
	"bhubaneswar": {Label: "Bhubaneswar", State: "Odisha", Lat: 20.2888, Lon: 85.8283},
	"bodh_gaya": {Label: "Bodh Gaya", State: "Bihar", Lat: 24.709, Lon: 84.9843},
	"chandigarh": {Label: "Chandigarh", State: "Chandigarh (UT)", Lat: 30.7299, Lon: 76.777},
	"chennai": {Label: "Chennai", State: "Tamil Nadu", Lat: 13.0436, Lon: 80.2357},
	"coimbatore": {Label: "Coimbatore", State: "Tamil Nadu", Lat: 11.009, Lon: 76.9672},
	"delhi": {Label: "Delhi", State: "Delhi (NCT)", Lat: 28.6439, Lon: 77.0931},
	"faridabad": {Label: "Faridabad", State: "Haryana", Lat: 28.4043, Lon: 77.2889},
	"hyderabad": {Label: "Hyderabad", State: "Telangana", Lat: 17.4262, Lon: 78.4465},
	"jaipur": {Label: "Jaipur", State: "Rajasthan", Lat: 26.8983, Lon: 75.8003},
	"kanpur": {Label: "Kanpur", State: "Uttar Pradesh", Lat: 26.4342, Lon: 80.3359},
	"katihar": {Label: "Katihar", State: "Bihar", Lat: 25.5497, Lon: 87.5712},
	"kishangarh": {Label: "Kishangarh", State: "Rajasthan", Lat: 26.59, Lon: 74.8475},
	"kochi": {Label: "Kochi", State: "Kerala", Lat: 9.9716, Lon: 76.2887},
	"kolkata": {Label: "Kolkata", State: "West Bengal", Lat: 22.5415, Lon: 88.3505},
	"lucknow": {Label: "Lucknow", State: "Uttar Pradesh", Lat: 26.8392, Lon: 80.946},
	"mumbai": {Label: "Mumbai", State: "Maharashtra", Lat: 19.0821, Lon: 72.878},
	"nmmc": {Label: "NMMC", State: "Maharashtra", Lat: 19.092, Lon: 73.0126},
	"pcmc": {Label: "PCMC", State: "Maharashtra", Lat: 18.6425, Lon: 73.8222},
	"pune": {Label: "Pune", State: "Maharashtra", Lat: 18.5036, Lon: 73.8752},
	"purnia": {Label: "Purnia", State: "Bihar", Lat: 25.8005, Lon: 87.4878},
	"vadodara": {Label: "Vadodara", State: "Gujarat", Lat: 22.2982, Lon: 73.1916},
	"vijayawada": {Label: "Vijayawada", State: "Andhra Pradesh", Lat: 16.5316, Lon: 80.6355},
}

// CityOrDefault returns the CityInfo for slug, or Hyderabad's if the slug is
// unknown (so a typo in CITY= never crashes the app).
func CityOrDefault(slug string) (string, CityInfo) {
	if c, ok := Cities[slug]; ok {
		return slug, c
	}
	return "hyderabad", Cities["hyderabad"]
}
