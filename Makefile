.PHONY: build new open

build:
	go run gen.go

new:
	go run gen.go new

open:
	xdg-open index.html
