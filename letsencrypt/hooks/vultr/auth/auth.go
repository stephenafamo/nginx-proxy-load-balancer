package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
	"github.com/stephenafamo/kronika"
	"github.com/vultr/govultr"
)

type Config struct {
	VULTR_API_KEY      string `env:"VULTR_API_KEY,required"`
	CERTBOT_DOMAIN     string `env:"CERTBOT_DOMAIN,required"`
	CERTBOT_VALIDATION string `env:"CERTBOT_VALIDATION,required"`

	LETSENCRYPT_DNS_PROPAGATION int `env:"LETSENCRYPT_DNS_PROPAGATION,default=120"`
}

func main() {
	var ctx = context.Background()
	var err error

	err = godotenv.Overload(".env")
	if err != nil {
		// Ignore error if file is not present
		if !errors.Is(err, os.ErrNotExist) {
			panic(err)
		}
	}

	var config = Config{}
	if err := envconfig.Process(ctx, &config); err != nil {
		panic(fmt.Errorf("error parsing config: %w", err))
	}

	log.Printf("Creating DNS record for %q", config.CERTBOT_DOMAIN)
	vultrClient := govultr.NewClient(nil, config.VULTR_API_KEY)
	err = vultrClient.DNSRecord.Create(
		ctx,
		config.CERTBOT_DOMAIN,
		"TXT",
		"_acme-challenge",
		fmt.Sprintf("%q", config.CERTBOT_VALIDATION),
		config.LETSENCRYPT_DNS_PROPAGATION, 0)
	if err != nil {
		err = fmt.Errorf("could not create vultr DNS record: %w", err)
		panic(err)
	}

	var wait = time.Second * time.Duration(config.LETSENCRYPT_DNS_PROPAGATION)
	log.Printf("Waiting for %f seconds", wait.Seconds())
	kronika.WaitFor(ctx, wait)
}
