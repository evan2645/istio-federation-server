.PHONY: ifs docker clean

ifs: *.go
	go build -o docker ./...

docker:
	docker build -t istio/ifs docker

clean:
	rm docker/istio-federation-server
