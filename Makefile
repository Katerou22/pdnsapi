build:
	go build -o bin/pdnsapi cmd/app/main.go



run: build
	./bin/pdnsapi