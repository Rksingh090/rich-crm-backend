
run:
	/Users/rishu/go/bin/air

build:
	go build -o bin/api cmd/api/main.go

test:
	go test -v ./...

swagger:
	# Make sure swag is installed: go install github.com/swaggo/swag/cmd/swag@latest
	$$(go env GOPATH)/bin/swag init -g cmd/api/main.go --output docs
