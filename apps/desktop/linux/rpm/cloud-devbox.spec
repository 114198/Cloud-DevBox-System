# Cloud DevBox RPM Spec File

Name:           cloud-devbox
Version:        %{version}
Release:        1%{?dist}
Summary:        Cloud Development Environment Manager

License:        MIT
URL:            https://devbox.io
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  webkit2gtk4.1-devel
BuildRequires:  gtk3-devel
BuildRequires:  openssl-devel

Requires:       webkit2gtk4.1
Requires:       gtk3
Requires:       libappindicator-gtk3
Requires:       openssl
Requires:       ca-certificates

Recommends:     git
Recommends:     openssh-clients

%description
Cloud DevBox is a powerful desktop client for managing cloud
development environments. Create, configure, and connect to
cloud-based development environments directly from your desktop.

Features:
- Quick environment creation from 30+ pre-configured templates
- Seamless IDE integration (VSCode, JetBrains)
- Real-time collaboration with team members
- Built-in terminal and file management
- Resource monitoring and management
- Git integration and version control

%prep
%setup -q

%install
rm -rf %{buildroot}

# Install binary
install -D -m 755 cloud-devbox %{buildroot}%{_bindir}/cloud-devbox

# Install desktop file
install -D -m 644 cloud-devbox.desktop %{buildroot}%{_datadir}/applications/cloud-devbox.desktop

# Install metainfo
install -D -m 644 cloud-devbox.metainfo.xml %{buildroot}%{_datadir}/metainfo/cloud-devbox.metainfo.xml

# Install icons
for size in 16 32 48 64 128 256 512; do
    install -D -m 644 icons/${size}x${size}.png \
        %{buildroot}%{_datadir}/icons/hicolor/${size}x${size}/apps/cloud-devbox.png
done

# Install MIME type
install -D -m 644 cloud-devbox.xml %{buildroot}%{_datadir}/mime/packages/cloud-devbox.xml

%post
# Update desktop database
update-desktop-database %{_datadir}/applications &> /dev/null || :

# Update icon cache
touch --no-create %{_datadir}/icons/hicolor &> /dev/null || :
gtk-update-icon-cache %{_datadir}/icons/hicolor &> /dev/null || :

# Update MIME database
update-mime-database %{_datadir}/mime &> /dev/null || :

%postun
# Update desktop database
update-desktop-database %{_datadir}/applications &> /dev/null || :

# Update icon cache
touch --no-create %{_datadir}/icons/hicolor &> /dev/null || :
gtk-update-icon-cache %{_datadir}/icons/hicolor &> /dev/null || :

# Update MIME database
update-mime-database %{_datadir}/mime &> /dev/null || :

%files
%license LICENSE
%doc README.md
%{_bindir}/cloud-devbox
%{_datadir}/applications/cloud-devbox.desktop
%{_datadir}/metainfo/cloud-devbox.metainfo.xml
%{_datadir}/icons/hicolor/*/apps/cloud-devbox.png
%{_datadir}/mime/packages/cloud-devbox.xml

%changelog
* Mon Jan 13 2026 Cloud DevBox Team <support@devbox.io> - 0.1.0-1
- Initial release
