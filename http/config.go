package http

type ConfigOption func(cfg *Config)

type Config struct {
	postRespondWithBody bool
}

func defaultConfig() Config {
	return Config{
		postRespondWithBody: true,
	}
}

func configFromOptions(configOpts []ConfigOption) Config {
	cfg := defaultConfig()
	for _, c := range configOpts {
		c(&cfg)
	}
	return cfg
}

func PostDoNotRespondWithBody() func(cfg *Config) {
	return func(cfg *Config) {
		cfg.postRespondWithBody = false
	}
}
