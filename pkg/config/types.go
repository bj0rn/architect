package config

import (
	"strings"
)

type ApplicationType string

const (
	JavaLeveransepakke   ApplicationType = "JavaLeveransepakke"
	NodeJsLeveransepakke ApplicationType = "NodeJsLeveranse"
)

type Config struct {
	ApplicationType ApplicationType
	MavenGav        *MavenGav
	NodeJSGav       *NodeJSGav
	Snapshot        bool //NPM don't have snapshot semantics, so this is lifted up.. Should be refactored
	DockerSpec      DockerSpec
	BuilderSpec     BuilderSpec
}

type MavenGav struct {
	ArtifactId string
	GroupId    string
	Version    string
	Classifier string
}

type NodeJSGav struct {
	NpmName string
	Version string
}

type DockerSpec struct {
	OutputRegistry   string
	OutputRepository string
	BaseImage        string
	BaseVersion      string
	PushExtraTags    PushExtraTags
	//This is the external docker registry where we check versions.
	ExternalDockerRegistry string
	TagWith                string
	RetagWith              string
	TagOverwrite           bool
}

type BuilderSpec struct {
	Version string
}

type PushExtraTags struct {
	Latest bool
	Major  bool
	Minor  bool
	Patch  bool
}

// Generates the tags given the appversion and extra tag configuration. Don't do any filtering
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

func ParseExtraTags(i string) PushExtraTags {
	p := PushExtraTags{}
	if strings.Contains(i, "major") {
		p.Major = true
	}
	if strings.Contains(i, "minor") {
		p.Minor = true
	}
	if strings.Contains(i, "patch") {
		p.Patch = true
	}
	if strings.Contains(i, "latest") {
		p.Latest = true
	}
	return p
}



