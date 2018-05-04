default: build

build:
	docker build -t bbklab/build-qrcode:latest -f Dockerfile .
	docker run --name buildqrcode --rm \
		-e CGO_ENABLED=1 \
		-v $(shell pwd):/build \
		-v /go:/go \
		bbklab/build-qrcode:latest \
		sh -c 'go build -ldflags "-linkmode external -extldflags -static" -o qrcode main.go'   # so the binary won't have any external .so dependencies

clean:
	rm -fv qrcode
