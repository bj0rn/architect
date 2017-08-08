package config

type BaseImageInfo struct {
	Repository string
	Version    string
}

func NewBaseImageInfo(version string, cfg Config) *BaseImageInfo {
	return &BaseImageInfo{Repository: cfg.DockerSpec.BaseImage, Version: version}
}



