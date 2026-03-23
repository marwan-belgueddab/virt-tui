BINARY_NAME=virt-tui
GO_FILES=$(shell find . -name "*.go")

.PHONY: all build clean install uninstall

all: build

build: $(BINARY_NAME)

$(BINARY_NAME): $(GO_FILES)
	go build -o $(BINARY_NAME) .

clean:
	rm -f $(BINARY_NAME)

install: build
	install -D -m 0755 $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

uninstall:
	rm -f /usr/local/bin/$(BINARY_NAME)
