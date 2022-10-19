.PHONY: build open

build:
	go run gen.go

open:
	xdg-open index.html
