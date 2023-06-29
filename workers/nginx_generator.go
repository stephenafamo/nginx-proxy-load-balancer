package workers

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/stephenafamo/janus/monitor"
	"github.com/stephenafamo/kronika"
	"github.com/stephenafamo/warden/internal"
	"github.com/stephenafamo/warden/letsencrypt"
	"github.com/stephenafamo/warden/models"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

type NginxGenerator struct {
	DB        *sql.DB
	Monitor   monitor.Monitor
	Settings  internal.Settings
	Templates *template.Template
}

func (n NginxGenerator) Play(ctx context.Context) error {
	for range kronika.Every(ctx, time.Now(), n.Settings.CONFIG_RELOAD_TIME) {
		err := n.GenerateNginxConfig(context.Background()) // use new context
		if err != nil {
			err = fmt.Errorf("error generating nginx configs: %w", err)
			n.Monitor.CaptureException(err, nil)
		}
	}

	return nil
}

func (n NginxGenerator) GenerateNginxConfig(ctx context.Context) error {
	err := n.deleteStaleConfigs(ctx)
	if err != nil {
		return fmt.Errorf("could not clean up stale configs: %w", err)
	}

	err = n.generateBaseConfigs(ctx)
	if err != nil {
		return fmt.Errorf("could not generate base configs: %w", err)
	}

	err = n.generateHttpsConfigs(ctx)
	if err != nil {
		return fmt.Errorf("could not generate https configs: %w", err)
	}

	err = n.generateNoHttpConfigs(ctx)
	if err != nil {
		return fmt.Errorf("could not generate no-http configs: %w", err)
	}

	return nil
}

func (n NginxGenerator) deleteStaleConfigs(ctx context.Context) error {
	nginxFiles, err := models.NginxConfigs(
		models.NginxConfigWhere.ServiceID.IsNull(),
	).All(ctx, n.DB)
	if err != nil {
		return fmt.Errorf("could not get stale nginx config files: %w", err)
	}

	// No stale files
	if len(nginxFiles) == 0 {
		return nil
	}

	for _, file := range nginxFiles {
		c := exec.Command("rm", "-f", file.Path)
		err = c.Run()
		if err != nil {
			return fmt.Errorf("could not delete nginx config file %q: %w", file.Path, err)
		}
	}

	_, err = nginxFiles.DeleteAll(ctx, n.DB)
	if err != nil {
		return fmt.Errorf("could not delete stale nginx configs from DB: %w", err)
	}

	return nil
}

func (n NginxGenerator) generateBaseConfigs(ctx context.Context) error {
	var wg sync.WaitGroup

	services, err := models.Services(
		qm.Load(models.ServiceRels.File),
		models.ServiceWhere.State.EQ(internal.StateNotConfigured),
	).All(ctx, n.DB)
	if err != nil {
		return fmt.Errorf("could not get unconfigured services: %w", err)
	}

	if len(services) == 0 {
		return nil
	}

	wg.Add(len(services))
	for _, service := range services {
		go n.generateBaseConfig(ctx, service, &wg)
	}
	wg.Wait()

	err = n.reloadNginx()
	if err != nil {
		return fmt.Errorf("could not reload nginx: %w", err)
	}

	return nil
}

func (n NginxGenerator) generateHttpsConfigs(ctx context.Context) error {
	services, err := models.Services(
		models.ServiceWhere.State.EQ(internal.StateToConfigureHttps),
		qm.Or2(
			qm.Expr(
				models.ServiceWhere.IsSSL.EQ(true),
				models.ServiceWhere.HTTPSConfigured.LT(
					null.TimeFrom(time.Now().Add(-n.Settings.HTTPS_VALIDITY)),
				),
			),
		),
		qm.Load(models.ServiceRels.File),
	).All(ctx, n.DB)
	if err != nil {
		return fmt.Errorf("could not get services to configure https: %w", err)
	}

	if len(services) == 0 {
		return nil
	}

	for _, service := range services {
		// Can only ask for one certificate at a time. Must be sequential
		n.generateHttpsConfig(ctx, service)
	}

	err = n.reloadNginx()
	if err != nil {
		return fmt.Errorf("could not reload nginx: %w", err)
	}

	return nil
}

