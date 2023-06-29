package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bobesa/go-domain-util/domainutil"
	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
	"github.com/stephenafamo/kronika"
	"github.com/vultr/govultr/v3"
	"golang.org/x/oauth2"
)

type Config struct {
	VULTR_API_KEY      string `env:"VULTR_API_KEY,required"`
	CERTBOT_DOMAIN     string `env:"CERTBOT_DOMAIN,required"`
	CERTBOT_VALIDATION string `env:"CERTBOT_VALIDATION,required"`

	LETSENCRYPT_DNS_PROPAGATION int `env:"LETSENCRYPT_DNS_PROPAGATION,default=120"`
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	err := godotenv.Overload(".env")
	if err != nil {
		// Ignore error if file is not present
		if !errors.Is(err, os.ErrNotExist) {
			panic(err)
		}
	}

	config := Config{}
	if err := envconfig.Process(ctx, &config); err != nil {
		panic(fmt.Errorf("error parsing config: %w", err))
	}

	log.Printf("Creating DNS record for %q", config.CERTBOT_DOMAIN)

	oauth2Config := &oauth2.Config{}
	ts := oauth2Config.TokenSource(ctx, &oauth2.Token{AccessToken: config.VULTR_API_KEY})
	vultrClient := govultr.NewClient(oauth2.NewClient(ctx, ts))

	rootDomain := domainutil.Domain(config.CERTBOT_DOMAIN)
	recordName := "_acme-challenge"
	if domainutil.HasSubdomain(config.CERTBOT_DOMAIN) {
		recordName += "." + domainutil.Subdomain(config.CERTBOT_DOMAIN)
	}

	_, _, err = vultrClient.DomainRecord.Create(
		ctx,
		rootDomain,
		&govultr.DomainRecordReq{
			Type: "TXT",
			Name: recordName,
			Data: fmt.Sprintf("%q", config.CERTBOT_VALIDATION),
			TTL:  config.LETSENCRYPT_DNS_PROPAGATION,
		},
	)
	if err != nil {
		err = fmt.Errorf("could not create vultr DNS record: %w", err)
		panic(err)
	}

	wait := time.Second * time.Duration(config.LETSENCRYPT_DNS_PROPAGATION)
	log.Printf("Waiting for %f seconds", wait.Seconds())
	kronika.WaitFor(ctx, wait)
}
