.PHONY: build test vet install clean zip

build:
	go build -buildvcs=false -o ai ./cmd/ai

test:
	go test ./...

vet:
	go vet ./...

install:
	./install.sh

clean:
	rm -f ai

zip:
	cd .. && zip -r ai-manager.zip ai-manager -x 'ai-manager/ai'
