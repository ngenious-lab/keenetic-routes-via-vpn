#!/bin/sh

# Функция для логирования и выхода при ошибке
fail() {
    echo "Ошибка: $1" >&2
    exit 1
}

# Проверка наличия opkg и Entware
command -v opkg >/dev/null 2>&1 || fail "opkg не найден. Убедитесь, что Entware установлен. См. https://help.keenetic.com/hc/en-us/articles/360000374559-Entware"

# Проверка интернета
ping -c 1 github.com >/dev/null 2>&1 || fail "Нет подключения к интернету. Проверьте сеть и повторите попытку."

# Обновление списка пакетов
echo "Обновление списка пакетов Entware..."
opkg update || fail "Не удалось обновить список пакетов. Проверьте интернет или конфигурацию Entware."

# Установка зависимостей (без golang и yq, если они недоступны)
echo "Установка зависимостей (git, git-http, ca-bundle, ca-certificates)..."
opkg install git git-http ca-bundle ca-certificates || fail "Не удалось установить зависимости. Проверьте интернет или место на диске."

# Проверка наличия бинарника vpn-router
if [ ! -f "/opt/bin/vpn-router" ]; then
    echo "Бинарник vpn-router не найден в /opt/bin."
    echo "Скомпилируйте его на другом устройстве (GOARCH=mipsle GOOS=linux go build -o vpn-router main.go) и скопируйте в /opt/bin/vpn-router."
    fail "Отсутствует бинарник vpn-router."
fi

# Создание директорий
echo "Создание директорий..."
mkdir -p /opt/etc/vpn-router || fail "Не удалось создать /opt/etc/vpn-router. Проверьте права доступа."
mkdir -p /opt/bin || fail "Не удалось создать /opt/bin. Проверьте права доступа."
mkdir -p /opt/etc/ndm/ifstatechanged.d || fail "Не удалось создать /opt/etc/ndm/ifstatechanged.d. Проверьте права доступа."

# Клонирование RockBlack-VPN/ip-address
if [ -d "/opt/etc/ip-address" ]; then
    echo "Директория /opt/etc/ip-address уже существует."
    if [ -n "$(ls -A /opt/etc/ip-address)" ]; then
        echo "Предупреждение: /opt/etc/ip-address не пуста. Пропускаю клонирование."
    else
        rm -rf /opt/etc/ip-address || fail "Не удалось удалить пустую директорию /opt/etc/ip-address."
        echo "Клонирование RockBlack-VPN/ip-address..."
        git clone https://github.com/RockBlack-VPN/ip-address /opt/etc/ip-address || fail "Не удалось клонировать RockBlack-VPN/ip-address. Проверьте интернет или права доступа."
    fi
else
    echo "Клонирование RockBlack-VPN/ip-address..."
    git clone https://github.com/RockBlack-VPN/ip-address /opt/etc/ip-address || fail "Не удалось клонировать RockBlack-VPN/ip-address. Проверьте интернет или права доступа."
fi

# Клонирование репозитория сервиса
echo "Клонирование репозитория сервиса..."
rm -rf /tmp/vpn-router
git clone https://github.com/ngenious-lab/keenetic-routes-via-vpn /tmp/vpn-router || fail "Не удалось клонировать репозиторий сервиса. Проверьте интернет или URL репозитория."
cd /tmp/vpn-router || fail "Не удалось перейти в /tmp/vpn-router."

# Проверка наличия config.yaml.example
if [ ! -f "config.yaml.example" ]; then
    echo "Предупреждение: config.yaml.example не найден. Создается базовый config.yaml."
    cat <<EOF > /opt/etc/vpn-router/config.yaml
vpn_interface: "ovpn_br0"
repo_dir: "/opt/etc/ip-address"
files:
    - "Global/Youtube/youtube.bat"
    - "Global/Instagram/instagram.bat"
EOF
else
    cp config.yaml.example /opt/etc/vpn-router/config.yaml || fail "Не удалось скопировать config.yaml.example."
fi

# Установка хук-скрипта (без yq)
echo "Установка хук-скрипта..."
cat <<EOF > /opt/etc/vpn-router/ifstatechanged.sh
#!/bin/sh
IFACE=\$(grep 'vpn_interface' /opt/etc/vpn-router/config.yaml | cut -d'"' -f2)
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
chmod +x /opt/etc/vpn-router/ifstatechanged.sh || fail "Не удалось установить права на ifstatechanged.sh."
ln -sf /opt/etc/vpn-router/ifstatechanged.sh /opt/etc/ndm/ifstatechanged.d/vpn-router.sh || fail "Не удалось создать симлинк для хука."

# Установка cron-задания
echo "Установка cron-задания для ежедневного обновления..."
echo "0 0 * * * /opt/bin/vpn-router update" > /opt/etc/cron.d/vpn-router || fail "Не удалось создать cron-задание."

# Очистка
echo "Очистка временных файлов..."
rm -rf /tmp/vpn-router || fail "Не удалось удалить временные файлы."

echo "Установка завершена успешно!"
echo "1. Отредактируйте /opt/etc/vpn-router/config.yaml, указав ваш VPN-интерфейс и нужные файлы."
echo "2. Перезапустите VPN-соединение для применения маршрутов."
echo "Для ручного тестирования:"
echo "- /opt/bin/vpn-router update (обновить маршруты)"
echo "- /opt/bin/vpn-router start (применить маршруты)"
echo "- /opt/bin/vpn-router stop (удалить маршруты)"
echo "Логи: /var/log/messages или настройте /opt/etc/vpn-router/ifstatechanged.sh для записи в файл."