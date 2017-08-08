package java

import (
	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/docker"
	"github.com/skatteetaten/architect/pkg/java/prepare"
)

type Tagger interface {
	Tag() error
	TagTemporary() error
	RetagTemporary() error
}

type Builder interface {
	Tagger
	Build() error
}

type BuildContext struct {
	cfg config.Config
	deliverable config.Deliverable
	client docker.DockerClient
	imageInfoProvider docker.ImageInfoProvider
}

type JavaReleaseBuilder struct {
	BuildContext
}

type JavaSnapshotBuilder struct {
	BuildContext
}

func NewReleaseBuilder(cfg config.Config, deliverable config.Deliverable, imageInfoProvider docker.ImageInfoProvider, client docker.DockerClient) (JavaReleaseBuilder, error) {
	base := BuildContext{cfg: cfg, deliverable: deliverable, client: client, imageInfoProvider: imageInfoProvider}
	return JavaReleaseBuilder{BuildContext: base}, nil
}

func NewSnapshotBuilder(cfg config.Config, deliverable config.Deliverable, imageInfoProvider docker.ImageInfoProvider, client docker.DockerClient) (JavaSnapshotBuilder, error) {
	base := BuildContext{cfg: cfg, deliverable: deliverable, client: client, imageInfoProvider: imageInfoProvider}
	return JavaSnapshotBuilder{BuildContext: base}, nil
}

func CreateTagger(cfg config.Config) (Tagger, error) {
	return CreateBuilder(cfg, nil)
}

func CreateBuilder(cfg config.Config, deliverable *config.Deliverable) (Builder, error) {

	client, err := docker.NewDockerClient()

	if err != nil {
		return nil, errors.Wrap(err, "Error initializing Docker")
	}

	provider := docker.NewRegistryClient(cfg.DockerSpec.ExternalDockerRegistry)

	if config.IsSemantic(cfg.MavenGav.Version) {
		return NewReleaseBuilder(cfg, *deliverable, provider, *client)
	} else if config.IsSnapshot(cfg.MavenGav.Version) {
		return NewSnapshotBuilder(cfg, *deliverable, provider, *client)
	} else {
		return nil, nil
	}
}

func (b *JavaReleaseBuilder) Build() error {
	baseImageVersion, err := getBaseImageVersion(b.imageInfoProvider, b.cfg)

	if err != nil {
		return err
	}

	ver := config.GetReleaseVersion(b.cfg, baseImageVersion)

	return doBuild(b.cfg, b.deliverable, b.client, nil, ver)
}

func (b *JavaReleaseBuilder) Tag() error {
	baseImageVersion, err := getBaseImageVersion(b.imageInfoProvider, b.cfg)

	if err != nil {
		return err
	}

	ver := config.GetReleaseVersion(b.cfg, baseImageVersion)

	repoTags, err := b.imageInfoProvider.GetTags(b.cfg.DockerSpec.OutputRepository)

	if err != nil {
		return errors.Wrapf(err, "Error in GetTags, repository=%s", b.cfg.DockerSpec.OutputRepository)
	}

	tags, err := config.GetFilteredReleaseVersionTags(ver, b.cfg.DockerSpec.PushExtraTags, repoTags.Tags)

	if err != nil {
		return err
	}

	return doTag(b.cfg, b.client, tags)
}

func (b *JavaReleaseBuilder) TagTemporary() error {
	tags := config.GetTemporaryVersionTags(b.cfg)
	return doTag(b.cfg, b.client, tags)
}

func (b *JavaReleaseBuilder) RetagTemporary() error {
	tempTag := b.cfg.DockerSpec.RetagWith
	repository := b.cfg.DockerSpec.OutputRepository

	envMap, err := b.imageInfoProvider.GetManifestEnvMap(repository, tempTag)

	if err != nil {
		return err
	}

	auroraVersion, ok := envMap[docker.ENV_AURORA_VERSION]

	if !ok {
		return errors.Errorf("Failed to extract ENV variable %s from temporary image manifest", docker.ENV_AURORA_VERSION)
	}

	appVersion, ok := envMap[docker.ENV_APP_VERSION]

	if !ok {
		return errors.Errorf("Failed to extract ENV variable %s from temporary image manifest", docker.ENV_APP_VERSION)
	}

	ver := config.VersionInfo{appVersion, auroraVersion}

	extraTags, ok := envMap[docker.ENV_PUSH_EXTRA_TAGS]

	if !ok {
		return errors.Errorf("Failed to extract ENV variable %s from temporary image manifest", docker.ENV_PUSH_EXTRA_TAGS)
	}

	repoTags, err := b.imageInfoProvider.GetTags(b.cfg.DockerSpec.OutputRepository)

	if err != nil {
		return errors.Wrapf(err, "Error in GetTags, repository=%s", b.cfg.DockerSpec.OutputRepository)
	}

	tags, err := config.GetFilteredReleaseVersionTags(ver, extraTags, repoTags.Tags)

	if err != nil {
		return err
	}

	return doTag(b.cfg, b.client, tags)
}

