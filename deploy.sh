#!/usr/bin/env bash
# NSP өөрчлөлттэй (nsp offline push г.м) openim-server-ийг Docker image болгож build хийгээд
# Nexus Docker registry (repo.nextsocial.mn) руу push хийнэ. Container-ийг энэ script асаадаггүй —
# prod дээр docker compose-ийн OPENIM_SERVER_IMAGE-ийг доор хэвлэгдэх image-ээр солиод
# `docker compose pull openim-server && docker compose up -d openim-server` хийнэ.
#
# Шаардлагатай:
#   - `docker login repo.nextsocial.mn` хийсэн байх
#   - Go суулгасан байх: Dockerfile нь host-ийн Go module cache-ийг (modcache build context)
#     ашигладаг тул build-ийн өмнө `go mod download`-оор cache-ийг бүрэн татна
#
# Ашиглах: ./deploy.sh
# Сонголт: VERSION=v3.8.3-nsp.1 ./deploy.sh, PLATFORM=linux/arm64 ./deploy.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

IMAGE_NAME="nsp-openim-server"
REGISTRY="repo.nextsocial.mn"
# Prod server-ийн архитектур. Mac (arm64) дээр build хийхэд default нь arm64 болж,
# server дээр "no matching manifest for linux/amd64" алдаа өгдөг тул заавал зааж өгнө.
PLATFORM="${PLATFORM:-linux/amd64}"
MODCACHE="$(go env GOMODCACHE)"

# Upstream хувилбар (version/version) + git commit. Commit хийгээгүй өөрчлөлттэй бол -dirty нэмнэ.
if [[ -z "${VERSION:-}" ]]; then
	UPSTREAM_VERSION="$(tr -d '[:space:]' < version/version)"
	GIT_SHA="$(git rev-parse --short HEAD)"
	VERSION="${UPSTREAM_VERSION}-nsp-${GIT_SHA}"
	if [[ -n "$(git status --porcelain --untracked-files=no)" ]]; then
		VERSION="${VERSION}-dirty"
	fi
fi

echo "==> Go module cache-ийг шинэчилж байна ($MODCACHE)..."
go mod download

if ! docker info >/dev/null 2>&1; then
	echo "Алдаа: Docker daemon-тай холбогдож чадсангүй — Docker Desktop асаалттай эсэхийг шалгана уу" >&2
	exit 1
fi

# --push: build хийсэн image-ийг шууд registry руу илгээнэ. Builder нь docker-container
# driver-тэй үед image local docker руу орохгүй тул тусдаа `docker push` ажилладаггүй.
echo "==> Docker image build хийж, Nexus registry-рүү push хийж байна ($IMAGE_NAME:$VERSION, $PLATFORM)..."
docker buildx build \
	--platform "$PLATFORM" \
	--build-context modcache="$MODCACHE" \
	-t "$REGISTRY/$IMAGE_NAME:$VERSION" \
	-t "$REGISTRY/$IMAGE_NAME:latest" \
	--push \
	.

echo "==> Амжилттай. Prod-ийн .env дээр:"
echo "    OPENIM_SERVER_IMAGE=$REGISTRY/$IMAGE_NAME:$VERSION"
