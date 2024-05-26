%global goipath github.com/led0nk/ark-clusterinfo

%define debug_package %{nil}

Version: 0.1.0

%gometa

Name: ark-clusterinfo
Release:  1%{?dist}
Summary:  steam observation tool

License:  BSD
Source0: %{name}-%{version}.tar.gz

BuildRequires: golang
BuildRequires: make
BuildRequires: git

%description
ark-clusterinfo is a steam observation tool to track players


%prep
%goprep
%autosetup

%build
go build -v -buildmode pie -mod vendor -o %{gobuilddir}/bin/%{name} cmd/server/main.go

%install
install -m 0755 -vd                     %{buildroot}%{_bindir}
install -m 0755 -vd                     %{buildroot}%{_unitdir}
install -m 0755 -vp %{gobuilddir}/bin/* %{buildroot}%{_bindir}/
install -m 0644 -vp %{name}.service %{buildroot}%{_unitdir}/

%check
%gocheck

%post
%systemd_post %{name}.service

%preun
%systemd_preun %{name}.service

%files
%{_bindir}/%{name}
%{_unitdir}/%{name}.service

%changelog
%autochangelog

