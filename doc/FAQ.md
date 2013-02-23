zfswatcher frequently asked questions
=====================================

1. Why GPLv3 license?
    * Because we want to keep this project open sourced. This way
      anyone who chooses to build on this software will have to release
      their improvements as open source.

2. Why parsing ZFS command output instead of linking to ZFS libraries?
    * Because ZFS on Linux developers do not recommend linking to the ZFS
      libraries and also because of license incompatibility. See the
      following discussion thread:

    https://groups.google.com/a/zfsonlinux.org/d/topic/zfs-devel/AiEi96Kde-k/discussion

