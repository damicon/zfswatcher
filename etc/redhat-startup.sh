#!/bin/sh
#
# zfswatcher - ZFS monitoring daemon
#
# chkconfig:   2345 80 20
# description: ZFS monitoring daemon

# http://fedoraproject.org/wiki/FCNewInit/Initscripts
### BEGIN INIT INFO
# Provides:		zfswatcher
# Required-Start:	zfs $syslog $network $remote_fs
# Required-Stop:	zfs $syslog $network $remote_fs
# Default-Start:	2 3 4 5
# Default-Stop:		0 1 6
# Short-Description:	ZFS monitoring daemon
# Description:		ZFS monitoring daemon
### END INIT INFO

# Source function library.
. /etc/rc.d/init.d/functions

exec="/usr/sbin/zfswatcher"
prog=$(basename $exec)

[ -e /etc/sysconfig/$prog ] && . /etc/sysconfig/$prog

lockfile=/var/lock/subsys/$prog

start() {
    echo -n $"Starting $prog: "
    daemon "$exec &"
    retval=$?
    echo
    [ $retval -eq 0 ] && touch $lockfile
    return $retval
}

stop() {
    echo -n $"Stopping $prog: "
    killproc $prog
    retval=$?
    echo
    [ $retval -eq 0 ] && rm -f $lockfile
    return $retval
}

restart() {
    stop
    start
}

case "$1" in
    start|stop|restart)
        $1
        ;;
    force-reload)
        restart
        ;;
    status)
        status $prog
        ;;
    try-restart|condrestart)
        if status $prog >/dev/null ; then
            restart
        fi
	;;
    reload)
        status $prog >/dev/null || exit 7
        killproc $prog -HUP
        ;;
    *)
        echo $"Usage: $0 {start|stop|status|restart|try-restart|reload|force-reload}"
        exit 2
esac