func (b JavaSnapshotBuilder) Build() error {
	baseImageVersion, err := getBaseImageVersion(b.imageInfoProvider, b.cfg)

	if err != nil {
		return err
	}

	ver := config.GetSnapshotVersion(b.cfg, b.deliverable.Path, baseImageVersion)
	return doBuild(b.cfg, b.deliverable, b.client, nil, ver)
}

func (b JavaSnapshotBuilder) Tag() error {
	baseImageVersion, err := getBaseImageVersion(b.imageInfoProvider, b.cfg)

	if err != nil {
		return err
	}

	ver := config.GetSnapshotVersion(b.cfg, b.deliverable.Path, baseImageVersion)

	tags := config.GetSnapshotVersionTags(ver)

	return doTag(b.cfg, b.client, tags)
}

func (b JavaSnapshotBuilder) TagTemporary() error {
	tags := config.GetTemporaryVersionTags(b.cfg)
	return doTag(b.cfg, b.client, tags)
}

func (b JavaSnapshotBuilder) RetagTemporary() error {
	tempTag := b.cfg.DockerSpec.RetagWith
	repository := b.cfg.DockerSpec.OutputRepository

	envMap, err := b.imageInfoProvider.GetManifestEnvMap(repository, tempTag)

	if err != nil {
		return err
	}

	snapshotTag, ok := envMap[docker.ENV_SNAPSHOT_TAG]

	if !ok {
		return errors.Errorf("Failed to extract ENV variable %s from temporary image manifest", docker.ENV_SNAPSHOT_TAG)
	}

	return doTag(b.cfg, b.client, []string{snapshotTag})
}

func doBuild(cfg config.Config, deliverable config.Deliverable, client docker.DockerClient, env map[string]string, v config.VersionInfo) error {

	logrus.Debug("Prepare output image")
	path, err := prepare.Prepare(v.AppVersion, v.AuroraVersion, deliverable)

	if err != nil {
		return errors.Wrap(err, "Error prepare artifact")
	}

	logrus.Debugf("Build and push docker image, path=%s", path)
	tagsToPush := createTags([]string{v.AuroraVersion}, cfg.DockerSpec)

	buildConf := docker.DockerBuildConfig{
		Tags:        tagsToPush,
		BuildFolder: path,
	}

	imageid, err := client.BuildImage(buildConf)

	if err != nil {
		return errors.Wrap(err, "Fuckup!")
	} else {
		logrus.Infof("Done building. Imageid: %s", imageid)
	}

	return nil
}

func doTag(cfg config.Config, client docker.DockerClient, tags []string) error {

	tagsToPush := createTags(tags, cfg.DockerSpec)

	logrus.Debug("Push tags")

	err := client.PushImages(tagsToPush, cfg.DockerSpec.OutputRegistryCredentials)
	if err != nil {
		return errors.Wrap(err, "Error pushing images")
	}

	return nil
}


func createTags(tags []string, dockerSpec config.DockerSpec) []string {
	output := make([]string, len(tags))
	for i, t := range tags {
		name := &docker.ImageName{dockerSpec.OutputRegistry,dockerSpec.OutputRepository, t}
		output[i] = name.String()
	}
	return output
}

func getBaseImageVersion(provider docker.ImageInfoProvider, cfg config.Config) (string, error) {
	biv, err := provider.GetManifestEnv(cfg.DockerSpec.BaseImage, cfg.DockerSpec.BaseVersion, "BASE_IMAGE_VERSION")

	if err != nil {
		return "", errors.Wrap(err, "Failed to extract version in getBaseImageVersion")
	} else if biv == "" {
		return "", errors.Errorf("Failed to extract version in getBaseImageVersion, registry: %s, "+
			"BaseImage: %s, BaseVersion: %s ",
			provider, cfg.DockerSpec.BaseImage, cfg.DockerSpec.BaseVersion)
	}
	return biv, nil
}


