#
# Makefile - ZFS pool monitoring and notification daemon
#
# Copyright © 2012-2013 Damicon Kraa Oy
#
# This file is part of zfswatcher.
#
# Zfswatcher is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# Zfswatcher is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with zfswatcher. If not, see <http://www.gnu.org/licenses/>.
#

SHELL = /bin/sh

# Go tool:
GO=go

# Go library path:
GOPATH=`pwd`/golibs

# Rules:
all: zfswatcher

zfswatcher: zfswatcher.go leds.go util.go webserver.go
	GOPATH=$(GOPATH) $(GO) build -o $@

clean: 
	GOPATH=$(GOPATH) $(GO) clean
	rm -f zfswatcher

install: zfswatcher
	install -d $(DESTDIR)/usr/sbin $(DESTDIR)/etc/zfs \
		$(DESTDIR)/usr/share/zfswatcher
	install -c zfswatcher $(DESTDIR)/usr/sbin/zfswatcher
	install -c -m 644 etc/zfswatcher.conf $(DESTDIR)/etc/zfs/zfswatcher.conf
	cp -R www $(DESTDIR)/usr/share/zfswatcher/www

deb:
	dpkg-buildpackage -b -uc -tc

# eof
