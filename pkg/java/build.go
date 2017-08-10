package java

import (
	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/java/nexus"
	"github.com/skatteetaten/architect/pkg/java/prepare"
	"github.com/skatteetaten/architect/pkg/process/build"
	"path"
	"strings"
)

func Prepper(downloader nexus.Downloader) process.Prepper {
	return func(cfg *config.Config, provider docker.ImageInfoProvider) ([]docker.DockerBuildConfig, error) {
		logrus.Debugf("Download deliverable for GAV %-v", cfg.MavenGav)
		deliverable, err := downloader.DownloadArtifact(cfg.MavenGav)
		if err != nil {
			return nil, errors.Wrapf(err, "Could not download deliverable %-v", cfg.MavenGav)
		}

		logrus.Debug("Extract build info")

		getAppVersionString := getAppVersion(cfg, deliverable.Path)
		completeBuildImageVersion, err := provider.GetCompleteBaseImageVersion(cfg.DockerSpec.BaseImage,
			cfg.DockerSpec.BaseVersion)
		if err != nil {
			return nil, errors.Wrap(err, "Unable to get the complete build version")
		}
		auroraVersions, err := config.NewAuroraVersions(getAppVersionString, cfg.Snapshot,
			cfg.MavenGav.Version, cfg.DockerSpec, cfg.BuilderSpec, completeBuildImageVersion)
		if err != nil {
			return nil, errors.Wrap(err, "Error creating version information")
		}

		logrus.Debug("Prepare output image")
		buildPath, err := prepare.Prepare(cfg.DockerSpec, auroraVersions, *deliverable)

		if err != nil {
			return nil, errors.Wrap(err, "Error prepare artifact")
		}

		buildConf := docker.DockerBuildConfig{
			AppVersion: auroraVersions.GetAppVersion().GetVersionTags(cfg.DockerSpec.PushExtraTags),
			BuildFolder: buildPath,
		}
		return []docker.DockerBuildConfig{buildConf}, nil
	}
}

/*
  Create app version. If not snapshot build, then return version from GAV.
  Otherwise, create new snapshot version based on deliverable.
*/
func getAppVersion(cfg *config.Config, deliverablePath string) string {
	if strings.Contains(cfg.MavenGav.Version, "SNAPSHOT") {
		replacer := strings.NewReplacer(cfg.MavenGav.ArtifactId, "", "-Leveransepakke.zip", "")
		version := "SNAPSHOT-" + replacer.Replace(path.Base(deliverablePath))
		return version
	}
	return cfg.MavenGav.Version
}
