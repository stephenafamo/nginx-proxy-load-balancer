package workers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/stephenafamo/janus/monitor"
	"github.com/stephenafamo/kronika"
	"github.com/stephenafamo/warden/internal"
	"github.com/stephenafamo/warden/models"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

type ServiceConfigurer struct {
	DB       *sql.DB
	Monitor  monitor.Monitor
	Settings internal.Settings
}

func (s ServiceConfigurer) Play(ctx context.Context) error {
	for range kronika.Every(ctx, time.Now(), s.Settings.CONFIG_RELOAD_TIME) {
		err := s.setFileServices(context.Background()) // use new context
		if err != nil {
			err = fmt.Errorf("error configuring services: %w", err)
			s.Monitor.CaptureException(err, nil)
		}
	}

	return nil
}

func (s ServiceConfigurer) setFileServices(ctx context.Context) error {
	var wg sync.WaitGroup

	// Get all the files
	files, err := models.Files(
		models.FileWhere.IsConfigured.EQ(false),
	).All(ctx, s.DB)
	if err != nil {
		return fmt.Errorf("could not retrieve files from DB: %w", err)
	}

	wg.Add(len(files))
	for _, file := range files {
		go s.createFileServices(ctx, file, &wg)
	}
	wg.Wait()

	servicesToDelete, err := models.Services(
		qm.InnerJoin(fmt.Sprintf(
			"%s on %s.%s = %s.%s",
			models.TableNames.Files,
			models.TableNames.Files,
			models.FileColumns.ID,
			models.TableNames.Services,
			models.ServiceColumns.FileID,
		)),
		qm.Where(fmt.Sprintf(
			"%s.%s < %s.%s",
			models.TableNames.Services,
			models.ServiceColumns.LastModified,
			models.TableNames.Files,
			models.FileColumns.LastModified,
		)),
	).All(ctx, s.DB)
	if err != nil {
		return fmt.Errorf("could not get services to delete: %w", err)
	}

	if len(servicesToDelete) > 0 {
		_, err = servicesToDelete.DeleteAll(ctx, s.DB)
		if err != nil {
			return fmt.Errorf("could not delete services from DB: %w", err)
		}
	}

	return nil
}

func (s ServiceConfigurer) createFileServices(ctx context.Context, file *models.File, wg *sync.WaitGroup) {
	defer wg.Done()

	services := models.ServiceSlice{}

	for key, config := range file.Content {

		service := &models.Service{
			Name:         key,
			Content:      config,
			IsSSL:        config.Ssl,
			State:        internal.StateNotConfigured,
			LastModified: file.LastModified,
		}

		services = append(services, service)
	}

	// Just add a new relationship. setFileServices cleans the old ones
	if err := file.AddServices(ctx, s.DB, true, services...); err != nil {
		err = fmt.Errorf("could not add file services: %w", err)
		s.Monitor.CaptureException(err, nil)
		return
	}

	log.Printf("ADDED SERVICES FOR: %s\n", file.Path)

	// Mark the file as configured in the DB
	file.IsConfigured = true
	if _, err := file.Update(ctx, s.DB, boil.Infer()); err != nil {
		err = fmt.Errorf("could not update file: %w", err)
		s.Monitor.CaptureException(err, nil)
		return
	}

	log.Printf("RECONFIGURED SERVICES FOR: %s \n", file.Path)
}
