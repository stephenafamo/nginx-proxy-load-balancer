package cmd

import (
	"context"
	"database/sql"
	"time"

	"github.com/stephenafamo/warden/models"
)

var lastPurge time.Time

func PurgeConfigFiles(db *sql.DB) error {
	duration, err := time.ParseDuration(settings.ReloadDuration)
	if err != nil {
		return err
	}

	if time.Now().Sub(lastPurge) < duration*5 {
		return nil
	}

	_, err = models.Files().DeleteAll(context.Background(), db)
	return err
}
