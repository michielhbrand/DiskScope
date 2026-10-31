BIN := bin/diskscope
PREFIX ?= $(HOME)/.local

.PHONY: build run install uninstall test vet fmt clean

build:
	go build -trimpath -o $(BIN) .

run: build
	./$(BIN)

install: build
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 755 $(BIN) $(DESTDIR)$(PREFIX)/bin/diskscope

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/diskscope

test:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

clean:
	rm -rf bin
