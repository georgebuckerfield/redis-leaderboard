NAME = main

run: build-linux
	docker-compose up --build

build:
	go build -o bin/$(NAME)

build-linux:
	env GOOS=linux go build -o bin/$(NAME)
