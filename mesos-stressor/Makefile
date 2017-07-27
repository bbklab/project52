
IMG="offlineregistry.dataman-inc.com:5000/library/blackicebird-2048"
IMG="busybox"

deault: build

build:
	# env GOOS=darwin go build -o stressor .
	go build -o stressor .

run: build
	./stressor 192.168.1.196:5050 100 10  ${IMG}

clean:
	rm -rfv stressor
