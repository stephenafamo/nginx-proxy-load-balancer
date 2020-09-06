package letsencrypt

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/stephenafamo/warden/internal"
)

func GetCertificate(ctx context.Context, settings internal.Settings, config internal.Config) (string, string, error) {
	var err error

	outputDir := "/etc/letsencrypt/live"
	certPath := filepath.Join(outputDir, config.Domains[0], "fullchain.pem")
	keyPath := filepath.Join(outputDir, config.Domains[0], "privkey.pem")

	switch {
	case config.LetsEncryptDNSPlugin != "":
		err = getCertificateDNS(ctx, settings, config)
	case config.LetsEncryptAuthenticator != "" && config.LetsEncryptCleaner != "":
		err = getCertificateManual(ctx, settings, config)
	default:
		err = getCertificateAuto(ctx, settings, config, certPath)
	}

	return certPath, keyPath, err
}

func getCertificateDNS(ctx context.Context, settings internal.Settings, config internal.Config) error {
	dnsPlugin := config.LetsEncryptDNSPlugin

	// Check if it is our internal DNS plugin
	switch dnsPlugin {
	case "vultr":
		config.LetsEncryptAuthenticator = "./bin/vultr-auth"
		config.LetsEncryptCleaner = "./bin/vultr-clean"
		return getCertificateManual(ctx, settings, config)
	}

	dnsPluginFlag := fmt.Sprintf("--dns-%s", dnsPlugin)
	dnsCredsFlag := fmt.Sprintf("--dns-%s-credentials", dnsPlugin)
	dnsPropagationFlag := fmt.Sprintf("--dns-%s-propagation-seconds", dnsPlugin)

	var dnsCredFilename string
	switch dnsPlugin {
	case "google":
		dnsCredFilename = fmt.Sprintf("%s.json", dnsPlugin)
	case "route53":
		dnsCredFilename = ""
	default:
		dnsCredFilename = fmt.Sprintf("%s.ini", dnsPlugin)
	}

	cmd := exec.CommandContext(
		ctx,
		"certbot",
		"certonly",
		"--agree-tos",
		"-q", "-n",
		"--email", settings.EMAIL,
		"--preferred-challenges", "dns",
		dnsPluginFlag,
		dnsPropagationFlag, strconv.Itoa(settings.LETSENCRYPT_DNS_PROPAGATION),
		"--cert-name", config.Domains[0],
	)

	// Useful for route53 which does not take any flag
	if dnsCredFilename != "" {
		dnsCredFilepath := filepath.Join(settings.LETSENCRYPT_CREDS_DIR, dnsCredFilename)
		cmd.Args = append(cmd.Args, dnsCredsFlag, dnsCredFilepath)
	}

	if settings.TESTING {
		cmd.Args = append(cmd.Args, "--test-cert")
	}

	for _, domain := range config.Domains {
		cmd.Args = append(cmd.Args, "-d")
		cmd.Args = append(cmd.Args, domain)
	}

	log.Printf("Generating dns certificate for: %q\n", config.Unique)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Can't get certificate from letsencrypt: %w: %s", err, output)
	}

	return nil
}

func getCertificateManual(ctx context.Context, settings internal.Settings, config internal.Config) error {

	cmd := exec.CommandContext(
		ctx,
		"certbot",
		"certonly",
		"--agree-tos",
		"-q", "-n",
		"--email", settings.EMAIL,
		"-a", "manual",
		"--preferred-challenges", "dns",
		"--manual-public-ip-logging-ok",
		"--manual-auth-hook", config.LetsEncryptAuthenticator,
		"--manual-cleanup-hook", config.LetsEncryptCleaner,
		"--cert-name", config.Domains[0],
	)

	if settings.TESTING {
		cmd.Args = append(cmd.Args, "--test-cert")
	}

	for _, domain := range config.Domains {
		cmd.Args = append(cmd.Args, "-d")
		cmd.Args = append(cmd.Args, domain)
	}

	log.Printf("Generating manual certificate for: %q\n", config.Unique)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Can't get certificate from letsencrypt: %w: %s", err, output)
	}

	return nil
}

func getCertificateAuto(ctx context.Context, settings internal.Settings, config internal.Config, path string) error {
	webrootPath := filepath.Join("/docker/challenge", config.Unique)

	err := os.MkdirAll(webrootPath, 0755)
	if err != nil {
		return fmt.Errorf("Can't make letsencrypt webroot dir: %w", err)
	}

	cmd := exec.CommandContext(
		ctx,
		"certbot",
		"certonly",
		"--agree-tos",
		"-q", "-n",
		"--email", settings.EMAIL,
		"-a", "webroot",
		"--webroot-path", webrootPath,
		"--cert-name", config.Domains[0],
	)

	for _, domain := range config.Domains {
		cmd.Args = append(cmd.Args, "-d")
		cmd.Args = append(cmd.Args, domain)
	}

	if settings.TESTING {
		cmd.Args = append(cmd.Args, "--test-cert")
	}

	log.Printf("Generating webroot certificate for: %q\n", config.Unique)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Can't get certificate from letsencrypt: %w: %s", err, output)
	}

	return nil
}
