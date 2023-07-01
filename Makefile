build:
	CGO_ENABLED=0 GOARCH=amd64 go build -mod vendor -a -o ./bin/warden .
	CGO_ENABLED=0 GOARCH=amd64 go build -mod vendor -a -o ./bin/vultr-auth ./letsencrypt/hooks/vultr/auth
	CGO_ENABLED=0 GOARCH=amd64 go build -mod vendor -a -o ./bin/vultr-clean ./letsencrypt/hooks/vultr/clean