func (n NginxGenerator) generateNoHttpConfigs(ctx context.Context) error {
	var wg sync.WaitGroup

	services, err := models.Services(
		models.ServiceWhere.State.EQ(internal.StateToDisableHttp),
		qm.Load(models.ServiceRels.File),
	).All(ctx, n.DB)
	if err != nil {
		return fmt.Errorf("could not get services to configure https only: %w", err)
	}

	if len(services) == 0 {
		return nil
	}

	wg.Add(len(services))
	for _, service := range services {
		go n.generateNoHttpConfig(ctx, service, &wg)
	}
	wg.Wait()

	err = n.reloadNginx()
	if err != nil {
		return fmt.Errorf("could not reload nginx: %w", err)
	}

	return nil
}

func (n NginxGenerator) generateBaseConfig(ctx context.Context, s *models.Service, wg *sync.WaitGroup) {
	defer wg.Done()

	var err error
	var b bytes.Buffer

	config, err := n.getFullConfig(s)
	if err != nil {
		err = fmt.Errorf("could not get full config: %w", err)
		n.Monitor.CaptureException(err, nil)
		return
	}

	configDirectory := ""
	fileType := ""
	var configContents []byte

	switch strings.ToLower(config.Type) {
	case "tcp", "udp", "stream":
		fileType = "stream"
		configDirectory = filepath.Join(n.Settings.CONFIG_OUTPUT_DIR, "streams")
		err = n.Templates.ExecuteTemplate(&b, "streams", config)
		if err != nil {
			err = fmt.Errorf("error generating stream base config for %q in %q: %w", s.Name, s.R.File.Path, err)
			n.Monitor.CaptureException(err, nil)
			return
		}
		configContents = b.Bytes()
	case "http":
		fileType = "http"
		configDirectory = filepath.Join(n.Settings.CONFIG_OUTPUT_DIR, "http")
		err = n.Templates.ExecuteTemplate(&b, "httpBase", config)
		if err != nil {
			err = fmt.Errorf("error generating http base config for %q in %q: %w", s.Name, s.R.File.Path, err)
			n.Monitor.CaptureException(err, nil)
			return
		}
		configContents = b.Bytes()
	default:
		err = fmt.Errorf("Unknown config type for %q in %q", s.Name, s.R.File.Path)
		n.Monitor.CaptureException(err, nil)
		return
	}

	ok, unreachableUpstream := n.pingUpstreams(config)

	if !ok {
		log.Printf("Cannot reach upstream %q for %q in %q", unreachableUpstream, s.Name, s.R.File.Path)
		n.sendServiceEvent(s, UnreachableUpstream)
		return
	}

	ngf := &models.NginxConfig{
		Type:         fileType,
		Path:         filepath.Join(configDirectory, config.Unique+".conf"),
		LastModified: s.LastModified,
	}

	// Start transaction
	tx, err := n.DB.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("could not begin transaction: %w", err)
		n.Monitor.CaptureException(err, nil)
		return
	}
	defer func() {
		if err == nil {
			if commitErr := tx.Commit(); commitErr != nil {
				n.Monitor.CaptureException(fmt.Errorf("could not commit transaction: %w", commitErr), nil)
				n.deleteFile(ngf.Path)
				return
			}
			log.Printf("CONFIGURED BASE FOR: %s", s.Name)
		} else {
			if rollBkErr := tx.Rollback(); rollBkErr != nil {
				n.Monitor.CaptureException(fmt.Errorf("could not rollback transaction: %w", rollBkErr), nil)
				n.deleteFile(ngf.Path)
				return
			}
		}
	}()

	err = s.AddNginxConfigs(ctx, tx, true, ngf)
	if err != nil {
		err = fmt.Errorf("could not add nginx config to service in DB: %w", err)
		n.Monitor.CaptureException(err, nil)
		return
	}

	s.State = internal.StateConfigured
	if strings.ToLower(config.Type) == "http" && config.Ssl {
		s.State = internal.StateToConfigureHttps
	}

	_, err = s.Update(ctx, tx, boil.Infer())
	if err != nil {
		err = fmt.Errorf("could not update service in DB: %w", err)
		n.Monitor.CaptureException(err, nil)
		return
	}

	// Add the nginx config file and rollback the transaction if there's an error
	err = os.WriteFile(ngf.Path, configContents, 0o644)
	if err != nil {
		err = fmt.Errorf("error writing nginx config file for %q to %q: %w", s.Name, ngf.Path, err)
		n.Monitor.CaptureException(err, nil)
		return
	}
}

