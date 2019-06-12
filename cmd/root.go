package cmd

import (
	"database/sql"
	"log"
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
		log.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	viper.AutomaticEnv() // read in environment variables that match
	settings.DbPath = "./db"
	settings.Email = viper.GetString("EMAIL")
	settings.ConfigDir = viper.GetString("CONFIG_DIR")
	settings.ReloadDuration = viper.GetString("CONFIG_RELOAD_TIME")
	settings.PurgeDuration = viper.GetString("CONFIG_VALIDITY")
}

func rootFunc(cmd *cobra.Command, args []string) error {
	boil.DebugMode = false

	log.Println("Cleaning up...")
	c := exec.Command(
		"/bin/sh", 
		"-c", 
		"rm -rf "+settings.DbPath+" /etc/nginx/conf.d/http/* /etc/nginx/conf.d/streams/*",
	)

	output, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"Error Cleaning UP: %s: %s",
			err,
			output,
		)
	}

	log.Println("Connecting to DB...")
	db, err := sql.Open("sqlite3", settings.DbPath+"?_fk=1")
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
	log.Println("Starting NGINX")
	cmd := exec.Command("nginx", "-g", "daemon off;")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"Can't start NGINX: %s: %s",
			err,
			output,
		)
	}
	return nil
}

func reloadNginx() error {
	log.Println("Reloading NGINX")
	cmd := exec.Command("nginx", "-s", "reload")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"Failed to reload NGINX: %s: %s",
			err,
			output,
		)
	}
	return nil
}
