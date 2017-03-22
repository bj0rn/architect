package architect

import (
	"github.com/skatteetaten/architect/pkg/config"
	"github.com/skatteetaten/architect/pkg/log"
	"github.com/spf13/cobra"
)

var JavaLeveransepakke = &cobra.Command{
	Use:   "build",
	Short: "Build Docker image from Zip",
	Long:  `TODO`,
	Run: func(cmd *cobra.Command, args []string) {
		var configReader = config.NewInClusterConfigReader()
		if len(cmd.Flag("fileconfig").Value.String()) != 0 {
			configReader = config.NewFileConfigReader(cmd.Flag("fileconfig").Value.String())
		}
		RunArchitect(configReader)
	},
}

func init() {
	JavaLeveransepakke.Flags().StringP("fileconfig", "f", "", "Path to file config. If not set, the environment variable BUILD is read")
	JavaLeveransepakke.Flags().StringP("skippush", "s", "", "If set, Docker push will not be performed")
	JavaLeveransepakke.Flags().StringP("localrepo", "l", "", "If set, the Leveransepakke will be fetched from the local repo")
}

func RunArchitect(configReader config.ConfigReader) {
	c, err := configReader.ReadConfig()
	if err != nil {
		log.Error.Fatalf("Could not read configuration: %s", err)
	}
	log.Info.Printf("%+v", c)
}
