package process

import (
	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/config/runtime"
	"github.com/Masterminds/glide/cfg"
	"github.com/skatteetaten/architect/pkg/process/tagger"
)

func Build(credentials *docker.RegistryCredentials, cfg *config.Config, prepper Prepper) error {
	provider := docker.NewRegistryClient(cfg.DockerSpec.ExternalDockerRegistry)
	dockerBuildConfig, err := prepper(cfg, provider)
	if err != nil {
		return errors.Wrap(err, "Error preparing image")
	}

	client, err := docker.NewDockerClient()
	if err != nil {
		return errors.Wrap(err, "Error initializing Docker")
	}

	for _, buildConfig := range dockerBuildConfig {
		imageid, err := client.BuildImage(buildConfig)

		if err != nil {
			return errors.Wrap(err, "Fuckup!")
		} else {
			logrus.Infof("Done building. Imageid: %s", imageid)
		}

		tagger := tagger.NormalTagResolver{
			Overwrite: cfg.DockerSpec.TagOverwrite,



		}
		buildConfig.AuroraVersion.GetApplicationVersion().GetApplicationVersionTagsToPush()
		logrus.Debug("Push images and tags")
		err = client.PushImages(buildConfig., credentials)
		if err != nil {
			return errors.Wrap(err, "Error pushing images")
		}
	}
	return nil
}

