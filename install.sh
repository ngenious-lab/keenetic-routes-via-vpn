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

# Установка зависимостей
echo "Установка зависимостей (git, git-http, ca-bundle, ca-certificates, curl)..."
opkg install git git-http ca-bundle ca-certificates curl || fail "Не удалось установить зависимости. Проверьте интернет или место на диске."

# Проверка наличия sha256sum
command -v sha256sum >/dev/null 2>&1 || {
    echo "Установка sha256sum (входит в coreutils-sha256sum)..."
    opkg install coreutils-sha256sum || {
        echo "Предупреждение: Не удалось установить coreutils-sha256sum. Проверка SHA256 будет пропущена."
        SHA256_AVAILABLE=0
    }
}

# Проверка и скачивание бинарника
echo "Проверка бинарника vpn-router..."
if [ ! -f "/opt/bin/vpn-router" ] || [ ! -x "/opt/bin/vpn-router" ]; then
    ARCH=`uname -m`
    case $ARCH in
        mips*)
            BINARY="vpn-router-mips"
            ;;
        mipsel*|mips32el*)
            BINARY="vpn-router-mipsel"
            ;;
        aarch64*|arm64*)
            BINARY="vpn-router-aarch64"
            ;;
        arm*)
            echo "Предупреждение: ARM архитектура. Попробуем mipsel как fallback, но лучше соберите вручную."
            BINARY="vpn-router-mipsel"
            ;;
        *)
            echo "Неизвестная архитектура: $ARCH. Доступны: mips, mipsel, aarch64."
            echo "Соберите бинарник локально и скопируйте в /opt/bin/vpn-router."
            fail "Поддержка архитектуры $ARCH не реализована."
            ;;
    esac
    
    echo "Скачивание $BINARY для архитектуры $ARCH из GitHub Releases..."
    curl -L -o /opt/bin/vpn-router "https://github.com/ngenious-lab/keenetic-routes-via-vpn/releases/latest/download/$BINARY" || {
        echo "Ошибка скачивания бинарника. Проверьте интернет или создайте Release в репозитории."
        echo "Альтернатива: Соберите бинарник локально (GOOS=linux GOARCH=$ARCH go build -o /opt/bin/vpn-router main.go) и скопируйте."
        fail "Не удалось скачать бинарник."
    }
    
    # Проверка наличия бинарника перед установкой прав
    [ -f "/opt/bin/vpn-router" ] || fail "Бинарник /opt/bin/vpn-router не создан. Проверьте процесс скачивания."
    
    if [ -z "$SHA256_AVAILABLE" ] || [ "$SHA256_AVAILABLE" -ne 0 ]; then
        echo "Скачивание SHA256 checksum для $BINARY..."
        curl -L -o /opt/bin/$BINARY.sha256 "https://github.com/ngenious-lab/keenetic-routes-via-vpn/releases/latest/download/$BINARY.sha256" || {
            echo "Предупреждение: Не удалось скачать SHA256 checksum."
            if [ -t 0 ]; then
                echo "Продолжить без проверки SHA256? (y/n)"
                read -r response
                if [ "$response" != "y" ] && [ "$response" != "Y" ]; then
                    rm -f /opt/bin/vpn-router
                    fail "Установка прервана из-за отсутствия SHA256 checksum."
                fi
            else
                echo "Неинтерактивный режим, пропускаем проверку SHA256."
            fi
        }
        
        # Проверка SHA256, если checksum-файл скачан
        if [ -f "/opt/bin/$BINARY.sha256" ]; then
            echo "Проверка целостности бинарника..."
            sha256sum /opt/bin/vpn-router | cut -d" " -f1 | tr '[:upper:]' '[:lower:]' > /opt/bin/computed.sha256
            tr '[:upper:]' '[:lower:]' < /opt/bin/$BINARY.sha256 > /opt/bin/expected.sha256
            if cmp /opt/bin/computed.sha256 /opt/bin/expected.sha256; then
                echo "SHA256 проверка пройдена: бинарник цел."
                rm -f /opt/bin/computed.sha256 /opt/bin/expected.sha256
            else
                echo "Ошибка: SHA256 проверка не пройдена: бинарник поврежден или неверный."
                if [ -t 0 ]; then
                    echo "Продолжить установку без проверки SHA256? (y/n)"
                    read -r response
                    if [ "$response" != "y" ] && [ "$response" != "Y" ]; then
                        rm -f /opt/bin/vpn-router /opt/bin/$BINARY.sha256 /opt/bin/computed.sha256 /opt/bin/expected.sha256
                        fail "Установка прервана из-за неудачной проверки SHA256."
                    fi
                else
                    echo "Неинтерактивный режим, пропускаем проверку SHA256."
                fi
                rm -f /opt/bin/computed.sha256 /opt/bin/expected.sha256
            fi
        fi
    else
        echo "sha256sum не доступен, пропускаем проверку SHA256."
    fi
    
    chmod +x /opt/bin/vpn-router || fail "Не удалось установить права на /opt/bin/vpn-router."
    echo "Бинарник скачан и готов к использованию."
