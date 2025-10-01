# Keenetic Routes via VPN

Этот проект предоставляет сервис на Go для маршрутизации определенного сетевого трафика через VPN-туннель на роутерах Keenetic с использованием [Entware](https://entware.net/). Сервис клонирует и обрабатывает списки IP-адресов из репозитория [RockBlack-VPN/ip-address](https://github.com/RockBlack-VPN/ip-address), а также позволяет задавать кастомные IP-сети в конфигурации. Это позволяет направлять трафик для таких сервисов, как YouTube, Instagram, Twitter или Netflix, а также пользовательских сетей через указанный VPN-интерфейс, в то время как остальной трафик идет через провайдера. Сервис легковесный, настраиваемый и разработан для бесшовной интеграции с Entware на Keenetic.

## Возможности

- **Выборочная маршрутизация**: Направляет трафик для указанных пользователем списков IP и кастомных IP-сетей через VPN-интерфейс, остальной трафик идет через провайдера.
- **Ежедневное обновление**: Автоматически обновляет списки IP из репозитория RockBlack-VPN раз в день через `git pull` и применяет их в таблицу маршрутов 1000.
- **Кастомные IP-сети**: Поддержка пользовательских IP-сетей в формате CIDR в конфигурационном файле.
- **Обработка ошибок**: При сбое обновления или отсутствии файла использует предыдущий список маршрутов.
- **Гибкая конфигурация**: Использует простой YAML-файл для указания VPN-интерфейса, файлов со списками IP и кастомных IP-сетей.
- **Интеграция с VPN**: Автоматически применяет/удаляет маршруты при включении/выключении VPN.
- **Логирование**: Вывод в syslog (`/var/log/messages`) и файл `/opt/var/log/vpn-router.log`.

## Требования

- Роутер Keenetic с установленным [Entware](https://entware.net/).
- Настроенное VPN-соединение (например, OpenVPN, WireGuard, PPTP) на роутере.
- Базовые навыки работы с SSH и командной строкой.

## Скачивание бинарников

Для удобства мы предоставляем готовые бинарники для популярных архитектур Keenetic роутеров. Скачайте версию, соответствующую архитектуре вашего устройства (проверьте с помощью `uname -m` на роутере). SHA256 checksum-файлы включены для верификации целостности бинарников.

| Архитектура | Команда для скачивания | Прямая ссылка | SHA256 |
|-------------|------------------------|---------------|--------|
| MIPS       | `curl -L -o /opt/bin/vpn-router https://github.com/ngenious-lab/keenetic-routes-via-vpn/releases/latest/download/vpn-router-mips` | [Скачать](https://github.com/ngenious-lab/keenetic-routes-via-vpn/releases/latest/download/vpn-router-mips) | [SHA256](https://github.com/ngenious-lab/keenetic-routes-via-vpn/releases/latest/download/vpn-router-mips.sha256) |
| MIPSel     | `curl -L -o /opt/bin/vpn-router https://github.com/ngenious-lab/keenetic-routes-via-vpn/releases/latest/download/vpn-router-mipsel` | [Скачать](https://github.com/ngenious-lab/keenetic-routes-via-vpn/releases/latest/download/vpn-router-mipsel) | [SHA256](https://github.com/ngenious-lab/keenetic-routes-via-vpn/releases/latest/download/vpn-router-mipsel.sha256) |
| AArch64    | `curl -L -o /opt/bin/vpn-router https://github.com/ngenious-lab/keenetic-routes-via-vpn/releases/latest/download/vpn-router-aarch64` | [Скачать](https://github.com/ngenious-lab/keenetic-routes-via-vpn/releases/latest/download/vpn-router-aarch64) | [SHA256](https://github.com/ngenious-lab/keenetic-routes-via-vpn/releases/latest/download/vpn-router-aarch64.sha256) |

После скачивания выполните `chmod +x /opt/bin/vpn-router`. Установочный скрипт (`install.sh`) автоматически скачивает бинарник и проверяет его целостность с помощью SHA256 checksum, игнорируя регистр символов. В неинтерактивном режиме (например, `curl | sh`) проверка SHA256 пропускается автоматически, если она не удалась. Для ручной проверки:
```bash
sha256sum /opt/bin/vpn-router | cut -d" " -f1 | tr '[:upper:]' '[:lower:]' > /opt/bin/computed.sha256
cat /opt/bin/vpn-router-<arch>.sha256 | tr '[:upper:]' '[:lower:]' > /opt/bin/expected.sha256
cmp /opt/bin/computed.sha256 /opt/bin/expected.sha256
```

**Примечание**: Замените `latest` на конкретную версию (например, `v1.0.0`), если требуется. Для автоматического обновления бинарников используйте GitHub API или инструмент вроде `eget` (см. [zyedidia/eget](https://github.com/zyedidia/eget)).

## Установка

Следуйте этим шагам для установки и настройки сервиса на роутере Keenetic с Entware:

1. **Убедитесь, что Entware установлен**  
   Проверьте, что Entware настроен на вашем роутере Keenetic. Инструкции по установке см. в [документации Keenetic по Entware](https://help.keenetic.com/hc/en-us/articles/360000374559-Entware).

2. **Проверьте VPN-соединение**  
   Убедитесь, что VPN-интерфейс (например, `nwg1` для WireGuard) активен и имеет доступ к интернету:
   ```bash
   ifconfig nwg1
   ping -I nwg1 8.8.8.8
   ```

3. **Запустите установочный скрипт**  
   Для неинтерактивной установки используйте:
   ```bash
   curl -sfL https://raw.githubusercontent.com/ngenious-lab/keenetic-routes-via-vpn/main/install.sh | sh
   ```
   Для интерактивной установки (чтобы отвечать на запросы о пропуске SHA256):
   ```bash
   curl -sL https://raw.githubusercontent.com/ngenious-lab/keenetic-routes-via-vpn/main/install.sh -o install.sh
   sh install.sh
   ```
   Скрипт:
   - Устанавливает зависимости (`git`, `git-http`, `ca-bundle`, `ca-certificates`, `curl`, `coreutils-sha256sum`).
   - Скачивает подходящий бинарник из GitHub Releases (на основе архитектуры роутера) и проверяет его SHA256 checksum.
   - Клонирует репозиторий [RockBlack-VPN/ip-address](https://github.com/RockBlack-VPN/ip-address) в `/opt/etc/ip-address`.
   - Настраивает сервис (конфиг, хук для VPN-интерфейса и задание cron для ежедневных обновлений).
   - Защищает локальную сеть (192.168.0.0/16) от маршрутизации через VPN.
   - Использует синтаксис, совместимый с BusyBox `ash`, и исправляет окончания строк.

4. **Настройте сервис**  
   Отредактируйте конфигурационный файл `/opt/etc/vpn-router/config.yaml`:
   ```yaml
   vpn_interface: "nwg1"  # Укажите ваш VPN-интерфейс (например, nwg1 для WireGuard)
   repo_dir: "/opt/etc/ip-address"  # Путь к клонированному репозиторию RockBlack-VPN
   files:
     - "Global/Youtube/youtube.bat"  # Пример: путь к файлу в репозитории
     - "Global/Instagram/instagram.bat"
   ips:
     - "192.168.100.0/24"  # Кастомная IP-сеть в формате CIDR
     - "10.0.0.0/16"       # Дополнительная кастомная сеть
   ```
   - **vpn_interface**: Имя вашего VPN-интерфейса (например, `ovpn_br0` для OpenVPN, `nwg1` для WireGuard).
   - **repo_dir**: Директория, куда клонирован репозиторий [RockBlack-VPN/ip-address](https://github.com/RockBlack-VPN/ip-address) (по умолчанию `/opt/etc/ip-address`).
   - **files**: Список файлов (`.bat` или аналогичных) из репозитория [RockBlack-VPN/ip-address](https://github.com/RockBlack-VPN/ip-address) с маршрутами IP (в формате `route ADD IP MASK MASK`). Указывайте пути относительно `repo_dir`.
   - **ips**: Список кастомных IP-сетей в формате CIDR (например, `192.168.100.0/24`). Эти сети добавляются к маршрутам из `.bat` файлов и применяются через VPN.

5. **Протестируйте сервис**  
   Выполните команды для ручного тестирования:
   - Обновление и применение маршрутов: `/opt/bin/vpn-router update`
   - Применение маршрутов: `/opt/bin/vpn-router start`
   - Удаление маршрутов: `/opt/bin/vpn-router stop`
   - Проверка маршрутов: `ip route show table 1000`

## Проверка маршрутов

Чтобы убедиться, что маршруты (включая кастомные IP) корректно обновлены и применены, выполните следующие шаги:

1. **Проверка активных маршрутов в таблице маршрутизации**:
   Сервис добавляет маршруты в таблицу 1000. Выполните:
   ```bash
   ip route show table 1000
   ```
   Вывод покажет маршруты в формате `IP/маска dev интерфейс` (например, `192.168.100.0/24 dev nwg1`).

2. **Проверка сохраненного списка маршрутов**:
   После обновления (`/opt/bin/vpn-router update`) маршруты сохраняются в `/opt/etc/vpn-router/current_routes.txt` и применяются в таблицу 1000. Просмотрите файл:
   ```bash
   cat /opt/etc/vpn-router/current_routes.txt
   ```
   Вывод должен содержать IP-адреса из `.bat` файлов и кастомные IP в формате CIDR (например, `192.168.100.0/24`).

3. **Проверка исходных файлов маршрутов**:
   Проверьте `.bat` файлы, указанные в `/opt/etc/vpn-router/config.yaml`:
   ```bash
   cat /opt/etc/ip-address/Global/Youtube/youtube.bat
   ```
   Убедитесь, что репозиторий обновлен:
   ```bash
   cd /opt/etc/ip-address && git pull
   ```

4. **Проверка кастомных IP**:
   Убедитесь, что кастомные IP из `config.yaml` добавлены в `current_routes.txt` и таблицу 1000:
   ```bash
   grep "192.168.100.0/24" /opt/etc/vpn-router/current_routes.txt
   ip route show table 1000 | grep "192.168.100.0/24"
   ```

5. **Проверка логов**:
   Логи сервиса записываются в `/var/log/messages` и `/opt/var/log/vpn-router.log`. Просмотрите их:
   ```bash
   cat /var/log/messages | grep vpn-router
   cat /opt/var/log/vpn-router.log
   ```

6. **Ручное обновление и применение**:
   Команда `update` теперь обновляет и применяет маршруты. Для проверки выполните:
   ```bash
   /opt/bin/vpn-router update
   ip route show table 1000
   ```

## Логирование

Логи записываются в `/var/log/messages` (syslog) и `/opt/var/log/vpn-router.log`.  
Для проверки логов:
```bash
cat /opt/var/log/vpn-router.log
```

## Подробности конфигурации

Сервис использует YAML-файл конфигурации (`/opt/etc/vpn-router/config.yaml`) для указания:
- VPN-интерфейса для маршрутизации.
- Директории с клонированным репозиторием RockBlack-VPN.
- Списка файлов с маршрутами IP.
- Кастомных IP-сетей в формате CIDR.

Если указанный файл отсутствует или не может быть обработан, сервис выдает предупреждение и пропускает его, используя последний действительный список маршрутов.  
Маршруты применяются ко всем устройствам, за исключением локальной сети (192.168.0.0/16), которая защищена правилом маршрутизации.

Пример конфигурации:
```yaml
vpn_interface: "nwg1"
repo_dir: "/opt/etc/ip-address"
files:
  - "Global/Youtube/youtube.bat"
  - "Global/Instagram/instagram.bat"
ips:
  - "192.168.100.0/24"
  - "10.0.0.0/16"
```

- **vpn_interface**: Имя VPN-интерфейса (проверьте через `ifconfig` или `ip address show`).
- **repo_dir**: Путь к репозиторию RockBlack-VPN.
- **files**: Список файлов с маршрутами из репозитория.
- **ips**: Список кастомных IP-сетей в формате CIDR. Эти сети добавляются к маршрутам из `.bat` файлов и записываются в `/opt/etc/vpn-router/current_routes.txt`.

## Ежедневные обновления

Сервис обновляет списки IP и применяет их в таблицу маршрутов 1000 ежедневно в полночь через задание cron:
```bash
0 0 * * * /opt/bin/vpn-router update >> /opt/var/log/vpn-router.log 2>&1
```

### Процесс обновления:
1. Выполняет `git pull` в репозитории RockBlack-VPN.
2. Парсит указанные файлы `.bat` для получения маршрутов IP (в формате `route ADD IP MASK MASK`).
3. Преобразует маршруты в CIDR-нотацию (например, `1.2.3.0 MASK 255.255.255.0` → `1.2.3.0/24`).
4. Добавляет кастомные IP-сети из `ips` в конфигурации.
5. Сохраняет объединённый список маршрутов в `/opt/etc/vpn-router/current_routes.txt`.
6. Проверяет активность VPN-интерфейса и применяет маршруты в таблицу 1000.
7. При сбое обновления сохраняется предыдущий список маршрутов.

## Устранение неполадок

- **Потеря интернета или доступа к веб-интерфейсу**:
  - **Восстановление**:
    - Перезагрузите роутер (выключите и включите питание).
    - Очистите таблицу маршрутов:
      ```bash
      ip route flush table 1000
      /opt/bin/vpn-router stop
      ```
    - Отключите VPN-интерфейс:
      ```bash
      ifconfig nwg1 down
      ```
  - **Причина**: Некорректные маршруты в таблице 1000 или неактивный VPN-интерфейс. Убедитесь, что VPN работает:
    ```bash
    ping -I nwg1 8.8.8.8
    ```
  - Проверьте `/opt/etc/vpn-router/current_routes.txt` на наличие слишком широких диапазонов (например, `/11` или `/16`):
    ```bash
    cat /opt/etc/vpn-router/current_routes.txt
    ```
  - Проверьте кастомные IP в `config.yaml` на корректность формата CIDR:
    ```bash
    cat /opt/etc/vpn-router/config.yaml
    ```
  - Ограничьте файлы и IP в `config.yaml` для тестирования:
    ```bash
    nano /opt/etc/vpn-router/config.yaml
    ```
    Оставьте только один файл или IP, например:
    ```yaml
    vpn_interface: "nwg1"
    repo_dir: "/opt/etc/ip-address"
    files:
      - "Global/Youtube/youtube.bat"
    ips:
      - "192.168.100.0/24"
    ```
    Затем обновите и примените маршруты:
    ```bash
    /opt/bin/vpn-router update
    ```

- **Ошибка "syntax error: unexpected '('" в ifstatechanged.sh**:
  - **Причина**: Скрипт использует синтаксис, несовместимый с BusyBox `ash` (например, `$(...)`). Установочный скрипт теперь использует обратные кавычки `` `...` `` для совместимости.
  - **Решение**:
    - Обновите хук-скрипт:
      ```bash
      sed -i 's/\$(/`/g; s/)/`/g; s/\r$//' /opt/etc/vpn-router/ifstatechanged.sh
      ```
    - Проверьте выполнение:
      ```bash
      INTERFACE=nwg1 STATE=up /opt/etc/vpn-router/ifstatechanged.sh
      ```
    - Убедитесь, что файл имеет окончания строк `LF`:
      ```bash
      sed -i 's/\r$//' /opt/etc/vpn-router/ifstatechanged.sh
      ```

- **Ошибка "exit code 2" в логе ndm**:
  - Проверьте логи:
    ```bash
    cat /opt/var/log/vpn-router.log
    cat /var/log/messages | grep vpn-router
    ```
  - Выполните хук вручную для диагностики:
    ```bash
    INTERFACE=nwg1 STATE=up /opt/etc/vpn-router/ifstatechanged.sh; echo $?
    ```
  - Проверьте `/opt/bin/vpn-router update`:
    ```bash
    /opt/bin/vpn-router update; echo $?
    ```

- **VPN-интерфейс не найден**:
  - Проверьте имя интерфейса:
    ```bash
    ifconfig
    ip address show
    ```
  - Обновите `vpn_interface` в `/opt/etc/vpn-router/config.yaml`.

- **Маршруты не применяются**:
  - Проверьте логи: `cat /var/log/messages | grep vpn-router` или `cat /opt/var/log/vpn-router.log`.
  - Убедитесь, что VPN включен и хук-скрипт находится в `/opt/etc/ndm/ifstatechanged.d`.
  - Проверьте таблицу маршрутов: `ip route show table 1000`.
  - Убедитесь, что `/opt/etc/vpn-router/current_routes.txt` содержит маршруты:
    ```bash
    cat /opt/etc/vpn-router/current_routes.txt
    ```
  - Проверьте, что кастомные IP добавлены:
    ```bash
    grep "192.168.100.0/24" /opt/etc/vpn-router/current_routes.txt
    ip route show table 1000 | grep "192.168.100.0/24"
    ```
  - Обновите и примените маршруты вручную: `/opt/bin/vpn-router update`.

- **Ошибка "RTNETLINK answers: File exists"**:
  - Ошибка возникает, если маршруты уже существуют в таблице 1000. Команда `update` теперь автоматически очищает таблицу перед добавлением маршрутов. Если ошибка сохраняется:
    ```bash
    ip route flush table 1000
    /opt/bin/vpn-router update
    ```
  - Проверьте, нет ли дубликатов в `/opt/etc/vpn-router/current_routes.txt`:
    ```bash
    sort -u /opt/etc/vpn-router/current_routes.txt > /opt/etc/vpn-router/current_routes.txt.new
    mv /opt/etc/vpn-router/current_routes.txt.new /opt/etc/vpn-router/current_routes.txt
    ```

- **Отсутствуют файлы**: Убедитесь, что пути к файлам в `config.yaml` соответствуют структуре репозитория RockBlack-VPN:
  ```bash
  cat /opt/etc/ip-address/Global/Youtube/youtube.bat
  ```

- **Неверный формат кастомных IP**:
  - Убедитесь, что IP-сети в поле `ips` указаны в формате CIDR (например, `192.168.100.0/24`).
  - Проверьте логи на наличие предупреждений о неверных CIDR:
    ```bash
    cat /opt/var/log/vpn-router.log
    ```

- **Проблемы с обновлением через Git**: Проверьте доступ в интернет и наличие `ca-certificates`:
  ```bash
  cd /opt/etc/ip-address && git pull
  ```

- **Бинарник не скачивается или поврежден**: Убедитесь, что Release опубликован в репозитории. Проверьте SHA256 checksum вручную или соберите бинарник локально:
  ```bash
  GOOS=linux GOARCH=<arch> go build -o vpn-router main.go
  scp vpn-router root@<router-ip>:/opt/bin/
  ```

- **Проблемы с SHA256**: Установочный скрипт игнорирует регистр символов при проверке хешей и автоматически пропускает проверку в неинтерактивном режиме (например, `curl | sh`). Если проверка всё равно не проходит:
  - Проверьте хеши вручную, игнорируя регистр:
    ```bash
    sha256sum /opt/bin/vpn-router | cut -d" " -f1 | tr '[:upper:]' '[:lower:]' > /opt/bin/computed.sha256
    cat /opt/bin/vpn-router-<arch>.sha256 | tr '[:upper:]' '[:lower:]' > /opt/bin/expected.sha256
    cmp /opt/bin/computed.sha256 /opt/bin/expected.sha256
    ```
  - Для интерактивной установки скачайте скрипт отдельно:
    ```bash
    curl -sL https://raw.githubusercontent.com/ngenious-lab/keenetic-routes-via-vpn/main/install.sh -o install.sh
    sh install.sh
    ```
  - Убедитесь, что `coreutils-sha256sum` установлен:
    ```bash
    opkg install coreutils-sha256sum
    ```

## Удаление

Для удаления сервиса выполните:
```bash
rm -rf /opt/etc/vpn-router /opt/etc/ip-address /opt/bin/vpn-router /opt/var/log/vpn-router.log
rm -f /opt/etc/ndm/ifstatechanged.d/vpn-router.sh /opt/etc/cron.d/vpn-router
ip route flush table 1000
ip rule del from 192.168.0.0/16 lookup main prio 100 2>/dev/null
```

## Вклад в проект

Приглашаем открывать issues или отправлять pull requests для улучшения сервиса. Убедитесь, что изменения совместимы с роутерами Keenetic и Entware.

## Лицензия

Проект распространяется под лицензией MIT. Подробности см. в файле [LICENSE](LICENSE).

## Ответственность

Автор не несет ответственности за любые последствия использования данного сервиса, включая, но не ограничиваясь, сбои в работе сети, потерю данных или любые другие проблемы, связанные с его использованием. Используйте сервис на свой страх и риск.