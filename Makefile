build:
	CGO_ENABLED=1 GOARCH=amd64 go build -a -o ./bin/warden .
	CGO_ENABLED=1 GOARCH=amd64 go build -a -o ./bin/vultr-auth ./letsencrypt/hooks/vultr/auth
	CGO_ENABLED=1 GOARCH=amd64 go build -a -o ./bin/vultr-clean ./letsencrypt/hooks/vultr/clean
