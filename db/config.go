package db

type Config struct {
	Addr string `toml:"address"`
	Port int    `toml:"port"`
}
