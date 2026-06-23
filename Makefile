.PHONY: build test screenshots clean setup

setup:
	go mod tidy

build:
	go build ./...

test:
	go test ./...

screenshots: screenshots/dropdown.gif screenshots/styled.gif

screenshots/dropdown.gif: vhs/dropdown.tape
	vhs vhs/dropdown.tape

screenshots/styled.gif: vhs/styled.tape
	vhs vhs/styled.tape

clean:
	rm -f screenshots/*.gif
