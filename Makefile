.PHONY: clean,gh-release

MAC = GOOS=darwin GOARCH=amd64
LINUX = GOOS=linux GOARCH=amd64
SOURCES = $(wildcard pkg/*.go cmd/*.go)
FLAGS = -ldflags "-X main.versionNumber=${VERSION}"
VERSION ?= DEVELOPMENT

RELEASE_TARBALL = release/stscreds-${VERSION}.tar.gz

${RELEASE_TARBALL}: release/mac/stscreds release/linux/stscreds
	mkdir -p release/
	tar -zcf ${RELEASE_TARBALL} -C release/ mac/stscreds linux/stscreds

release/mac/stscreds: ${SOURCES}
	${MAC} go build ${FLAGS} -o release/mac/stscreds cmd/main.go

release/linux/stscreds: ${SOURCES}
	${LINUX} go build ${FLAGS} -o release/linux/stscreds cmd/main.go

gh-release: ${RELEASE_TARBALL}
	gh-release create uswitch/stscreds ${VERSION}

clean:
	rm -rf release/
