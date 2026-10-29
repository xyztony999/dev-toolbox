#!/usr/bin/env bash
# 构建 dev-toolbox CLI 的 deb 与 rpm 包。
# 用法：build-linux-packages.sh <版本(不带v)> <amd64|arm64> <静态CLI二进制路径>
# 产物输出到当前目录：dev-toolbox_<版本>_<deb架构>.deb / dev-toolbox-<版本>-1.<rpm架构>.rpm
set -euo pipefail

version="${1:?用法: build-linux-packages.sh <版本> <amd64|arm64> <二进制路径>}"
deb_arch="${2:?用法: build-linux-packages.sh <版本> <amd64|arm64> <二进制路径>}"
bin="${3:?用法: build-linux-packages.sh <版本> <amd64|arm64> <二进制路径>}"

case "$deb_arch" in
  amd64) rpm_arch=x86_64 ;;
  arm64) rpm_arch=aarch64 ;;
  *) echo "不支持的架构：$deb_arch" >&2; exit 2 ;;
esac

repo="$(cd "$(dirname "$0")/../.." && pwd)"
stage="$(mktemp -d)"
trap 'rm -rf "$stage"' EXIT

install -D -m 0755 "$bin" "$stage/usr/bin/dev-toolbox"
install -D -m 0644 "$repo/LICENSE" "$stage/usr/share/licenses/dev-toolbox/LICENSE"
install -D -m 0644 "$repo/NOTICE" "$stage/usr/share/licenses/dev-toolbox/NOTICE"
install -D -m 0644 "$repo/README.md" "$stage/usr/share/doc/dev-toolbox/README.md"

# ── deb（Debian/Ubuntu）──
mkdir -p "$stage/DEBIAN"
cat > "$stage/DEBIAN/control" <<CTRL
Package: dev-toolbox
Version: $version
Section: utils
Priority: optional
Architecture: $deb_arch
Depends: git
Maintainer: Xinyi Zhang <42613048+xyztony999@users.noreply.github.com>
Description: 多技术栈项目的开发环境与 Git 身份管理工具箱（CLI）
 项目发现、运行时登记与下载安装、Git 身份/目录绑定/仓库审计。
 本包仅含 CLI；桌面客户端见 GitHub Releases。
CTRL
dpkg-deb --build --root-owner-group "$stage" "dev-toolbox_${version}_${deb_arch}.deb"
echo "built dev-toolbox_${version}_${deb_arch}.deb"

# ── rpm（Fedora/RHEL/openSUSE）──
rpm_stage="$stage-rpm"
rm -rf "$rpm_stage"
mkdir -p "$rpm_stage/SPECS" "$rpm_stage/BUILD" "$rpm_stage/RPMS" "$rpm_stage/SOURCES"
cp -r "$stage/usr" "$rpm_stage/BUILD/"
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
本包仅含 CLI；桌面客户端见 GitHub Releases。

%files
/usr/bin/dev-toolbox
/usr/share/licenses/dev-toolbox/LICENSE
/usr/share/licenses/dev-toolbox/NOTICE
/usr/share/doc/dev-toolbox/README.md
SPEC
rpmbuild -bb --define "_topdir $rpm_stage" --define "_rpmdir $(pwd)" \
  --define "_builddir $rpm_stage/BUILD" --target "$rpm_arch" \
  "$rpm_stage/SPECS/dev-toolbox.spec"
find "$(pwd)" -name "dev-toolbox-*.${rpm_arch}.rpm" -exec mv {} . \;
echo "built dev-toolbox-$version-1.${rpm_arch}.rpm"
