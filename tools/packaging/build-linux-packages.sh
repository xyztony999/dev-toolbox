#!/usr/bin/env bash
# 构建 dev-toolbox 的 deb 与 rpm 包。
# 用法：build-linux-packages.sh <版本(不带v)> <amd64|arm64> <二进制路径> [gui]
#   缺省打包 CLI；第 4 个参数为 "gui" 时打包桌面客户端（依赖 GTK/WebKitGTK 4.1）。
# 产物输出到当前目录：
#   CLI: dev-toolbox_<版本>_<架构>.deb / dev-toolbox-<版本>-1.<rpm架构>.rpm
#   GUI: dev-toolbox-gui_<版本>_<架构>.deb（rpm 暂不打包 GUI）
set -euo pipefail

version="${1:?用法: build-linux-packages.sh <版本> <amd64|arm64> <二进制路径> [gui]}"
deb_arch="${2:?用法: build-linux-packages.sh <版本> <amd64|arm64> <二进制路径> [gui]}"
bin="${3:?用法: build-linux-packages.sh <版本> <amd64|arm64> <二进制路径> [gui]}"
variant="${4:-cli}"

case "$deb_arch" in
  amd64) rpm_arch=x86_64 ;;
  arm64) rpm_arch=aarch64 ;;
  *) echo "不支持的架构：$deb_arch" >&2; exit 2 ;;
esac

repo="$(cd "$(dirname "$0")/../.." && pwd)"
repo="$(realpath "$repo")"
bin="$(realpath "$bin")"

pkg_name="dev-toolbox"
bin_name="dev-toolbox"
deps_line="git"
if [ "$variant" = gui ]; then
  pkg_name="dev-toolbox-gui"
  bin_name="dev-toolbox-gui"
  deps_line="git, libgtk-3-0, libwebkit2gtk-4.1-0"
fi

stage="$(mktemp -d)"
trap 'rm -rf "$stage"' EXIT

install -D -m 0755 "$bin" "$stage/usr/bin/$bin_name"
install -D -m 0644 "$repo/LICENSE" "$stage/usr/share/licenses/$pkg_name/LICENSE"
install -D -m 0644 "$repo/NOTICE" "$stage/usr/share/licenses/$pkg_name/NOTICE"
install -D -m 0644 "$repo/README.md" "$stage/usr/share/doc/$pkg_name/README.md"

# ── deb（Debian/Ubuntu）──
mkdir -p "$stage/DEBIAN"
cat > "$stage/DEBIAN/control" <<CTRL
Package: $pkg_name
Version: $version
Section: utils
Priority: optional
Architecture: $deb_arch
Depends: $deps_line
Maintainer: Xinyi Zhang <42613048+xyztony999@users.noreply.github.com>
Description: 多技术栈项目的开发环境与 Git 身份管理工具箱
 项目发现、运行时登记与下载安装、Git 身份/目录绑定/仓库审计。
CTRL
dpkg-deb --build --root-owner-group "$stage" "${pkg_name}_${version}_${deb_arch}.deb"
echo "built ${pkg_name}_${version}_${deb_arch}.deb"

# ── rpm（Fedora/RHEL/openSUSE）——仅 CLI ──
if [ "$variant" != gui ]; then
  rpm_stage="$stage-rpm"
  rm -rf "$rpm_stage"
  mkdir -p "$rpm_stage/SPECS" "$rpm_stage/BUILD" "$rpm_stage/RPMS" "$rpm_stage/SOURCES"
  cat > "$rpm_stage/SPECS/dev-toolbox.spec" <<SPEC
Name:           dev-toolbox
Version:        $version
Release:        1
Summary:        多技术栈项目的开发环境与 Git 身份管理工具箱（CLI）
License:        MIT
URL:            https://github.com/xyztony999/dev-toolbox
Requires:       git
AutoReqProv:    no

%description
项目发现、运行时登记与下载安装、Git 身份/目录绑定/仓库审计。

%install
rm -rf "%{buildroot}"
install -D -m 0755 "%{_pkgbin}" "%{buildroot}/usr/bin/dev-toolbox"
install -D -m 0644 "%{_pkgrepo}/LICENSE" "%{buildroot}/usr/share/licenses/dev-toolbox/LICENSE"
install -D -m 0644 "%{_pkgrepo}/NOTICE" "%{buildroot}/usr/share/licenses/dev-toolbox/NOTICE"
install -D -m 0644 "%{_pkgrepo}/README.md" "%{buildroot}/usr/share/doc/dev-toolbox/README.md"

%files
/usr/bin/dev-toolbox
/usr/share/licenses/dev-toolbox/LICENSE
/usr/share/licenses/dev-toolbox/NOTICE
/usr/share/doc/dev-toolbox/README.md
SPEC
  rpmbuild -bb --define "_topdir $rpm_stage" --define "_rpmdir $(pwd)" \
    --define "_builddir $rpm_stage/BUILD" --define "_pkgbin $bin" --define "_pkgrepo $repo" \
    --target "$rpm_arch" \
    "$rpm_stage/SPECS/dev-toolbox.spec"
  find "$(pwd)" -name "dev-toolbox-*.${rpm_arch}.rpm" -exec mv {} . \;
  echo "built dev-toolbox-$version-1.${rpm_arch}.rpm"
fi
