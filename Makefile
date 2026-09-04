BINARY      := thugsflooder
MODULE      := github.com/thugsred/thugsflooder
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo 0.1.0)
DIST        := dist
LDFLAGS     := -s -w -X $(MODULE)/internal/about.Version=$(VERSION)

# GOOS/GOARCH pair -> Debian architecture name.
# linux/386   -> i386   (32-bit Intel)
# linux/amd64 -> amd64  (64-bit Intel)
# linux/arm   -> armhf  (32-bit ARM, GOARM=7)
# linux/arm64 -> arm64  (64-bit ARM)
PLATFORMS := linux/386/i386 linux/amd64/amd64 linux/arm/armhf linux/arm64/arm64

.PHONY: all build deb clean test vet

all: build deb

test:
	go test ./...

vet:
	go vet ./...

build: vet
	@for p in $(PLATFORMS); do \
		goos=$$(echo $$p | cut -d/ -f1); \
		goarch=$$(echo $$p | cut -d/ -f2); \
		debarch=$$(echo $$p | cut -d/ -f3); \
		outdir=$(DIST)/$$debarch; \
		mkdir -p $$outdir; \
		echo "==> building $$goos/$$goarch ($$debarch) -> $$outdir/$(BINARY)"; \
		goarm=""; \
		if [ "$$goarch" = "arm" ]; then goarm="GOARM=7"; fi; \
		env CGO_ENABLED=0 GOOS=$$goos GOARCH=$$goarch $$goarm \
			go build -trimpath -ldflags "$(LDFLAGS)" -o $$outdir/$(BINARY) ./cmd/$(BINARY); \
	done

deb: build
	@for p in $(PLATFORMS); do \
		debarch=$$(echo $$p | cut -d/ -f3); \
		stage=$(DIST)/deb/$$debarch; \
		rm -rf $$stage; \
		mkdir -p $$stage/DEBIAN $$stage/usr/bin $$stage/usr/share/doc/$(BINARY); \
		sed -e "s/{{VERSION}}/$(VERSION)/" -e "s/{{ARCH}}/$$debarch/" \
			packaging/debian/control.tmpl > $$stage/DEBIAN/control; \
		cp packaging/debian/postinst $$stage/DEBIAN/postinst; \
		chmod 755 $$stage/DEBIAN/postinst; \
		cp $(DIST)/$$debarch/$(BINARY) $$stage/usr/bin/$(BINARY); \
		chmod 755 $$stage/usr/bin/$(BINARY); \
		cp LICENSE README.md $$stage/usr/share/doc/$(BINARY)/; \
		echo "==> packaging $$debarch"; \
		dpkg-deb --build --root-owner-group -Zxz $$stage $(DIST)/$(BINARY)_$(VERSION)_$$debarch.deb; \
	done

clean:
	rm -rf $(DIST)
