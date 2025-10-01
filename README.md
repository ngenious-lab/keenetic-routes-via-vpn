# Keenetic Routes via VPN

Этот проект предоставляет сервис на Go для маршрутизации определенного сетевого трафика через VPN-туннель на роутерах Keenetic с использованием [Entware](https://entware.net/). Сервис клонирует и обрабатывает списки IP-адресов из репозитория [RockBlack-VPN/ip-address](https://github.com/RockBlack-VPN/ip-address), позволяя направлять трафик для таких сервисов, как YouTube, Instagram, Twitter или Netflix, через указанный VPN-интерфейс, в то время как остальной трафик идет через провайдера. Сервис легковесный, настраиваемый и разработан для бесшовной интеграции с Entware на Keenetic.

## Возможности

- **Выборочная маршрутизация**: Направляет трафик для указанных пользователем списков IP через VPN-интерфейс, остальной трафик идет через провайдера.
- **Ежедневное обновление**: Автоматически обновляет списки IP из репозитория RockBlack-VPN раз в день через `git pull`.
- **Обработка ошибок**: При сбое обновления или отсутствии файла использует предыдущий список маршрутов.
- **Гибкая конфигурация**: Использует простой YAML-файл для указания VPN-интерфейса и файлов со списками IP.
- **Интеграция с VPN**: Автоматически применяет/удаляет маршруты при включении/выключении VPN.
- **Логирование**: Вывод в syslog (`/var/log/messages`) с возможностью записи в файл.

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

После скачивания выполните `chmod +x /opt/bin/vpn-router`. Установочный скрипт (`install.sh`) автоматически скачивает бинарник и проверяет его целостность с помощью SHA256 checksum. Для ручной проверки:
```bash
sha256sum /opt/bin/vpn-router | cut -d" " -f1 > /opt/bin/computed.sha256
cmp /opt/bin/computed.sha256 /opt/bin/vpn-router-<arch>.sha256
```

**Примечание**: Замените `latest` на конкретную версию (например, `v1.0.0`), если требуется. Для автоматического обновления бинарников используйте GitHub API или инструмент вроде `eget` (см. [zyedidia/eget](https://github.com/zyedidia/eget)).

## Установка

Следуйте этим шагам для установки и настройки сервиса на роутере Keenetic с Entware:

1. **Убедитесь, что Entware установлен**  
   Проверьте, что Entware настроен на вашем роутере Keenetic. Инструкции по установке см. в [документации Keenetic по Entware](https://help.keenetic.com/hc/en-us/articles/360000374559-Entware).

2. **Запустите установочный скрипт**  
   Скачайте и выполните скрипт для установки сервиса:
   ```bash
   curl -sfL https://raw.githubusercontent.com/ngenious-lab/keenetic-routes-via-vpn/main/install.sh | sh
   ```
   Скрипт:
   - Устанавливает зависимости (`git`, `git-http`, `ca-bundle`, `ca-certificates`, `curl`).
   - Скачивает подходящий бинарник из GitHub Releases (на основе архитектуры роутера) и проверяет его SHA256 checksum.
   - Клонирует репозиторий [RockBlack-VPN/ip-address](https://github.com/RockBlack-VPN/ip-address) в `/opt/etc/ip-address`.
   - Клонирует и настраивает сервис (конфиг, хук для VPN-интерфейса и задание cron для ежедневных обновлений).

3. **Настройте сервис**  
   Отредактируйте конфигурационный файл `/opt/etc/vpn-router/config.yaml`:
   ```yaml
   vpn_interface: "ovpn_br0"  # Укажите ваш VPN-интерфейс (например, nwg0 для WireGuard, проверьте через `ifconfig` или `ip address show`)
   repo_dir: "/opt/etc/ip-address"  # Путь к клонированному репозиторию RockBlack-VPN/ip-address
   files:
     - "Global/Youtube/youtube.bat"  # Пример: путь к файлу в репозитории RockBlack-VPN
     - "Global/Instagram/instagram.bat"
     - "Global/Twitter/twitter.bat"
   ```
   - **vpn_interface**: Имя вашего VPN-интерфейса (например, `ovpn_br0` для OpenVPN, `nwg0` для WireGuard).
   - **repo_dir**: Директория, куда клонирован репозиторий [RockBlack-VPN/ip-address](https://github.com/RockBlack-VPN/ip-address) (по умолчанию `/opt/etc/ip-address`).
   - **files**: Список файлов (`.bat` или аналогичных) из репозитория [RockBlack-VPN/ip-address](https://github.com/RockBlack-VPN/ip-address) с маршрутами IP (в формате `route ADD IP MASK MASK`). Указывайте пути относительно `repo_dir`.

4. **Запустите VPN-соединение**  
   Включите VPN-соединение через веб-интерфейс Keenetic или CLI. Сервис автоматически отслеживает изменение состояния VPN (вкл/выкл) и применяет/удаляет маршруты с помощью хука в `/opt/etc/ndm/ifstatechanged.d`.

5. **Протестируйте сервис**  
   Выполните команды для ручного тестирования:
   - Обновление маршрутов: `/opt/bin/vpn-router update`
   - Применение маршрутов: `/opt/bin/vpn-router start`
   - Удаление маршрутов: `/opt/bin/vpn-router stop`

## Логирование

Логи записываются в `/var/log/messages` (syslog).  
Для записи в файл добавьте перенаправление в хук-скрипт (`/opt/etc/vpn-router/ifstatechanged.sh`):
```bash
/opt/bin/vpn-router start >> /opt/var/log/vpn-router.log 2>&1
```

## Подробности конфигурации

Сервис использует YAML-файл конфигурации (`/opt/etc/vpn-router/config.yaml`) для указания:
- VPN-интерфейса для маршрутизации.
- Директории с клонированным репозиторием RockBlack-VPN.
- Списка файлов с маршрутами IP.

Если указанный файл отсутствует или не может быть обработан, сервис выдает предупреждение и пропускает его, используя последний действительный список маршрутов.  
Маршруты применяются ко всем устройствам, включая сам роутер, с использованием policy routing (таблица 1000).

Пример конфигурации для маршрутизации трафика YouTube и Instagram:
```yaml
vpn_interface: "nwg0"
repo_dir: "/opt/etc/ip-address"
files:
  - "Global/Youtube/youtube.bat"
  - "Global/Instagram/instagram.bat"
```

## Ежедневные обновления

Сервис обновляет списки IP ежедневно в полночь через задание cron:
```bash
0 0 * * * /opt/bin/vpn-router update
```

### Процесс обновления:
1. Выполняет `git pull` в репозитории RockBlack-VPN.
2. Парсит указанные файлы `.bat` для получения маршрутов IP (в формате `route ADD IP MASK MASK`).
3. Преобразует маршруты в CIDR-нотацию (например, `1.2.3.0 MASK 255.255.255.0` → `1.2.3.0/24`).
4. Сохраняет новые маршруты в `/opt/etc/vpn-router/current_routes.txt`.
5. При сбое обновления сохраняется предыдущий список маршрутов.

## Устранение неполадок

- **VPN-интерфейс не найден**: Проверьте имя интерфейса с помощью `ifconfig` или `ip address show` и обновите `vpn_interface` в `config.yaml`.
- **Маршруты не применяются**: Проверьте логи в `/var/log/messages` на наличие ошибок. Убедитесь, что VPN включен и хук-скрипт находится в `/opt/etc/ndm/ifstatechanged.d`.
- **Отсутствуют файлы**: Убедитесь, что пути к файлам в `config.yaml` соответствуют структуре репозитория RockBlack-VPN.
- **Проблемы с обновлением через Git**: Проверьте доступ в интернет и наличие `ca-certificates` на роутере.
- **Бинарник не скачивается или поврежден**: Убедитесь, что Release опубликован в репозитории. Проверьте SHA256 checksum вручную или соберите бинарник локально.
- **Проблемы с SHA256**: Если проверка целостности не проходит, убедитесь, что бинарник и checksum-файл из одного релиза. Скачайте заново или соберите вручную.

## Удаление

Для удаления сервиса выполните:
```bash
rm -rf /opt/etc/vpn-router /opt/etc/ip-address /opt/bin/vpn-router
rm -f /opt/etc/ndm/ifstatechanged.d/vpn-router.sh /opt/etc/cron.d/vpn-router
```

## Вклад в проект

Приглашаем открывать issues или отправлять pull requests для улучшения сервиса. Убедитесь, что изменения совместимы с роутерами Keenetic и Entware.

## Лицензия

Проект распространяется под лицензией MIT. Подробности см. в файле [LICENSE](LICENSE).

## Ответственность

Автор не несет ответственности за любые последствия использования данного сервиса, включая, но не ограничиваясь, сбои в работе сети, потерю данных или любые другие проблемы, связанные с его использованием. Используйте сервис на свой страх и риск.