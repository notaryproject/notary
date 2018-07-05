test: linter test-standard test-gopherjs

clean: clean-cache
	rm -f serve/files.go

clean-cache:
	rm -rf ${GOPATH}/pkg/*_js

linter: clean
	# ./travis/test.sh linter

test-standard: generate
	./travis/test.sh standard

test-gopherjs: generate clean-cache
	./travis/test.sh gopherjs

generate:
	go generate $$(go list ./... | grep -v /vendor/)
