Name:		zfswatcher
Version:	%{version}
Release:	1%{?dist}
Summary:	ZFS pool monitoring and notification daemon

Group:		Applications/System
License:	GPLv3+
Vendor:		Damicon Kraa Oy <http://www.damicon.fi/>
Packager:	Janne Snabb <snabb@epipe.com>
URL:		http://zfswatcher.damicon.fi/
Source0:	%{name}-%{version}.tar.gz
ExclusiveArch:	x86_64
BuildRoot:	%{_tmppath}/%{name}-%{version}-%{release}-root-%(%{__id_u} -n)

#BuildRequires:	# Go 1.0.3
Requires:		zfs
Requires(post):		chkconfig
Requires(preun):	chkconfig
Requires(preun):	initscripts
Requires(postun):	initscripts

%description
Zfswatcher is ZFS pool monitoring and notification daemon
with the following main features:
 * Periodically inspects the zpool status.
 * Sends configurable notifications on status changes.
 * Controls the disk enclosure LEDs.
 * Web interface for displaying status and logs.

%prep
%setup -q


%build
make


%install
rm -rf $RPM_BUILD_ROOT
make install DESTDIR=$RPM_BUILD_ROOT

%__mkdir_p ${RPM_BUILD_ROOT}%{_initddir}
%__install -p -m 755 etc/redhat-startup.sh ${RPM_BUILD_ROOT}%{_initddir}/%{name}

%__mkdir_p -m 755 ${RPM_BUILD_ROOT}%{_sysconfdir}/logrotate.d
%__install -p -m 644 etc/logrotate.conf ${RPM_BUILD_ROOT}%{_sysconfdir}/logrotate.d/%{name}


%clean
rm -rf $RPM_BUILD_ROOT


%files
%defattr(-,root,root,-)
%doc README.md COPYING NEWS
%{_sbindir}/*
%{_mandir}/man8/*
%{_datadir}/%{name}/
%{_initddir}/%{name}
%config(noreplace) %{_sysconfdir}/zfs/*.conf
%config(noreplace) %{_sysconfdir}/logrotate.d/*


%post
# This adds the proper /etc/rc*.d links for the script
/sbin/chkconfig --add zfswatcher


%preun
if [ $1 -eq 0 ] ; then
	/sbin/service zfswatcher stop >/dev/null 2>&1
	/sbin/chkconfig --del zfswatcher
fi


%postun
if [ "$1" -ge "1" ] ; then
	/sbin/service zfswatcher condrestart >/dev/null 2>&1 || :
fi


