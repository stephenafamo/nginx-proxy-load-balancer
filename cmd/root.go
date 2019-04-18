package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/sqlboiler/boil"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "warden",
	Short: "Setup and manage a reverse proxy",
	Long:  "Setup and manage a reverse proxy",
	RunE:  rootFunc,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	viper.AutomaticEnv() // read in environment variables that match
	settings.DbPath = "~/db?_fk=1"
	settings.Email = viper.GetString("EMAIL")
	settings.ConfigDir = viper.GetString("CONFIG_DIR")
	settings.ReloadDuration = viper.GetString("CONFIG_RELOAD_TIME")
}

func rootFunc(cmd *cobra.Command, args []string) error {
	boil.DebugMode = false

	fmt.Println("running dev.sh")
	c := exec.Command("./dev.sh")
	err := c.Run()
	if err != nil {
		return err
	}

	fmt.Println("Connecting to DB...")
	db, err := sql.Open("sqlite3", settings.DbPath)
	if err != nil {
		return err
	}

	err = createTables(db)
	if err != nil {
		return err
	}
	defer db.Close()

	err = startConfigDirectoryWatcher(db)
	if err != nil {
		return err
	}

	err = startFileServicesConfigurator(db)
	if err != nil {
		return err
	}

	err = startServicesNginxConfigGenerator(db)
	if err != nil {
		return err
	}

	err = startNginx()
	if err != nil {
		return err
	}

	return nil
}

func startNginx() error {
	fmt.Println("starting nginx LOL")
	cmd := exec.Command("tail", "-f", "./dev.sh")
	// cmd = exec.Command("service", "nginx", "reload")
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func reloadNginx() error {
	fmt.Println("Reloading NGINX")
	cmd := exec.Command("nginx", "-s", "reload")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Failed to reload NGINX %s", err)
	}
	return nil
}