else
    echo "Бинарник /opt/bin/vpn-router уже существует и исполняемый."
fi

# Создание директорий
echo "Создание директорий..."
mkdir -p /opt/etc/vpn-router || fail "Не удалось создать /opt/etc/vpn-router. Проверьте права доступа."
mkdir -p /opt/etc/ndm/ifstatechanged.d || fail "Не удалось создать /opt/etc/ndm/ifstatechanged.d. Проверьте права доступа."
mkdir -p /opt/var/log || fail "Не удалось создать /opt/var/log. Проверьте права доступа."

# Клонирование RockBlack-VPN/ip-address
if [ -d "/opt/etc/ip-address" ]; then
    echo "Директория /opt/etc/ip-address уже существует."
    if [ -n "`ls -A /opt/etc/ip-address`" ]; then
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

# Клонирование репозитория сервиса (для config и скриптов)
echo "Клонирование репозитория сервиса..."
rm -rf /tmp/vpn-router
git clone https://github.com/ngenious-lab/keenetic-routes-via-vpn /tmp/vpn-router || fail "Не удалось клонировать репозиторий сервиса. Проверьте интернет или URL репозитория."
cd /tmp/vpn-router || fail "Не удалось перейти в /tmp/vpn-router."

# Проверка наличия config.yaml.example
if [ ! -f "config.yaml.example" ]; then
    echo "Предупреждение: config.yaml.example не найден. Создается базовый config.yaml."
    cat <<EOF > /opt/etc/vpn-router/config.yaml
vpn_interface: "nwg1"
repo_dir: "/opt/etc/ip-address"
files:
  - "Global/Youtube/youtube.bat"
  - "Global/Instagram/instagram.bat"
ips:
  - "192.168.100.0/24"
  - "10.0.0.0/16"
EOF
else
    cp config.yaml.example /opt/etc/vpn-router/config.yaml || fail "Не удалось скопировать config.yaml.example."
fi

# Установка хук-скрипта
echo "Установка хук-скрипта..."
cat <<EOF > /opt/etc/vpn-router/ifstatechanged.sh
#!/bin/sh
# Хук-скрипт для обработки изменения состояния VPN-интерфейса

