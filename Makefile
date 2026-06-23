.PHONY: build test screenshots clean setup

setup:
	go mod tidy

build:
	go build ./...

test:
	go test ./...

screenshots: screenshots/dropdown.gif

screenshots/dropdown.gif: vhs/dropdown.tape
	vhs vhs/dropdown.tape

clean:
	rm -f screenshots/*.gif
