#!/usr/bin/env bash
set -euo pipefail

# Параметры
OUTDIR="./bin"
NAME="vpn-router"
PKG="."   # пакет для сборки (.) — текущая директория с main.go

# Настройки GOMIPS по умолчанию (если нужен другой, редактируй)
GOMIPS_FOR_MIPS="softfloat"  # замените на hardfloat если требуется

# Убедимся, что go доступен
if ! command -v go >/dev/null 2>&1; then
  echo "Go не найден. Установите Go и повторите."
  exit 1
fi

# Очистка/создание выходной папки
rm -rf "$OUTDIR"
mkdir -p "$OUTDIR"

# Функция сборки
build() {
  local goos="$1"
  local goarch="$2"
  local gomips="$3"   # может быть пустой
  local outname="$4"

  echo "Собираю: GOOS=${goos} GOARCH=${goarch} ${gomips:+GOMIPS=${gomips}} ..."
  env CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" ${gomips:+GOMIPS="$gomips"} \
    go build -trimpath -o "$OUTDIR/$outname" "$PKG"

  chmod +x "$OUTDIR/$outname"
  echo "Готово: $OUTDIR/$outname"
}

# 1) linux/mips (big-endian)
build "linux" "mips" "$GOMIPS_FOR_MIPS" "${NAME}-linux-mips"

# 2) linux/mipsle (little-endian)
# (GOMIPS применяется только к mips; для mipsle обычно не обязателен,
# но можно указывать GOMIPS тоже если нужно)
build "linux" "mipsle" "" "${NAME}-linux-mipsle"

# 3) linux/arm64 (aarch64)
build "linux" "arm64" "" "${NAME}-linux-aarch64"

# Генерация SHA256 сумм
echo "Генерирую SHA256 checksums..."
cd "$OUTDIR"
# prefer sha256sum if present, otherwise use shasum -a 256
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum "${NAME}-linux-mips" > "${NAME}-linux-mips.sha256"
  sha256sum "${NAME}-linux-mipsle" > "${NAME}-linux-mipsle.sha256"
  sha256sum "${NAME}-linux-aarch64" > "${NAME}-linux-aarch64.sha256"
else
  shasum -a 256 "${NAME}-linux-mips" > "${NAME}-linux-mips.sha256"
  shasum -a 256 "${NAME}-linux-mipsle" > "${NAME}-linux-mipsle.sha256"
  shasum -a 256 "${NAME}-linux-aarch64" > "${NAME}-linux-aarch64.sha256"
fi

# Вывод результата
echo "Сборка завершена. Содержимое $OUTDIR:"
ls -lh "$OUTDIR"
echo ""
echo "Проверьте size и, при необходимости, подпишите/загрузите артефакты."
