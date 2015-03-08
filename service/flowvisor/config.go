package flowvisor

type Config struct {
	Addr string `toml:"controller_address"`
	Port int    `toml:"controller_port"`
}
