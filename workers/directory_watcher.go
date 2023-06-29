package workers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/stephenafamo/janus/monitor"
	"github.com/stephenafamo/kronika"
	"github.com/stephenafamo/warden/internal"
	"github.com/stephenafamo/warden/models"
	"github.com/volatiletech/sqlboiler/v4/boil"
)

type FilePathAndInfo struct {
	os.FileInfo
	Path string
}

type DirectoryWatcher struct {
	DB       *sql.DB
	Monitor  monitor.Monitor
	Settings internal.Settings
}

func (d DirectoryWatcher) Play(ctx context.Context) error {
	for range kronika.Every(ctx, time.Now(), d.Settings.CONFIG_RELOAD_TIME) {
		err := d.WalkConfigDirectory(context.Background()) // use new context
		if err != nil {
			err = fmt.Errorf("error walking config dir: %w", err)
			d.Monitor.CaptureException(err, nil)
		}
	}

	return nil
}

func (d DirectoryWatcher) WalkConfigDirectory(ctx context.Context) error {
	var wg sync.WaitGroup
	var filepaths []string
	var files []FilePathAndInfo

	err := filepath.Walk(
		d.Settings.CONFIG_DIR,
		setFilesInfo(&filepaths, &files),
	)
	if err != nil {
		return fmt.Errorf("error while walking config directory: %w", err)
	}

	wg.Add(len(files))
	for _, file := range files {
		d.checkFile(ctx, file, &wg)
	}
	wg.Wait()

	// Delete all files if there's no filepaths
	query := models.Files()
	if len(filepaths) > 0 {
		query = models.Files(models.FileWhere.Path.NIN(filepaths))
	}

	_, err = query.DeleteAll(ctx, d.DB)
	if err != nil {
		return fmt.Errorf("error deleting redundant filepaths: %w", err)
	}

	return nil
}

func setFilesInfo(filepaths *[]string, files *[]FilePathAndInfo) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".toml" {
			return nil
		}
		*filepaths = append(*filepaths, path)
		*files = append(*files, FilePathAndInfo{FileInfo: info, Path: path})
		return nil
	}
}

func (d DirectoryWatcher) checkFile(ctx context.Context, file FilePathAndInfo, wg *sync.WaitGroup) {
	defer wg.Done()

	oldFile, err := models.Files(models.FileWhere.Path.EQ(file.Path)).One(ctx, d.DB)
	if err == sql.ErrNoRows {
		err = d.addFile(ctx, file)
		if err != nil {
			err = fmt.Errorf("error adding file to DB: %w", err)
			d.Monitor.CaptureException(err, nil)
		}
		return
	}
	if err != nil {
		err = fmt.Errorf("error getting file from DB: %w", err)
		d.Monitor.CaptureException(err, nil)
		return
	}

	if file.ModTime().After(oldFile.LastModified) {
		err = d.updateFile(ctx, oldFile, file)
		if err != nil {
			err = fmt.Errorf("error updating file in DB: %w", err)
			d.Monitor.CaptureException(err, nil)
			return
		}
	}
}

func (d DirectoryWatcher) addFile(ctx context.Context, file FilePathAndInfo) error {
	content, err := getFileContent(file.Path)
	if err != nil {
		return fmt.Errorf("error adding file: %w", err)
	}

	fModel := models.File{
		Name:         strings.TrimSuffix(file.Name(), filepath.Ext(file.Name())),
		Path:         file.Path,
		Content:      content,
		LastModified: file.ModTime(),
		IsConfigured: false,
	}

	err = fModel.Insert(ctx, d.DB, boil.Infer())
	if err != nil {
		return fmt.Errorf("error inserting file %d in db: %w", fModel.ID, err)
	}

	log.Printf("ADDED: %s", file.Path)
	return nil
}

func (d DirectoryWatcher) updateFile(ctx context.Context, oldFile *models.File, file FilePathAndInfo) error {
	content, err := getFileContent(file.Path)
	if err != nil {
		return fmt.Errorf("error updating file: %w", err)
	}

	oldFile.Content = content
	oldFile.IsConfigured = false
	oldFile.LastModified = file.ModTime()

	_, err = oldFile.Update(ctx, d.DB, boil.Infer())
	if err != nil {
		return fmt.Errorf("error updating file %d in db: %w", oldFile.ID, err)
	}
	log.Printf("UPDATED: %s", file.Path)
	return nil
}

func getFileContent(path string) (internal.ServiceMap, error) {
	var configs map[string]internal.Service
	if _, err := toml.DecodeFile(path, &configs); err != nil {
		err = fmt.Errorf("could not decode file: %w", err)
		return nil, err
	}

	return configs, nil
}
