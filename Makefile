run:
	go run ./cmd/spark-whatsapp-module
build:
	mkdir -p ./bin
	go build -o ./bin/app ./cmd/spark-whatsapp-module
start:
	./bin/app
setup:
	go run ./cmd/setup