func (n NginxGenerator) generateHttpsConfig(ctx context.Context, s *models.Service) {
	var err error
	var b bytes.Buffer

	config, err := n.getFullConfig(s)
	if err != nil {
		err = fmt.Errorf("could not get full config: %w", err)
		n.Monitor.CaptureException(err, nil)
		return
	}

	if config.SslSource != "manual" {
		err = n.setSslCertificatePath(ctx, &config)
		if err != nil {
			err = fmt.Errorf("could set SSL cert paths: %w", err)
			n.sendServiceEvent(s, SSLCertGenerationFail)
			n.Monitor.CaptureException(err, nil)
			return
		}
	}

	configDirectory := filepath.Join(n.Settings.CONFIG_OUTPUT_DIR, "http")
	fileType := "https"

	err = n.Templates.ExecuteTemplate(&b, "https", config)
	if err != nil {
		err = fmt.Errorf("error generating https config for %q in %q: %w", s.Name, s.R.File.Path, err)
		n.Monitor.CaptureException(err, nil)
		return
	}
	configContents := b.Bytes()

	configPath := filepath.Join(configDirectory, config.Unique+".SSL.conf")
	_, err = models.NginxConfigs(models.NginxConfigWhere.Path.EQ(configPath)).DeleteAll(ctx, n.DB)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		err = fmt.Errorf("could not delete old nginx http config at %q: %w", configPath, err)
		n.Monitor.CaptureException(err, nil)
		return
	}

	ngf := &models.NginxConfig{
		Type:         fileType,
		Path:         configPath,
		LastModified: s.LastModified,
	}

	// Start transaction
	tx, err := n.DB.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("could not begin transaction: %w", err)
		n.Monitor.CaptureException(err, nil)
		return
	}
	defer func() {
		if err == nil {
			if commitErr := tx.Commit(); commitErr != nil {
				n.Monitor.CaptureException(fmt.Errorf("could not commit transaction: %w", commitErr), nil)
				n.deleteFile(ngf.Path)
				return
			}
			log.Printf("CONFIGURED HTTPS FOR: %s", s.Name)
		} else {
			if rollBkErr := tx.Rollback(); rollBkErr != nil {
				n.Monitor.CaptureException(fmt.Errorf("could not rollback transaction: %w", rollBkErr), nil)
				n.deleteFile(ngf.Path)
				return
			}
		}
	}()

	err = s.AddNginxConfigs(ctx, tx, true, ngf)
	if err != nil {
		err = fmt.Errorf("could not add nginx config to service in DB: %w", err)
		n.Monitor.CaptureException(err, nil)
		return
	}

	if s.State != internal.StateConfigured {
		// If the https regenration was triggered by the validity, don't change the state
		// E.g. if a https generation was triggered by the config.Validity, then it will already
		// have state as Configured.
		// If we change the state, we needlessly do the httpToHttps redirect generation
		s.State = internal.StateConfigured
		if config.HttpsOnly {
			s.State = internal.StateToDisableHttp
		}
	}

	s.HTTPSConfigured = null.TimeFrom(time.Now())

	_, err = s.Update(ctx, tx, boil.Infer())
	if err != nil {
		err = fmt.Errorf("could not update service in DB: %w", err)
		n.Monitor.CaptureException(err, nil)
		return
	}

	err = os.WriteFile(ngf.Path, configContents, 0o644)
	if err != nil {
		err = fmt.Errorf("error writing nginx config file for %q to %q: %w", s.Name, ngf.Path, err)
		n.Monitor.CaptureException(err, nil)
		return
	}
}

