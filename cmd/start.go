package cmd

import (
	"database/sql"
	"fmt"

	"github.com/getsentry/sentry-go"
	"github.com/stephenafamo/janus/monitor"
	jSentry "github.com/stephenafamo/janus/monitor/sentry"
	"github.com/stephenafamo/orchestra"
	"github.com/stephenafamo/warden/internal"
	"github.com/stephenafamo/warden/workers"
)

func setPlayers(db *sql.DB, settings internal.Settings, mon monitor.Monitor) (map[string]orchestra.Player, error) {
	templates, err := internal.GetTemplates()
	if err != nil {
		return nil, fmt.Errorf("could not get templates: %w", err)
	}

	players := map[string]orchestra.Player{}

	players["directory-watcher"] = workers.DirectoryWatcher{
		DB:       db,
		Monitor:  mon,
		Settings: settings,
	}

	players["service-configurer"] = workers.ServiceConfigurer{
		DB:       db,
		Monitor:  mon,
		Settings: settings,
	}

	players["nginx-generator"] = workers.NginxGenerator{
		DB:        db,
		Monitor:   mon,
		Settings:  settings,
		Templates: templates,
	}

	players["nginx-server"] = workers.NginxServer{
		Settings: settings,
		Monitor:  mon,
	}

	return players, nil
}

func getMonitor(config internal.Settings) (monitor.Monitor, error) {
	options := sentry.ClientOptions{
		Dsn:   config.SENTRY_DSN,
		Debug: true,
	}

	// Add logging during testing
	if config.TESTING || config.SENTRY_DSN == "" {
		options.Integrations = func(in []sentry.Integration) []sentry.Integration {
			return append(in, jSentry.LoggingIntegration{
				Logger:        sentryLogger{},
				SupressErrors: config.TESTING,
			})
		}
	}

	// Get the sentry client
	client, err := sentry.NewClient(options)
	if err != nil {
		return nil, fmt.Errorf("could not create sentry client: %w", err)
	}

	hub := sentry.NewHub(client, sentry.NewScope())
	return jSentry.Sentry{Hub: hub}, nil
}

type sentryLogger struct{}

func (sentryLogger) Printf(format string, a ...interface{}) (n int, err error) {
	return fmt.Printf(format, a...)
}
