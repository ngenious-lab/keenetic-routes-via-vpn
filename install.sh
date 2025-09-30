#!/bin/sh

# Установка зависимостей (git, go, yq для парсинга yaml в shell)
opkg update
opkg install git git-http ca-bundle ca-certificates golang yq

# Директории
mkdir -p /opt/etc/vpn-router
mkdir -p /opt/bin
mkdir -p /opt/etc/ndm/ifstatechanged.d

# Клон RockBlack репо
git clone https://github.com/RockBlack-VPN/ip-address /opt/etc/ip-address

# Клон вашего репо с кодом (замените на реальный URL репо)
git clone https://github.com/ngenious-lab/keenetic-routes-via-vpn.git /tmp/vpn-router
cd /tmp/vpn-router

# Компиляция (статическая для MIPS/ARM Keenetic)
GOARCH=mipsle GOOS=linux go build -o /opt/bin/vpn-router main.go

# Копируем config example (предполагая, что в репо есть config.yaml.example)
cp config.yaml.example /opt/etc/vpn-router/config.yaml

# Хук скрипт
cat <<EOF > /opt/etc/vpn-router/ifstatechanged.sh
#!/bin/sh
IFACE=\$(yq e '.vpn_interface' /opt/etc/vpn-router/config.yaml)
if [ "\$INTERFACE" != "\$IFACE" ]; then
  exit 0
fi
case "\$STATE" in
  up)
    /opt/bin/vpn-router start
    ;;
  down)
    /opt/bin/vpn-router stop
    ;;
esac
EOF
chmod +x /opt/etc/vpn-router/ifstatechanged.sh
ln -s /opt/etc/vpn-router/ifstatechanged.sh /opt/etc/ndm/ifstatechanged.d/vpn-router.sh

# Cron для обновления раз в день
echo "0 0 * * * /opt/bin/vpn-router update" > /opt/etc/cron.d/vpn-router

# Очистка
rm -rf /tmp/vpn-router

echo "Installation complete. Edit /opt/etc/vpn-router/config.yaml, then restart VPN to apply."