func (n NginxGenerator) generateNoHttpConfig(ctx context.Context, s *models.Service, wg *sync.WaitGroup) {
	defer wg.Done()

	var err error
	var b bytes.Buffer

	config, err := n.getFullConfig(s)
	if err != nil {
		err = fmt.Errorf("could not get full config: %w", err)
		n.Monitor.CaptureException(err, nil)
		return
	}

	ngf, err := s.NginxConfigs(models.NginxConfigWhere.Type.EQ("http")).One(ctx, n.DB)
	if err != nil {
		err = fmt.Errorf("could not get base http config for service %q: %w", s.Name, err)
		n.Monitor.CaptureException(err, nil)
		return
	}

	err = n.Templates.ExecuteTemplate(&b, "httptoHttps", config)
	if err != nil {
		err = fmt.Errorf("error generating https only config for %q in %q: %w", s.Name, s.R.File.Path, err)
		n.Monitor.CaptureException(err, nil)
		return
	}
	configContents := b.Bytes()

	// We are not creating a transaction since we're only running one db query
	// We will write the file first, and delete it if there's an error while updating the service
	err = os.WriteFile(ngf.Path, configContents, 0o644)
	if err != nil {
		err = fmt.Errorf("error writing nginx config file for %q to %q: %w", s.Name, ngf.Path, err)
		n.Monitor.CaptureException(err, nil)
		return
	}

	s.State = internal.StateConfigured
	_, err = s.Update(ctx, n.DB, boil.Infer())
	if err != nil {
		n.deleteFile(ngf.Path)
		err = fmt.Errorf("could not update service in DB: %w", err)
		n.Monitor.CaptureException(err, nil)
		return
	}

	log.Printf("CONFIGURED HTTPS ONLY FOR: %s", s.Name)
}

func (n NginxGenerator) pingUpstreams(config internal.Config) (bool, string) {
	for _, u := range config.Upstream {
		host := strings.Split(u.Address, ":")[0]

		log.Printf("PINGING %q", host)

		cmd := exec.Command("ping", "-c", "1", host)
		err := cmd.Run()
		if err != nil {
			return false, host
		}
	}

	return true, ""
}

func (n NginxGenerator) setSslCertificatePath(ctx context.Context, config *internal.Config) error {
	switch config.SslSource {
	case "manual":
		return nil

	case "letsencrypt":
		CertPath, KeyPath, err := letsencrypt.GetCertificate(ctx, n.Settings, *config)
		config.CertPath = CertPath
		config.KeyPath = KeyPath
		return err

	default:
		return fmt.Errorf("Unknown SSL source %q", config.SslSource)
	}
}

func (n NginxGenerator) getFullConfig(s *models.Service) (internal.Config, error) {
	var config internal.Config
	service := s.Content

	if service.Type == "" {
		service.Type = "http"
	}

	if service.Location == "" && len(service.Locations) == 0 {
		service.Location = "/"
	}

	config = internal.Config{
		Service: service,
		Unique:  s.Name + "-" + s.R.File.Name + "-" + strconv.FormatInt(s.ID, 10),
	}

	return config, nil
}

func (n NginxGenerator) reloadNginx() error {
	log.Println("Reloading NGINX")

	if n.Settings.TESTING {
		return nil
	}

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

func (n NginxGenerator) deleteFile(path string) {
	// Cleanup nginx config file
	err := os.RemoveAll(path)
	if err != nil {
		err = fmt.Errorf("could not cleanup nginx conf file after failed query: %w", err)
		n.Monitor.CaptureException(err, nil)
	}
}

type serviceEvent struct {
	Code int    `json:"code"`
	Msg  string `json:"message"`
}

var (
	UnreachableUpstream = serviceEvent{
		Code: 404,
		Msg:  "could not reach upstream",
	}
	SSLCertGenerationFail = serviceEvent{
		Code: 500,
		Msg:  "ssl certificate generation failed",
	}
)

func (n NginxGenerator) sendServiceEvent(service *models.Service, event serviceEvent) {
	if service.Content.Webhook == "" {
		return
	}
	var b bytes.Buffer
	err := toml.NewEncoder(&b).Encode(service.Content)
	if err != nil {
		err = fmt.Errorf("could not encode service config: %w", err)
		n.Monitor.CaptureException(err, nil)
	}

	values := url.Values{}
	values.Set("id", service.Name)
	values.Set("data", b.String())
	values.Set("event_code", strconv.Itoa(event.Code))
	values.Set("event_msg", event.Msg)

	client := http.Client{Timeout: 5 * time.Second}
	_, err = client.PostForm(service.Content.Webhook, values)
	if err != nil {
		err = fmt.Errorf("error sending event to webhook: %w", err)
		n.Monitor.CaptureException(err, nil)
	}
}
