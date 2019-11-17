.PHONY: clean

ifs: *.go
	go build -o istio-federation-server

docker: Dockerfile ifs
	docker build -t istio/ifs .

clean:
	rm -f istio-federation-server
