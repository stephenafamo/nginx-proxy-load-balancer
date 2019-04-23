package cmd

import (
	"os/exec"
	"log"
	"fmt"
)

func getLetsEncryptCertificate(config *ConfigTemplateStruct) (string, string, error) {
	webrootPath := fmt.Sprintf("/docker/challenge/%s", config.Unique)

	cmd := exec.Command("mkdir", "-p", webrootPath)
	err := cmd.Run()
	if err != nil {
		return "", "", fmt.Errorf("Can't make letsencrypt webroot dir: %s", err)
	}

	cmd = exec.Command(
		"letsencrypt", 
		"certonly",
		"--agree-tos",
		"--email",
		settings.Email,
		"-q",
		"--cert-name",
		config.Domains[0],
		"-a",
		"webroot",
		"--webroot-path",
		webrootPath,
	)

	for _, domain := range config.Domains {
		cmd.Args = append(cmd.Args, "-d")
		cmd.Args = append(cmd.Args, domain)
	}

	log.Printf("Asking for certificate for: %q\n", config.Unique)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf(
			"Can't get certificate from letsencrypt: %s: %s",
			err,
			output,
		)
	}

	// default for letsencrypt
	certPath := fmt.Sprintf("/etc/letsencrypt/live/%s/fullchain.pem", config.Domains[0]) 
	keyPath := fmt.Sprintf("/etc/letsencrypt/live/%s/privkey.pem", config.Domains[0]) 

	return certPath, keyPath, nil
}