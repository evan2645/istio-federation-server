.PHONY: docker clean

ifs: *.go
	go build -o istio-federation-server

docker: docker/istio-federation-server docker/
	docker build -t istio/ifs .

clean:
	rm istio-federation-server