# Получаем имя VPN-интерфейса из конфигурации, используя обратные кавычки для совместимости с BusyBox
IFACE=\`grep 'vpn_interface' /opt/etc/vpn-router/config.yaml | cut -d'"' -f2\`
if [ -z "\$IFACE" ]; then
  echo "Ошибка: Не удалось определить vpn_interface из /opt/etc/vpn-router/config.yaml" >> /opt/var/log/vpn-router.log 2>&1
  exit 1
fi

# Пропускаем, если интерфейс не совпадает
if [ "\$INTERFACE" != "\$IFACE" ]; then
  exit 0
fi

# Создаем директорию для логов, если не существует
mkdir -p /opt/var/log 2>/dev/null

# Проверка состояния VPN-интерфейса
if ! ip link show "\$IFACE" up >/dev/null 2>&1; then
  echo "Предупреждение: VPN-интерфейс \$IFACE не активен. Пропускаем применение маршрутов." >> /opt/var/log/vpn-router.log 2>&1
  exit 0
fi

# Проверка доступности интернета через VPN
if ! ping -c 1 -W 2 -I "\$IFACE" 8.8.8.8 >/dev/null 2>&1; then
  echo "Предупреждение: VPN-интерфейс \$IFACE не имеет доступа к интернету. Пропускаем применение маршрутов." >> /opt/var/log/vpn-router.log 2>&1
  exit 0
fi

case "\$STATE" in
  up)
    echo "VPN-интерфейс \$IFACE включен. Применяем маршруты..." >> /opt/var/log/vpn-router.log 2>&1
    # Очищаем таблицу маршрутов 1000
    ip route flush table 1000 2>/dev/null || {
      echo "Предупреждение: Не удалось очистить таблицу маршрутов 1000" >> /opt/var/log/vpn-router.log 2>&1
    }
    # Защищаем локальную сеть от маршрутизации через VPN
    ip rule add from 192.168.0.0/16 lookup main prio 100 2>/dev/null || {
      echo "Предупреждение: Не удалось добавить правило для локальной сети 192.168.0.0/16" >> /opt/var/log/vpn-router.log 2>&1
    }
    # Запускаем vpn-router start
    /opt/bin/vpn-router start >> /opt/var/log/vpn-router.log 2>&1
    if [ \$? -ne 0 ]; then
      echo "Ошибка: /opt/bin/vpn-router start завершился с ошибкой" >> /opt/var/log/vpn-router.log 2>&1
      exit 2
    fi
    echo "Маршруты успешно применены для таблицы 1000" >> /opt/var/log/vpn-router.log 2>&1
    ;;
  down)
    echo "VPN-интерфейс \$IFACE выключен. Очищаем маршруты..." >> /opt/var/log/vpn-router.log 2>&1
    # Очищаем таблицу маршрутов и правила
    ip route flush table 1000 2>/dev/null || {
      echo "Предупреждение: Не удалось очистить таблицу маршрутов 1000" >> /opt/var/log/vpn-router.log 2>&1
    }
    ip rule del from 192.168.0.0/16 lookup main prio 100 2>/dev/null || true
    /opt/bin/vpn-router stop >> /opt/var/log/vpn-router.log 2>&1
    if [ \$? -ne 0 ]; then
      echo "Ошибка: /opt/bin/vpn-router stop завершился с ошибкой" >> /opt/var/log/vpn-router.log 2>&1
      exit 2
    fi
    echo "Маршруты успешно удалены из таблицы 1000" >> /opt/var/log/vpn-router.log 2>&1
    ;;
esac

exit 0
EOF
chmod +x /opt/etc/vpn-router/ifstatechanged.sh || fail "Не удалось установить права на ifstatechanged.sh."
ln -sf /opt/etc/vpn-router/ifstatechanged.sh /opt/etc/ndm/ifstatechanged.d/vpn-router.sh || fail "Не удалось создать симлинк для хука."
# Исправляем окончания строк
sed -i 's/\r$//' /opt/etc/vpn-router/ifstatechanged.sh || fail "Не удалось исправить окончания строк в ifstatechanged.sh."

# Установка cron-задания
echo "Установка cron-задания для ежедневного обновления..."
echo "0 0 * * * /opt/bin/vpn-router update >> /opt/var/log/vpn-router.log 2>&1" > /opt/etc/cron.d/vpn-router || fail "Не удалось создать cron-задание."

# Очистка
echo "Очистка временных файлов..."
rm -rf /tmp/vpn-router || fail "Не удалось удалить временные файлы."

echo "Установка завершена успешно!"
echo "1. Отредактируйте /opt/etc/vpn-router/config.yaml, указав ваш VPN-интерфейс (например, nwg1 для WireGuard), нужные файлы и кастомные IP-сети."
echo "2. Убедитесь, что VPN-интерфейс активен и имеет доступ к интернету (ping -I nwg1 8.8.8.8)."
echo "3. Перезапустите VPN-соединение для применения маршрутов."
echo "Для ручного тестирования:"
echo "- /opt/bin/vpn-router update (обновить и применить маршруты, включая кастомные IP)"
echo "- /opt/bin/vpn-router start (применить маршруты)"
echo "- /opt/bin/vpn-router stop (удалить маршруты)"
echo "- ip route show table 1000 (проверить активные маршруты)"
echo "Логи: /var/log/messages или /opt/var/log/vpn-router.log"