zfswatcher
==========

ZFS pool monitoring and notification daemon.

Features
--------

- Periodically gets the zpool status output and parses it.

- Sends configurable notifications on status changes.

- Supports the following notification destinations with configurable
  severity levels:
  * file
  * syslog
  * e-mail

- Controls the disk enclosure LEDs (currently using external ledctl
  utility).

- Web interface for viewing status and displaying logs.

- Disk "locate" LEDs can be controlled through web interface.


Supported operating systems/distributions
-----------------------------------------

Linux on x86_64/amd64 platform. In particular the following distributions:

- Debian 6.0 (squeeze)
- Ubuntu 12.04 (precise)
- Ubuntu 12.10 (quantal)
- RHEL/CentOS/Scientific Linux 6.X

Obtaining the software
----------------------

Downloads and other information:

http://zfswatcher.damicon.fi/

Source repository:

https://github.com/damicon/zfswatcher/


Installing and upgrading on Debian/Ubuntu
-----------------------------------------

Download the .deb package and install it with `dpkg`, for example:

    dpkg -i zfswatcher_0.01-1_amd64.deb

The service is started by default according to the Debian and Ubuntu
conventions.


Installing and upgrading on RHEL/CentOS/Scientific Linux
--------------------------------------------------------

Download the .rpm package and install it with `yum`, for example:

    yum install zfswatcher-0.01-1.x86_64.rpm

The service is **not** started by default according to Red Hat
conventions. It can be started as follows:

    service zfswatcher start


Installing from source
----------------------

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
(precise) but it should work fine on other Linux distributions.

Minor modifications are currently required if running on other platforms.


### Compiling

Optionally edit the Makefile to set the installation directories.
Then run:

    make


### Installation

    make install


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
  from `/usr/local/share/zfswatcher/www` to another location and change the
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

