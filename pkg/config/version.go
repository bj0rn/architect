package config

import (
	"fmt"
	extVersion "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/docker"
	"path"
	"regexp"
	"strings"
)

/*
EKSEMPEL:
Gitt følgende URL http://uil0map-paas-app01.skead.no:9090/v2/aurora/console/tags/list
COMPlETE	 		= 2.0.0-b1.11.0-oracle8-1.0.2
LATEST				= latest
MAJOR				= 2
MINOR				= 2.0
PATCH				= 2.0.0
OutputImage.Repository	aurora/console

*/

type VersionInfo struct {
	AppVersion string
	AuroraVersion string
}

func GetFilteredReleaseVersionTags(ver VersionInfo, extraTags string, repoTags []string) ([]string, error) {
	tags, err := getReleaseVersionTags(ver, extraTags)

	if err != nil {
		return nil, err
	}

	return filterVersionTags(ver.AppVersion, tags, repoTags)
}

func GetReleaseVersion(cfg Config, baseImageVersion string) VersionInfo {
	appVersion :=  cfg.MavenGav.Version
	auroraVersion := getAuroraVersion(baseImageVersion, appVersion, cfg)
	return VersionInfo{appVersion, auroraVersion}
}

/*
  Create app version. If not snapshot build, then return version from GAV.
  Otherwise, create new snapshot version based on deliverable.
*/
func GetSnapshotVersion(cfg Config, deliverablePath string, baseImageVersion string) VersionInfo {
	replacer := strings.NewReplacer(cfg.MavenGav.ArtifactId, "", "-Leveransepakke.zip", "")
	appVersion := "SNAPSHOT-" + replacer.Replace(path.Base(deliverablePath))
	auroraVersion := getAuroraVersion(baseImageVersion, appVersion, cfg)
	return VersionInfo{appVersion, auroraVersion}
}

func GetSnapshotVersionTags(ver VersionInfo) []string {
	return []string{ver.AuroraVersion}
}

func GetTemporaryVersionTags(cfg Config) []string {
	return []string{cfg.DockerSpec.TagWith}
}

func IsSnapshot(version string) bool {
	if strings.Contains(version, "SNAPSHOT") {
		return true
	}
	return false
}

func IsSemantic(version string) bool {
	var validStr = regexp.MustCompile(`^[0-9]+.[0-9]+.[0-9]+$`)
	if validStr.MatchString(version) {
		return true
	}
	return false
}

func filterVersionTags(appVersion string, newTags []string, repositoryTags []string) ([]string, error) {
	if !IsSemantic(appVersion) {
		return newTags, nil
	}

	var excludeMinor, excludeMajor, excludeLatest bool = true, true, true

	minorTagName, err := getMinor(appVersion, true)

	if err != nil {
		return nil, err
	}

	excludeMinor, err = tagCompare("> "+appVersion+", < "+minorTagName, repositoryTags)

	if err != nil {
		return nil, err
	}

	majorTagName, err := getMajor(appVersion, true)

	if err != nil {
		return nil, err
	}

	excludeMajor, err = tagCompare("> "+appVersion+", < "+majorTagName, repositoryTags)

	if err != nil {
		return nil, err
	}

	excludeLatest, err = tagCompare("> "+appVersion, repositoryTags)

	if err != nil {
		return nil, err
	}

	versions := make([]string, 0, 10)

	for _, tag := range newTags {
		if strings.EqualFold(strings.TrimSpace(tag), "latest") {
			if !excludeLatest {
				versions = append(versions, tag)
			}
		} else if isMinor(tag) {
			if !excludeMinor {
				versions = append(versions, tag)
			}
		} else if isMajor(tag) {
			if !excludeMajor {
				versions = append(versions, tag)
			}
		} else {
			versions = append(versions, tag)
		}
	}
	return versions, nil
}

func tagCompare(versionConstraint string, tags []string) (bool, error) {
	c, err := extVersion.NewConstraint(versionConstraint)

	if err != nil {
		return false, errors.Wrapf(err, "Could not create version constraint %s", versionConstraint)
	}

	for _, tag := range tags {
		if IsSemantic(tag) {
			v, err := extVersion.NewVersion(tag)

			if err != nil {
				return false, errors.Wrapf(err, "Could not create tag constraint %s", tag)
			}

			if c.Check(v) {
				return true, nil
			}
		}
	}

	return false, nil
}

func createEnv(appVersion string, auroraVersion string, cfg Config) map[string]string {
	env := make(map[string]string)

	env[docker.ENV_APP_VERSION] = appVersion
	env[docker.ENV_AURORA_VERSION] = auroraVersion
	env[docker.ENV_PUSH_EXTRA_TAGS] = cfg.DockerSpec.PushExtraTags
	env[docker.TZ] = "Europe/Oslo"

	if IsSnapshot(cfg.MavenGav.Version) {
		env[docker.ENV_SNAPSHOT_TAG] = cfg.MavenGav.Version
	}

	return env
}

func getReleaseVersionTags(ver VersionInfo, extraTags string) ([]string, error) {
	versions := make([]string, 0, 10)

	if strings.Contains(extraTags, "latest") {
		versions = append(versions, "latest")
	}

	versions = append(versions, ver.AuroraVersion)

	if strings.Contains(extraTags, "major") {
		majorVersion, err := getMajor(ver.AppVersion, false)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get major version")
		}

		versions = append(versions, majorVersion)
	}

	if strings.Contains(extraTags, "minor") {
		minorVersion, err := getMinor(ver.AppVersion, false)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to get minor version")
		}
		versions = append(versions, minorVersion)
	}

	if strings.Contains(extraTags, "patch") {
		versions = append(versions, ver.AppVersion)
	}

	return versions, nil
}

func getMajor(version string, bumpVersion bool) (string, error) {
	build_version, err := extVersion.NewVersion(version)

	if err != nil {
		return "", errors.Wrap(err, "Error in parsing major version: "+version)
	}

	versionMajor := build_version.Segments()[0]
	if bumpVersion {
		versionMajor += 1
	}

	return fmt.Sprintf("%d", versionMajor), nil
}

func isMajor(version string) bool {
	var validStr = regexp.MustCompile(`^[0-9]+$`)
	if validStr.MatchString(version) {
		return true
	}
	return false
}

func getMinor(version string, bumpVersion bool) (string, error) {
	build_version, err := extVersion.NewVersion(version)

	if err != nil {
		return "", errors.Wrap(err, "Error in parsing minor version: "+version)
	}

	versionMinor := build_version.Segments()[1]
	if bumpVersion {
		versionMinor += 1
	}

	return fmt.Sprintf("%d.%d", build_version.Segments()[0], versionMinor), nil
}

func isMinor(version string) bool {
	var validStr = regexp.MustCompile(`^[0-9]+.[0-9]+$`)
	if validStr.MatchString(version) {
		return true
	}
	return false
}

/*
  Create aurora version aka complete version
  <application-version>-<builder-version>-<baseimage-repository>-<baseimage-version>
  e.g. 2.0.0-b1.11.0-oracle8-1.0.2
*/
func getAuroraVersion(baseImageVersion, appVersion string, cfg Config) string {
	builderVersion := cfg.BuilderSpec.Version
	lastNameInRepo := getLastIndexInRepository(cfg.DockerSpec.BaseImage)

	return fmt.Sprintf("%s-b%s-%s-%s", appVersion, builderVersion, lastNameInRepo, baseImageVersion)
}

func getLastIndexInRepository(repository string) string {
	s := strings.Split(repository, "/")
	return s[len(s)-1]
}
