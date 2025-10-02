#!/bin/sh
set -e

log() { echo "[INFO] $1"; }
fail() { echo "[ERROR] $1" >&2; exit 1; }

# Проверка Entware
command -v opkg >/dev/null 2>&1 || fail "opkg не найден. Установите Entware (https://help.keenetic.com/hc/ru/articles/360000374559)."

# Проверка интернета
ping -c 1 github.com >/dev/null 2>&1 || fail "Нет доступа в интернет."

log "Обновление пакетов..."
opkg update

log "Установка зависимостей..."
opkg install git git-http ca-bundle ca-certificates curl coreutils-sha256sum || true

# Определение архитектуры
ARCH=$(uname -m)
case $ARCH in
  mips*)     BINARY="vpn-router-mips" ;;
  mipsel*|mips32el*) BINARY="vpn-router-mipsel" ;;
  aarch64*|arm64*)   BINARY="vpn-router-aarch64" ;;
  *) fail "Архитектура $ARCH не поддерживается. Соберите бинарник вручную." ;;
esac

# Скачивание бинарника
if [ ! -x /opt/bin/vpn-router ]; then
  log "Скачивание $BINARY..."
  curl -L -o /opt/bin/vpn-router "https://github.com/ngenious-lab/keenetic-routes-via-vpn/releases/latest/download/$BINARY" || fail "Не удалось скачать бинарник."
  chmod +x /opt/bin/vpn-router
fi

# Создание директорий
mkdir -p /opt/etc/vpn-router /opt/etc/ndm/ifstatechanged.d /opt/var/log

# Конфиг
if [ ! -f /opt/etc/vpn-router/config.yaml ]; then
  log "Создание базового config.yaml..."
  cat <<EOF > /opt/etc/vpn-router/config.yaml
vpn_interface: "interface_name"
repo_dir: "/opt/etc/ip-address"
files:
  - "Global/Youtube/youtube_minimum.bat"
ips:
  - "192.168.100.0/24"
EOF
fi

# Хук
cat <<'EOF' > /opt/etc/vpn-router/ifstatechanged.sh
#!/bin/sh
IFACE=$(grep 'vpn_interface' /opt/etc/vpn-router/config.yaml | cut -d'"' -f2)
[ "$INTERFACE" != "$IFACE" ] && exit 0

LOG="/opt/var/log/vpn-router.log"
mkdir -p /opt/var/log

case "$STATE" in
  up)
    echo "[$(date)] VPN $IFACE up, применяем маршруты" >> $LOG
    ip route flush table 1000
    ip rule add from 192.168.0.0/16 lookup main prio 100 || true
    ip rule add from 172.0.0.0/8 lookup main prio 100 || true
    /opt/bin/vpn-router start >> $LOG 2>&1
    ;;
  down)
    echo "[$(date)] VPN $IFACE down, очищаем маршруты" >> $LOG
    ip route flush table 1000
    ip rule del from 192.168.0.0/16 lookup main prio 100 || true
    /opt/bin/vpn-router stop >> $LOG 2>&1
    ;;
esac
EOF
chmod +x /opt/etc/vpn-router/ifstatechanged.sh
ln -sf /opt/etc/vpn-router/ifstatechanged.sh /opt/etc/ndm/ifstatechanged.d/vpn-router.sh

# Cron обновление
echo "0 0 * * * /opt/bin/vpn-router update >> /opt/var/log/vpn-router.log 2>&1" > /opt/etc/cron.d/vpn-router

log "Установка завершена."
log "1. Отредактируйте /opt/etc/vpn-router/config.yaml (vpn_interface)."
log "2. Проверьте, что ip rule показывает таблицу 1000."
log "3. Перезапустите VPN-соединение."
