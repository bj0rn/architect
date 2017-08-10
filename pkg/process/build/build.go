package process

import (
	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/docker"
)

func Build(credentials *docker.RegistryCredentials, cfg *config.Config, prepper Prepper) error {
	provider := docker.NewRegistryClient(cfg.DockerSpec.ExternalDockerRegistry)
	appVersion,dockerBuildConfig, err := prepper(cfg, provider)
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
		tags, err := FindCandidateTags(appVersion)
		logrus.Debug("Push images and tags")
		err = client.PushImages(buildConfig.VersionTags, credentials)
		if err != nil {
			return errors.Wrap(err, "Error pushing images")
		}
	}
	return nil
}

func FindCandidateTags(
appVersion config.AppVersion, candidateTags []string,
cfg *config.Config, provider docker.ImageInfoProvider) ([]string, error) {

	ds := cfg.DockerSpec
	versionTags := candidateTags
	if !ds.TagOverwrite {
		logrus.Debug("Tags Overwrite disabled, filtering tags")

		repositoryTags, err := provider.GetTags(ds.OutputRepository)
		logrus.Debug("Tags in repository ", repositoryTags)
		if err != nil {
			return nil, errors.Wrapf(err, "Error in GetTags, repository=%s", ds.OutputRepository)
		}

		versionTags, err = appVersion.FilterVersionTags(candidateTags, repositoryTags.Tags)
		if err != nil {
			return nil, errors.Wrapf(err, "Error in FilterVersionTags, app_version=%s, " +
				"versionTags=%v, repositoryTags=%v",
				appVersion, versionTags, repositoryTags.Tags)
		}
		logrus.Debug("Filtered tags ", versionTags)
	}
	return docker.CreateImageNameFromSpecAndTags(versionTags, ds.OutputRegistry, ds.OutputRepository), nil

}
