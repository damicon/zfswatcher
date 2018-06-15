zfswatcher
==========

Unmaintained
------------

NOTE: This project is unmaintained. There is a more active fork available
at the following URL: https://github.com/rouben/zfswatcher

***

ZFS pool monitoring and notification daemon.

Please see the project web site for general information about features,
supported operating environments and to download a tarball or packaged
version of this software:

http://zfswatcher.damicon.fi/

Source repository:

https://github.com/damicon/zfswatcher/


Installing and upgrading on Debian/Ubuntu
-----------------------------------------

Download the .deb package and install it with `dpkg`, for example:

    dpkg -i zfswatcher_0.03-1_amd64.deb

The service is started by default according to the Debian and Ubuntu
conventions.


Installing and upgrading on RHEL/CentOS/Scientific Linux
--------------------------------------------------------

Download the .rpm package and install it with `yum`, for example:

    yum install zfswatcher-0.03-1.x86_64.rpm

The service is **not** started by default according to Red Hat
conventions. It can be started as follows:

    service zfswatcher start


Installing from source on Linux and FreeBSD
-------------------------------------------

Generally it is best to use the ready made packages on Debian/Ubuntu
and RHEL/CentOS/Scientific Linux.

If you are packaging this software yourself or you want to compile
from source for other reasons, you can follow these instructions.


### Prerequisites

This software is implemented in Go programming language. Further
information about installing the Go environment is available
at the following URL:

http://golang.org/doc/install

The version 1.0.3 of the Go programming language on a 64 bit platform
is recommended.

The software has been developed on Debian 6.0 (squeeze) and Ubuntu 12.04
(precise) but it should work fine on other Linux distributions and also
recent FreeBSD versions.


### Compiling

Optionally edit the Makefile to set the installation directories.
Then run:

    make


### Installation

    make install


### Running

There are some OS/distribution specific startup scripts in "etc"
subdirectory. They may be useful.

See the installed zfswatcher(8) manual page for information on invoking
the zfswatcher process.


Installing from source on Solaris/OpenSolaris/OpenIndiana
---------------------------------------------------------

The normal "gc" Go toolchain is not available on this plaform.
You need to compile a recent version of
[gccgo](http://golang.org/doc/install/gccgo) from the Subversion
repository at `svn://gcc.gnu.org/svn/gcc/branches/gccgo`. After that
you can utilize the `etc/gccgo-build.sh` shell script.

Good luck!


Configuration
-------------

Edit the configuration file:

    editor /etc/zfs/zfswatcher.conf

Verify the configuration syntax:

    zfswatcher -t

Restart the process:

    service zfswatcher restart

Some notes:

- See the configuration file comments for information about the configuration
  settings.

- Enclosure LED management is disabled by default. Currently an external
  utility `ledctl` (part of ledmon package) is required for this
  functionality.

- Logging to file `/var/log/zfswatcher.log` and local syslog daemon is enabled
  by default.

- E-mail notifications are disabled by default.

- The embedded web server is disabled by default.

- If you change the default web interface templates, it is best to copy them
  from `/usr/share/zfswatcher/www` to another location and change the
  `templatedir` and `resourcedir` settings in the configuration accordingly.
  This way your local changes will not be overwritten if you upgrade the
  package at a later time.


Support
-------

If you encounter a bug, please open an issue at the GitHub issue
tracker at:

https://github.com/damicon/zfswatcher/issues

Reported bugs are fixed on best effort basis.

Commercial support, custom features, custom licensing and complete
storage solutions are available from Damicon Kraa Oy. Contact
<zfswatcher-support@damicon.fi>.


Contributions
-------------

Contributions and suggestions are welcome. Feel free to open an issue
or submit a merge request on GitHub.


Authors
-------

Janne Snabb <snabb@epipe.com>


License
-------

Copyright © 2012-2013 Damicon Kraa Oy

Zfswatcher is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Zfswatcher is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with zfswatcher. If not, see <http://www.gnu.org/licenses/>.

