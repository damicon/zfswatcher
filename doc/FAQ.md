zfswatcher frequently asked questions
=====================================

### Why GPLv3 license?

Because we want to keep this project open sourced. This way anyone who
chooses to build on this software will have to release their improvements
as open source.

### Why parsing ZFS command output instead of linking to ZFS libraries?

Because ZFS on Linux developers do not recommend linking to the ZFS and
also because of license incompatibility. See the following discussion
thread:

https://groups.google.com/a/zfsonlinux.org/d/topic/zfs-devel/AiEi96Kde-k/discussion

This approach makes it also simple to distribute binary packages which
are not bound to a specific version of ZoL binaries.

