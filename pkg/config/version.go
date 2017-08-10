package config

import (
	"fmt"
	extVersion "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config/runtime"
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


type AuroraVersion struct {
	applicationVerion runtime.ApplicationVersion // The complete version with builder, base image and application
	baseImage         runtime.DockerImage        // The complete version with builder, base image and application
	builderImage      runtime.DockerImage        // The complete version with builder, base image and application
}

func (m *AuroraVersion) GetAuroraVersion() string {
	s := strings.Split(m.baseImage, "/")
	return fmt.Sprintf("%s-b%s-%s-%s", m.applicationVerion,
		m.builderImage.AuroraVersionComponent(), s[len(s)-1],
		m.baseImage.AuroraVersionComponent())
}

func (m *AuroraVersion) GetBaseImage() string {
	return m.baseImage
}

func (m *AuroraVersion) GetBaseImageVersion() string {
	return m.baseImageVersion
}

func (m *AuroraVersion) GetGivenVersion() string {
	return m.givenVersion
}

func (m *AuroraVersion) GetAppVersion() AppVersion {
	return m.appVersion
}

// Generates the tags given the appversion and extra tag configuration. Don't do any filtering
func (m AuroraVersion) GetVersionTags(extraTags PushExtraTags) ([]string, error) {
	versions := make([]string, 0, 10)

	if m.applicationVerion.isSemanticReleaseVersion() {
		if extraTags.Latest {
			versions = append(versions, "latest")
		}

		if extraTags.Major {
			majorVersion, err := getMajor(string(m), false)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to get major version")
			}
			versions = append(versions, majorVersion)
		}
		if extraTags.Minor {
			minorVersion, err := getMinor(string(m), false)
			if err != nil {
				return nil, errors.Wrap(err, "Failed to get minor version")
			}
			versions = append(versions, minorVersion)
		}
		if extraTags.Patch {
			versions = append(versions, string(m))
		}
	}
	return versions, nil
}

func (m *PushExtraTags) ToStringValue() string {
	str := make([]string, 0, 5)
	if m.Major {
		str = append(str, "major")
	}
	if m.Minor {
		str = append(str, "minor")
	}
	if m.Patch {
		str = append(str, "patch")
	}
	if m.Latest {
		str = append(str, "latest")
	}
	return strings.Join(str, ",")
}

func NewAppVersion(appVersion string) AppVersion {
	return AppVersion(appVersion)
}

func NewAuroraVersions(appVersion string, snapshot bool, givenVersion string, d DockerSpec,
	b BuilderSpec, baseImageVersion string) (*AuroraVersions, error) {
	return &AuroraVersions{
		appVersion:       AppVersion(appVersion),
		Snapshot:         snapshot,
		givenVersion:     givenVersion,
		baseImageVersion: baseImageVersion,
		baseImage:        d.BaseImage,
		builderVersion:   runtime.DockerImage{},
	}, nil
}

func (m AppVersion) FilterVersionTags(newTags []string, repositoryTags []string) ([]string, error) {
	if !m.isSemanticReleaseVersion() {
		return newTags, nil
	}

	var excludeMinor, excludeMajor, excludeLatest bool = true, true, true

	minorTagName, err := getMinor(string(m), true)

	if err != nil {
		return nil, err
	}

	excludeMinor, err = tagCompare("> "+string(m)+", < "+minorTagName, repositoryTags)

	if err != nil {
		return nil, err
	}

	majorTagName, err := getMajor(string(m), true)

	if err != nil {
		return nil, err
	}

	excludeMajor, err = tagCompare("> "+string(m)+", < "+majorTagName, repositoryTags)

	if err != nil {
		return nil, err
	}

	excludeLatest, err = tagCompare("> "+string(m), repositoryTags)

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
		if AppVersion(tag).isSemanticReleaseVersion() {
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

func (m AppVersion) isSemanticReleaseVersion() bool {
	var validStr = regexp.MustCompile(`^[0-9]+.[0-9]+.[0-9]+$`)
	if validStr.MatchString(string(m)) {
		return true
	}
	return false
}

/*
  Create aurora version aka complete version
  <application-version>-<builder-version>-<baseimage-repository>-<baseimage-version>
  e.g. 2.0.0-b1.11.0-oracle8-1.0.2
*/
