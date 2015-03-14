package storage

type Config struct {
	Addr string `toml:"address"`
	Port int    `toml:"port"`
}
