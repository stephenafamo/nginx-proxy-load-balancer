package cmd

import (
	"os/exec"
	"fmt"
)

func getLetsEncryptCertificate(config *ConfigTemplateStruct) (string, string, error) {
	webrootPath := fmt.Sprintf("/docker/challenge%s", config.Unique)

	cmd := exec.Command("mkdir", "-p", webrootPath)
	err := cmd.Run()
	if err != nil {
		return "", "", fmt.Errorf("Can't make letsencrypt webroot dir: %s", err)
	}

	commandArg := fmt.Sprintf(
		"certonly --agree-tos --email %s -q  --cert-name=%s -a webroot --webroot-path=%s", 
		settings.Email,
		config.Domains[0],
		webrootPath)

	for _, domain := range config.Domains {
		commandArg += " -d " + domain
	}

	fmt.Printf("Asking for certificate:\n%s\n\n", commandArg)

	cmd = exec.Command("letsencrypt", commandArg)
	err = cmd.Run()
	if err != nil {
		return "", "", fmt.Errorf("Can't get certificate from letsencrypt: %s", err)
	}

	// default for letsencrypt
	certPath := fmt.Sprintf("/etc/letsencrypt/live/%s/fullchain.pem", config.Domains[0]) 
	keyPath := fmt.Sprintf("/etc/letsencrypt/live/%s/privkey.pem", config.Domains[0]) 

	return certPath, keyPath, nil
}