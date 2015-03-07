package neighbor

type Config struct {
	AdvertisementGroup string   `toml:"advertisement_group"`
	AdvertisementZones []string `toml:"advertisement_zones"`
}
