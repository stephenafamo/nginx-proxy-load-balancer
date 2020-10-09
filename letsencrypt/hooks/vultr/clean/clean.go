package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/bobesa/go-domain-util/domainutil"
	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
	"github.com/vultr/govultr"
)

type Config struct {
	VULTR_API_KEY  string `env:"VULTR_API_KEY,required"`
	CERTBOT_DOMAIN string `env:"CERTBOT_DOMAIN,required"`
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

	log.Printf("deleting DNS record for %q", config.CERTBOT_DOMAIN)

	vultrClient := govultr.NewClient(nil, config.VULTR_API_KEY)

	rootDomain := domainutil.Domain(config.CERTBOT_DOMAIN)
	recordName := "_acme-challenge"
	if domainutil.HasSubdomain(config.CERTBOT_DOMAIN) {
		recordName += "." + domainutil.Subdomain(config.CERTBOT_DOMAIN)
	}

	records, err := vultrClient.DNSRecord.List(ctx, rootDomain)
	if err != nil {
		err = fmt.Errorf("could not list dns records for %q: %w", config.CERTBOT_DOMAIN, err)
		panic(err)
	}

	var recordID int
	for _, record := range records {
		if record.Name == recordName {
			recordID = record.RecordID
			break
		}
	}

	if recordID == 0 {
		// No TXT record with that Name, everything is clean
		log.Printf("No record to delete")
		return
	}
	err = vultrClient.DNSRecord.Delete(ctx, config.CERTBOT_DOMAIN, strconv.Itoa(recordID))
	if err != nil {
		err = fmt.Errorf("could not delete vultr DNS record: %w", err)
		panic(err)
	}

	log.Printf("Deleted the record")
}
