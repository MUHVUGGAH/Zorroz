build:
	podman build -t camoufox-benchmark .

run:
	podman run camoufox-benchmark --mode headless --browser firefox
	podman run camoufox-benchmark --mode headless --browser camoufox
	podman run camoufox-benchmark --mode headless --browser camoufox-ubo

# Go benchmark targets
build-go:
	go build -o benchmark .

run-go:
	go run . --mode headless --browser firefox
	go run . --mode headless --browser camoufox
	go run . --mode headless --browser camoufox-ubo

build-go-docker:
	podman build -f Dockerfile.gobuild -t camoufox-benchmark-go ../../../

run-go-docker:
	podman run camoufox-benchmark-go --mode headless --browser firefox
	podman run camoufox-benchmark-go --mode headless --browser camoufox
	podman run camoufox-benchmark-go --mode headless --browser camoufox-ubo
