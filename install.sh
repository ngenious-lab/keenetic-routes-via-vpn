#!/bin/sh
set -e

REPO="https://github.com/ngenious-lab/keenetic-routes-via-vpn"
BINARY_DEST="/opt/bin/vpn-router"
REPO_IPS="https://github.com/RockBlack-VPN/ip-address"
IP_REPO_DIR="/opt/etc/ip-address"
CONFIG_DIR="/opt/etc/vpn-router"
LOG_DIR="/opt/var/log"
CRON_FILE="/opt/etc/cron.d/vpn-router"

fail() { echo "ERROR: $1" >&2; exit 1; }

# Проверка opkg (Entware)
command -v opkg >/dev/null 2>&1 || fail "opkg не найден. Установите Entware: https://help.keenetic.com"

echo "[*] Обновление opkg"
opkg update || true

echo "[*] Установка зависимостей (если отсутствуют)"
opkg install git git-http ca-bundle ca-certificates curl coreutils-sha256sum || true

# Создание директорий
mkdir -p /opt/bin "$CONFIG_DIR" "$IP_REPO_DIR" "$LOG_DIR" /opt/etc/ndm/ifstatechanged.d

# Скачивание бинарника (если не существует)
if [ ! -x "$BINARY_DEST" ]; then
  ARCH="$(uname -m)"
  case "$ARCH" in
#    mips*)   BNAME="vpn-router-mips" ;;
    mips*|mipsel*|mips32el*) BNAME="vpn-router-mipsel" ;;
    aarch64*|arm64*) BNAME="vpn-router-aarch64" ;;
    arm*) echo "ARM detected, trying aarch64 build fallback"; BNAME="vpn-router-aarch64" ;;
    *) fail "Архитектура $ARCH не поддерживается в автоматическом скачивании. Соберите вручную." ;;
  esac

  echo "[*] Скачивание бинарника $BNAME для $ARCH..."
  curl -fsSL -o "$BINARY_DEST" "https://github.com/ngenious-lab/keenetic-routes-via-vpn/releases/v1.0.0/download/$BNAME" || fail "Не удалось скачать бинарник."
  chmod +x "$BINARY_DEST"
  echo "[*] Бинарник размещён в $BINARY_DEST"
else
  echo "[*] Бинарник уже есть: $BINARY_DEST"
fi

# Клонирование репозитория с IP-адресами если отсутствует
if [ -d "$IP_REPO_DIR/.git" ]; then
  echo "[*] Репозиторий ip-address уже клонирован в $IP_REPO_DIR"
else
  echo "[*] Клонируем RockBlack-VPN/ip-address в $IP_REPO_DIR..."
  rm -rf "$IP_REPO_DIR"
  git clone "$REPO_IPS" "$IP_REPO_DIR" || {
    echo "Предупреждение: не удалось клонировать $REPO_IPS"
  }
fi

# Конфиг по умолчанию
if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
  echo "[*] Создаём базовый конфиг в $CONFIG_DIR/config.yaml"
  cat <<EOF > "$CONFIG_DIR/config.yaml"
vpn_interface: "nwg1"
repo_dir: "$IP_REPO_DIR"
files:
  - "Global/Youtube/youtube_minimum.bat"
ips:
  - "192.168.100.0/24"
EOF
fi

# Хук для ndm (ifstatechanged)
HOOK="$CONFIG_DIR/ifstatechanged.sh"
cat <<'EOF' > "$HOOK"
#!/bin/sh
# hook for ndm ifstatechanged
IFACE=`grep 'vpn_interface' /opt/etc/vpn-router/config.yaml | cut -d'"' -f2`
[ -z "$IFACE" ] && exit 1
# run only for our vpn interface
[ "$INTERFACE" != "$IFACE" ] && exit 0

LOG="/opt/var/log/vpn-router.log"
mkdir -p /opt/var/log

case "$STATE" in
  up)
    echo "[$(date)] $IFACE up" >> $LOG 2>&1
    ip route flush table 1000 2>/dev/null || true
    ip rule add from 192.168.0.0/16 lookup main prio 100 2>/dev/null || true
    /opt/bin/vpn-router start >> $LOG 2>&1 || echo "vpn-router start failed" >> $LOG 2>&1
    ;;
  down)
    echo "[$(date)] $IFACE down" >> $LOG 2>&1
    ip route flush table 1000 2>/dev/null || true
    ip rule del from 192.168.0.0/16 lookup main prio 100 2>/dev/null || true
    /opt/bin/vpn-router stop >> $LOG 2>&1 || echo "vpn-router stop failed" >> $LOG 2>&1
    ;;
esac
exit 0
EOF

chmod +x "$HOOK"
ln -sf "$HOOK" /opt/etc/ndm/ifstatechanged.d/vpn-router.sh

cat <<'EOF' > /opt/etc/init.d/S99vpn-router
#!/bin/sh

ENABLED=yes
PROCS=vpn-router
ARGS="start"
PREARGS=""
DESC="VPN Router service"
PATH=/opt/bin:/opt/sbin:/usr/bin:/bin:/usr/sbin:/sbin

. /opt/etc/init.d/rc.func

start() {
    echo "Starting $DESC..."
    /opt/bin/vpn-router start
}

stop() {
    echo "Stopping $DESC..."
    /opt/bin/vpn-router stop
}
EOF

chmod +x /opt/etc/init.d/S99vpn-router

# Создание cron с git pull + vpn-router update
cat <<EOF > "$CRON_FILE"
# обновление списка адресов и применение маршрутов
0 3 * * * cd $IP_REPO_DIR && git pull >> /opt/var/log/vpn-router.log 2>&1
0 4 * * * /opt/bin/vpn-router update >> /opt/var/log/vpn-router.log 2>&1
EOF

echo "[*] Cron задания установлены: $CRON_FILE"

echo "[*] Установка завершена. Отредактируйте $CONFIG_DIR/config.yaml при необходимости."
echo "Примеры команд:"
echo "  /opt/bin/vpn-router update"
echo "  /opt/bin/vpn-router start"
echo "  /opt/bin/vpn-router stop"
echo "  /opt/bin/vpn-router status"
echo "  /opt/bin/vpn-router update-repo"
