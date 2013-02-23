zfswatcher frequently asked questions
=====================================

### Why GPLv3 license?

Because we want to keep this project and any improvements open
sourced. This way anyone who chooses to build on this software will have
to release their improvements as open source.


### Why parsing ZFS command output instead of linking to ZFS libraries?

This approach makes it simple to distribute binary packages which are not
bound to a specific version of ZoL. If we link to the libraries, we would
have to make binary packages which depend on specific version of ZoL.

Also ZFS on Linux developers do not recommend linking to the ZFS
libraries. See the following discussion thread:

https://groups.google.com/a/zfsonlinux.org/d/topic/zfs-devel/AiEi96Kde-k/discussion

This way we can also ignore any license incompatibility issues.


### Why is the software implemented in Go instead of some other programming language>?

Because Go is cooler.

