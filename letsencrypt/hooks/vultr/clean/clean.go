package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bobesa/go-domain-util/domainutil"
	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
	"github.com/vultr/govultr/v3"
	"golang.org/x/oauth2"
)

type Config struct {
	VULTR_API_KEY  string `env:"VULTR_API_KEY,required"`
	CERTBOT_DOMAIN string `env:"CERTBOT_DOMAIN,required"`
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

	log.Printf("deleting DNS record for %q", config.CERTBOT_DOMAIN)

	oauth2Config := &oauth2.Config{}
	ts := oauth2Config.TokenSource(ctx, &oauth2.Token{AccessToken: config.VULTR_API_KEY})
	vultrClient := govultr.NewClient(oauth2.NewClient(ctx, ts))

	rootDomain := domainutil.Domain(config.CERTBOT_DOMAIN)
	recordName := "_acme-challenge"
	if domainutil.HasSubdomain(config.CERTBOT_DOMAIN) {
		recordName += "." + domainutil.Subdomain(config.CERTBOT_DOMAIN)
	}

	var records []govultr.DomainRecord
	var cursor string

	for {
		newRecords, meta, _, err := vultrClient.DomainRecord.List(ctx, rootDomain, &govultr.ListOptions{
			PerPage: 500,
			Cursor:  cursor,
		})
		if err != nil {
			err = fmt.Errorf("could not list dns records for %q: %w", config.CERTBOT_DOMAIN, err)
			panic(err)
		}
		records = append(records, newRecords...)

		if meta == nil || meta.Links == nil || meta.Links.Next == "" {
			break
		}
		cursor = meta.Links.Next
	}

	var recordID string
	for _, record := range records {
		if record.Name == recordName {
			recordID = record.ID
			break
		}
	}

	if recordID == "" {
		// No TXT record with that Name, everything is clean
		log.Printf("No record to delete")
		return
	}
	err = vultrClient.DomainRecord.Delete(ctx, config.CERTBOT_DOMAIN, recordID)
	if err != nil {
		err = fmt.Errorf("could not delete vultr DNS record: %w", err)
		panic(err)
	}

	log.Printf("Deleted the record")
}
