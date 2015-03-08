package neighbor

type Config struct {
	Group    string   `toml:"advertisement_group"`
	Zones    []string `toml:"advertisement_zones"`
	Interval string   `toml:"solicitation_interval"`
}
