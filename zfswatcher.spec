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
Requires:	zfs

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

%__mkdir_p -m 755 ${RPM_BUILD_ROOT}%{_sysconfdir}/logrotate.d
%__install -p -m 644 etc/logrotate.conf ${RPM_BUILD_ROOT}%{_sysconfdir}/logrotate.d/%{name}


%clean
rm -rf $RPM_BUILD_ROOT


%files
%defattr(-,root,root,-)
%doc README.md COPYING
%{_sbindir}/*
%{_mandir}/man8/*
%{_datadir}/%{name}/
%config(noreplace) %{_sysconfdir}/zfs/*.conf
%config(noreplace) %{_sysconfdir}/logrotate.d/*
#%config %{_sysconfdir}/init/zfswatcher.conf

%changelog
* Fri Feb  2 2013 Janne Snabb <snabb@epipe.com>
- Initial version of RPM package.
