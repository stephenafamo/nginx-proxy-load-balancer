package cmd

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/stephenafamo/warden/models"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

func WatchConfigDirectory(db *sql.DB) {
	defer r(db)
	var wg sync.WaitGroup
	var filepaths []interface{}
	var files []FilePathAndInfo
	var ctx = context.Background()

	err := filepath.Walk(
		settings.ConfigDir,
		setFilesInfo(&filepaths, &files),
	)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		wg.Add(1)
		go checkFile(db, file, &wg)
	}

	wg.Wait()

	_, err = models.Files(
		qm.WhereIn("path NOT IN ?", filepaths...),
	).DeleteAll(ctx, db)
	if err != nil {
		panic(err)
	}
}

func ConfigureFileServices(db *sql.DB) {
	defer r(db)
	var wg sync.WaitGroup
	var ctx = context.Background()

	files, err := models.Files(
		models.FileWhere.IsConfigured.EQ(false),
	).All(ctx, db)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		wg.Add(1)
		go DoConfigureFileServices(db, file, &wg)
	}

	wg.Wait()

	servicesToDelete, err := models.Services(
		qm.InnerJoin("files on files.id = services.file_id"),
		qm.Where("services.last_modified < files.last_modified"),
	).All(ctx, db)
	if err != nil {
		panic(err)
	}

	if len(servicesToDelete) > 0 {
		_, err = servicesToDelete.DeleteAll(ctx, db)
		if err != nil {
			panic(err)
		}
	}
}

func GenerateNginxConfig(db *sql.DB) {
	defer r(db)
	var wg sync.WaitGroup
	var ctx = context.Background()

	nginxFiles, err := models.NginxConfigFiles(
		models.NginxConfigFileWhere.ServiceID.IsNull(),
	).All(ctx, db)
	if err != nil {
		panic(err)
	}

	if len(nginxFiles) > 0 {
		for _, file := range nginxFiles {
			defer r(db)
			c := exec.Command("rm", "-f", file.Path)
			err = c.Run()
			if err != nil {
				panic(err)
			}
		}

		nginxFiles.DeleteAll(ctx, db)
	}

	services, err := models.Services(
		qm.Load(models.ServiceRels.File),
		models.ServiceWhere.State.EQ(stateNotConfigured),
	).All(ctx, db)
	if err != nil {
		panic(err)
	}

	if len(services) > 0 {
		for _, service := range services {
			wg.Add(1)
			go generateBaseConfig(db, service, &wg)
		}

		wg.Wait()

		err = reloadNginx()
		if err != nil {
			panic(err)
		}
	}

	services, err = models.Services(
		qm.Load(models.ServiceRels.File),
		models.ServiceWhere.State.EQ(stateToConfigureHttps),
	).All(ctx, db)
	if err != nil {
		panic(err)
	}

	if len(services) > 0 {
		for _, service := range services {
			// has to be sequential, can only ask for one certificate at a time
			generateHttpsConfig(db, service) 
		}

		err = reloadNginx()
		if err != nil {
			panic(err)
		}
	}

	services, err = models.Services(
		qm.Load(models.ServiceRels.File),
		models.ServiceWhere.State.EQ(stateToDisableHttp),
	).All(ctx, db)
	if err != nil {
		panic(err)
	}

	if len(services) > 0 {
		for _, service := range services {
			wg.Add(1)
			go redirectToHttpsConfig(db, service, &wg)
		}

		wg.Wait()

		err = reloadNginx()
		if err != nil {
			panic(err)
		}
	}
}

func getFullConfig(s *models.Service) ConfigTemplateStruct {
	var config ServiceConfig

	_, err := toml.Decode(s.Content, &config)
	if err != nil {
		panic(err)
	}

	if config.Type == "" {
		config.Type = "http"
	}

	if config.Location == "" && len(config.Locations) == 0 {
		config.Location = "/"
	}

	tStruct := ConfigTemplateStruct{
		ServiceConfig: config,
		Unique:        s.Name + "-" + s.R.File.Name + "-" + strconv.FormatInt(s.ID, 10),
	}

	return tStruct
}

