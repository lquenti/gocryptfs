.phony: build
build:
	./build.bash
	./Documentation/MANPAGE-render.bash

.phony: test
test:
	./test.bash

.phony: root_test
root_test:
	./build.bash
	cd tests/root_test ; go test -c ; sudo ./root_test.test -test.v

.phony: format
format:
	go fmt ./...

.phony: install
install:
	install -Dm755 -t "$(DESTDIR)/usr/bin/" gocryptfs
	install -Dm755 -t "$(DESTDIR)/usr/bin/" gocryptfs-xray/gocryptfs-xray
	install -Dm644 -t "$(DESTDIR)/usr/share/man/man1/" Documentation/gocryptfs.1
	install -Dm644 -t "$(DESTDIR)/usr/share/man/man1/" Documentation/gocryptfs-xray.1
	install -Dm644 -t "$(DESTDIR)/usr/share/licenses/gocryptfs" LICENSE
