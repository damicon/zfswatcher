zfswatcher
==========

ZFS pool monitoring and notification daemon.

Features
--------

- Periodically gets the zpool status output and parses it.

- Sends notifications based on status changes.

- Supports the following notification destinations with configurable
  severity levels:
  * file
  * syslog
  * e-mail

- Controls the disk enclosure LEDs (currently using external ledctl
  utility).

- Web interface for viewing status and displaying logs.

- Disk "locate" LEDs can be controlled through web interface.


Obtaining the software
----------------------

Downloads and other information:

http://zfswatcher.damicon.fi/

Source repository:

https://github.com/damicon/zfswatcher/


Installing on Debian/Ubuntu
---------------------------

Download the .deb package and install it with `dpkg`, for example:

    dpkg -i zfswatcher_0.01-1_amd64.deb


Installing from source
----------------------

Generally it is best to use ready made packages on Debian/Ubuntu.

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
(precise) but it should work fine on other Linux distributions.

Minor modifications are currently required if running on other platforms.


### Compiling

Optionally edit the Makefile to set the installation directories.
Then run:

    make


### Installation

    make install


### Debian/Ubuntu packaging

The distribution comes with Debian style packaging in `/debian/`
subdirectory. A Debian package can be produced with the following
command:

    make deb

Note that the Go programming language environment needs to be correctly
installed even though it is not listed as a build dependency. This is
because the newest version of the Go language is not available in the
Debian/Ubuntu repositories.

The resulting .deb package file, for example `zfswatcher_0.01-1_amd64.deb`
is placed one directory level up from the source directory.


Configuration
-------------

Edit the configuration file:

    editor /etc/zfs/zfswatcher.conf

If you change the default web templates, it is best to copy them to
another location from `/usr/local/share/zfswatcher/www` and change
the `templatedir` and `resourcedir` settings in the configuration
accordingly. This way your local changes will not be overwritten if you
upgrade the zfswatcher package at a later time.


Support
-------

If you encounter a bug, please open an issue at the GitHub issue
tracker at:

https://github.com/damicon/zfswatcher/issues

Reported bugs are fixed on best effort basis.

Commercial support, custom licensing as well as complete
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