func generateBaseConfig(db *sql.DB, s *models.Service, wg *sync.WaitGroup) {
	defer r(db)
	defer wg.Done()

	var err error
	var b bytes.Buffer
	var ctx = context.Background()

	config := getFullConfig(s)

	configDirectory := ""
	fileType := ""
	configContents := []byte{}

	switch strings.ToLower(config.Type) {
	case "tcp", "udp", "stream":
		fileType = "stream"
		configDirectory = "/etc/nginx/conf.d/streams"
		err = t.ExecuteTemplate(&b, "streams", config)
		if err != nil {
			panic(err)
		}
		configContents = b.Bytes()
	case "http":
		fileType = "http"
		configDirectory = "/etc/nginx/conf.d/http"
		err = t.ExecuteTemplate(&b, "httpBase", config)
		if err != nil {
			panic(err)
		}
		configContents = b.Bytes()
	default:
		panic(errors.New(fmt.Sprintf(
			"Unknown config type for service %q in file %q",
			s.Name,
			s.R.File.Path,
		)))
	}

	ok, unreachableUpstream := pingUpstreams(config)

	if !ok {
		fmt.Printf(
			"Cannot reach upstream %q for service %q in file %q\n",
			unreachableUpstream,
			s.Name,
			s.R.File.Path,
		)
		return
	}

	configPath := filepath.Join(configDirectory, config.Unique+".conf")
	err = ioutil.WriteFile(configPath, configContents, 0644)
	if err != nil {
		panic(err)
	}

	ngf := &models.NginxConfigFile{
		Type:         fileType,
		Path:         configPath,
		LastModified: s.LastModified,
	}

	err = s.AddNginxConfigFiles(ctx, db, true, ngf)
	if err != nil {
		panic(err)
	}

	s.State = stateConfigured
	if strings.ToLower(config.Type) == "http" && config.Ssl {
		s.State = stateToConfigureHttps
	}

	_, err = s.Update(ctx, db, boil.Infer())
	if err != nil {
		panic(err)
	}

	fmt.Printf("CONFIGURED BASE FOR: %s \n", s.Name)
}

func generateHttpsConfig(db *sql.DB, s *models.Service) {
	defer r(db)

	var err error
	var b bytes.Buffer
	var ctx = context.Background()

	config := getFullConfig(s)

	err = setSslCertificatePath(&config)
	if err != nil {
		panic(err)
	}

	configDirectory := "/etc/nginx/conf.d/http"
	fileType := "https"

	err = t.ExecuteTemplate(&b, "https", config)
	if err != nil {
		panic(err)
	}
	configContents := b.Bytes()

	configPath := filepath.Join(configDirectory, config.Unique+".SSL.conf")
	err = ioutil.WriteFile(configPath, configContents, 0644)
	if err != nil {
		panic(err)
	}

	ngf := &models.NginxConfigFile{
		Type:         fileType,
		Path:         configPath,
		LastModified: s.LastModified,
	}

	err = s.AddNginxConfigFiles(ctx, db, true, ngf)
	if err != nil {
		panic(err)
	}

	s.State = stateConfigured
	if config.HttpsOnly {
		s.State = stateToDisableHttp
	}

	_, err = s.Update(ctx, db, boil.Infer())
	if err != nil {
		panic(err)
	}

	fmt.Printf("CONFIGURED HTTPS FOR: %s \n", s.Name)
}

func redirectToHttpsConfig(db *sql.DB, s *models.Service, wg *sync.WaitGroup) {
	defer r(db)
	defer wg.Done()

	var err error
	var b bytes.Buffer
	var ctx = context.Background()

	config := getFullConfig(s)

	ngf, err := s.NginxConfigFiles(
		models.NginxConfigFileWhere.Type.EQ("http"),
	).One(ctx, db)

	err = t.ExecuteTemplate(&b, "httptoHttps", config)
	if err != nil {
		panic(err)
	}
	configContents := b.Bytes()

	err = ioutil.WriteFile(ngf.Path, configContents, 0644)
	if err != nil {
		panic(err)
	}

	s.State = stateConfigured

	_, err = s.Update(ctx, db, boil.Infer())
	if err != nil {
		panic(err)
	}

	fmt.Printf("CONFIGURED HTTPS ONLY FOR: %s \n", s.Name)
}

func pingUpstreams(config ConfigTemplateStruct) (bool, string) {
	for _, u := range config.Upstream {
		host := strings.Split(u.Address, ":")[0]

		fmt.Printf("PINGING %q\n", host)

		cmd := exec.Command("ping", "-c", "1", host)
		err := cmd.Run()
		if err != nil {
			return false, host
		}
	}

	return true, ""
}

func setSslCertificatePath(config *ConfigTemplateStruct) error {

	switch config.SslSource {
	case "manual":
		return nil

	case "letsencrypt":
		CertPath, KeyPath, err := getLetsEncryptCertificate(config)
		config.CertPath = CertPath
		config.KeyPath = KeyPath
		return err

	default:
		return fmt.Errorf("Unknown SSL source %q", config.SslSource)
	}

	return nil
}
