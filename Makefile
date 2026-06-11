.PHONY: build test lint clean install

build:
	go build -o dev-box .

test:
	go test ./... -v -cover

lint:
	go vet ./...

clean:
	rm -f dev-box dev-box.exe

install:
	go install .
