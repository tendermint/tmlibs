.PHONY: all test get_vendor_deps ensure_tools

GOTOOLS = \
	github.com/Masterminds/glide \
	github.com/alecthomas/gometalinter

all: test

test:
	go test `glide novendor`

test_100:
	for i in {0..100}; do make test; done

test_coverage:
	./test.sh

get_vendor_deps: ensure_tools
	@rm -rf vendor/
	@echo "--> Running glide install"
	@glide install

ensure_tools:
	go get $(GOTOOLS)

metalinter: ensure_tools
	@gometalinter --install
	gometalinter --vendor --deadline=600s --enable-all --disable=lll ./...

metalinter_test: ensure_tools
	@gometalinter --install
	gometalinter --vendor --deadline=600s --disable-all  \
		--enable=deadcode \
		--enable=gas \
		--enable=goconst \
		--enable=gosimple \
	 	--enable=ineffassign \
	   	--enable=interfacer \
		--enable=megacheck \
	 	--enable=misspell \
	   	--enable=staticcheck \
		--enable=safesql \
	   	--enable=structcheck \
	   	--enable=unconvert \
		--enable=unused \
	   	--enable=varcheck \
		--enable=vetshadow \
		--enable=vet \
		./...