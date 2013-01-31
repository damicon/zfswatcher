zfswatcher
==========

This is zfswatcher, ZFS pool monitoring and notification daemon.

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

Download:

http://zfswatcher.damicon.fi/

Source repository:

https://github.com/damicon/zfswatcher/


Prerequisites
-------------

This software is implemented in Go programming language. Further
information about installing the Go environment is available
at the following URL:

http://golang.org/doc/install

The version 1.0.3 of the Go programming language on 64 bit platform
is recommended.

The software has been developed on Debian 6.0 (squeeze) and Ubuntu 12.04
(precise) but it should work fine on other Linux distributions.

Minor modifications are currently required if running on other platforms.


Compiling
---------

Optionally edit the Makefile to set the installation directories.
Then run:

    make


Installation
------------

If you are upgrading an existing installation you should probably just
copy the zfswatcher binary manually. If this is a fresh installation or
you want to replace all customizations with defaults, you can run:

    make install


Configuration
-------------

Edit the configuration file:

    editor /etc/zfs/zfswatcher.conf


Support
-------


Contributions
-------------

Contributions and suggestions are welcome.


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

