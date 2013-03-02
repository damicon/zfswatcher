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

# Shell:
SHELL	= /bin/sh

# Go tool and library path:
GO	= /usr/local/go/bin/go
GOPATH	= `pwd`/golibs

# Installation directories:
prefix		= /usr
exec_prefix	= $(prefix)
bindir		= $(exec_prefix)/bin
sbindir		= $(exec_prefix)/sbin
datarootdir	= $(prefix)/share
datadir		= $(datarootdir)
sysconfdir	= /etc
docdir		= $(datarootdir)/doc/zfswatcher
mandir		= $(datarootdir)/man
man1dir		= $(mandir)/man1
man5dir		= $(mandir)/man5
man8dir		= $(mandir)/man8

VERSION		= `fgrep VERSION version.go | cut -d\" -f2`

# Rules:
all: zfswatcher

zfswatcher: zfswatcher.go leds.go setup.go util.go version.go webserver.go \
	webpagehandlers.go zparse.go \
	osutil_linux.go osutil_freebsd.go osutil_solaris.go
	GOPATH=$(GOPATH) $(GO) build -o $@

clean: 
	GOPATH=$(GOPATH) $(GO) clean
	version=$(VERSION) &&						\
	rm -f zfswatcher						\
		zfswatcher-$${version}.tar.gz				\
		zfswatcher-$${version}-*.rpm				\
		zfswatcher_$${version}-*.deb				\
		zfswatcher_$${version}-*.changes

install: zfswatcher
	install -d $(DESTDIR)$(sbindir) $(DESTDIR)$(sysconfdir)/zfs	\
		$(DESTDIR)$(datadir)/zfswatcher				\
		$(DESTDIR)$(man8dir)
	install -c zfswatcher $(DESTDIR)$(sbindir)/zfswatcher
	test -e $(DESTDIR)$(sysconfdir)/zfs/zfswatcher.conf		\
		|| install -c -m 644 etc/zfswatcher.conf		\
			$(DESTDIR)$(sysconfdir)/zfs/zfswatcher.conf
	install -c -m 644 doc/zfswatcher.8				\
		$(DESTDIR)$(man8dir)/zfswatcher.8
	cp -R www $(DESTDIR)$(datadir)/zfswatcher/

# Make tarball:
dist:
	version=$(VERSION) &&						\
	git archive --prefix=zfswatcher-$${version}/			\
		-o zfswatcher-$${version}.tar.gz $${version}

# Make a new Debian package version:
newdebversion:
	version=$(VERSION) &&						\
	dch --newversion $${version}-1					\
		--upstream						\
		--distribution unstable 				\
		"New version $${version}"

# Make Debian package:
deb:
	version=$(VERSION) &&						\
	dpkg-buildpackage -b -uc -tc &&					\
	mv ../zfswatcher_$${version}-*.deb 				\
		../zfswatcher_$${version}-*.changes 			\
		.

# Make RPM package:
rpm:	dist
	version=$(VERSION) &&						\
	rpmbuild=`mktemp -d "/tmp/zfswatcher-rpmbuild-XXXXXXXX"`;	\
	mkdir -p $$rpmbuild/TMP &&					\
	mkdir -p $$rpmbuild/BUILD &&					\
	mkdir -p $$rpmbuild/RPMS &&					\
	mkdir -p $$rpmbuild/SRPMS &&					\
	mkdir -p $$rpmbuild/SPECS &&					\
	cp zfswatcher.spec $$rpmbuild/SPECS/ &&				\
	mkdir -p $$rpmbuild/SOURCES &&					\
	cp zfswatcher-$${version}.tar.gz $$rpmbuild/SOURCES/ &&		\
	rpmbuild -ba							\
		--define "_topdir $$rpmbuild"				\
		--define "_tmppath $$rpmbuild/TMP"			\
		--define "version $${version}"				\
		zfswatcher.spec &&					\
	cp $$rpmbuild/RPMS/*/* . &&					\
	cp $$rpmbuild/SRPMS/* . &&					\
	rm -r $$rpmbuild &&						\
	echo "RPM build finished"

# eof
