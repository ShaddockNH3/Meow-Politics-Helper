.PHONY: build web test dev clean

web:
	cd web && npm install && npm run build

build: web
	go build -trimpath -o Meow-Politics-Helper .

test:
	go test ./...

dev:
	cd web && npm run dev

clean:
	rm -rf web/dist web/node_modules
