package cmd

import (
	"context"
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"time"

	"github.com/stephenafamo/warden/models"
)

type FilePathAndInfo struct {
	os.FileInfo
	Path string
}

func r(db *sql.DB) {
	if r := recover(); r != nil {
		log.Println("PANIC OCCURRED:", r)
		PurgeConfigFiles(db) // Used to reset all configuration
		debug.PrintStack()
	}
}

func startConfigDirectoryWatcher(db *sql.DB) error {
	duration, err := time.ParseDuration(settings.ReloadDuration)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(duration)
	go func() {
		for _ = range ticker.C {
			WalkConfigDirectory(db)
		}
	}()

	return nil
}

func startFileServicesConfigurator(db *sql.DB) error {
	duration, err := time.ParseDuration(settings.ReloadDuration)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(duration)
	go func() {
		for _ = range ticker.C {
			ConfigureFileServices(db)
		}
	}()
	return nil
}

func startServicesNginxConfigGenerator(db *sql.DB) error {
	duration, err := time.ParseDuration(settings.ReloadDuration)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(duration)
	go func() {
		for _ = range ticker.C {
			GenerateNginxConfig(db)
		}
	}()
	return nil
}

func setFilesInfo(filepaths *[]interface{}, files *[]FilePathAndInfo) filepath.WalkFunc {
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

func checkFile(db *sql.DB, file FilePathAndInfo, wg *sync.WaitGroup) {
	defer r(db)
	defer wg.Done()

	ctx := context.Background()
	oldFile, err := models.Files(models.FileWhere.Path.EQ(file.Path)).One(ctx, db)
	if err == sql.ErrNoRows {
		err = addFile(db, file)
		if err != nil {
			panic(err)
		}
		return
	}
	if err != nil {
		panic(err)
	}

	if file.ModTime().After(oldFile.LastModified) {
		err = updateFile(db, oldFile, file)
		if err != nil {
			panic(err)
		}
	}
}

func DoConfigureFileServices(db *sql.DB, file *models.File, wg *sync.WaitGroup) {
	defer r(db)
	defer wg.Done()

	ctx := context.Background()
	err := configureServices(db, file)
	if err != nil {
		panic(err)
	}

	_, err = models.Files(
		models.FileWhere.ID.EQ(file.ID),
		models.FileWhere.LastModified.EQ(file.LastModified),
	).UpdateAll(
		ctx,
		db,
		models.M{"is_configured": true},
	)
	if err != nil {
		panic(err)
	}

	log.Printf("RECONFIGURED SERVICES FOR: %s \n", file.Path)
}
