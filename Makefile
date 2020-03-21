all:
	sass --embed-sources assets:static
	CGO_ENABLED=0 go build cmd/server/main.go