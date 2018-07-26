package http

type ConfigOption func(cfg *config)

type config struct {
	postRespondWithBody bool
}

func defaultConfig() config {
	return config{
		postRespondWithBody: true,
	}
}

func configFromOptions(configOpts []ConfigOption) config {
	cfg := defaultConfig()
	for _, c := range configOpts {
		c(&cfg)
	}
	return cfg
}

func PostDoNotRespondWithBody() func(cfg *config) {
	return func(cfg *config) {
		cfg.postRespondWithBody = false
	}
}
