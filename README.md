# vpn-router

Утилита для управления маршрутами через отдельную таблицу маршрутизации (1000) и использования VPN-интерфейса.

## Возможности

* Чтение маршрутов из `.bat` файлов в локальном репозитории с адресами (`repo_dir`).
* Добавление собственных подсетей в формате CIDR.
* Очистка и обновление таблицы маршрутов `1000`.
* Управление правилом маршрутизации с приоритетом `1995`.
* Сохранение актуальных маршрутов в `/opt/etc/vpn-router/current_routes.txt`.

## Установка

```bash
curl -s https://raw.githubusercontent.com/your-repo/vpn-router/main/install.sh | sh
```

Скрипт установки:

* скачивает бинарник `vpn-router` в `/opt/bin/`,
* создаёт директорию `/opt/etc/vpn-router/`,
* копирует `config.yaml`,
* клонирует сторонний репозиторий с адресами:
  [`RockBlack-VPN/ip-address`](https://github.com/RockBlack-VPN/ip-address) в `/opt/etc/ip-address`.

## Конфигурация

Файл: `/opt/etc/vpn-router/config.yaml`

```yaml
vpn_interface: tun0
repo_dir: /opt/etc/ip-address
files:
  - RU.bat
  - UA.bat
ips:
  - 10.0.0.0/24
  - 192.168.100.0/24
```

* **vpn_interface** — интерфейс VPN (например, `tun0`).
* **repo_dir** — путь к локальному репозиторию `ip-address`.
* **files** — список файлов с маршрутами.
* **ips** — список кастомных сетей в CIDR.

## Использование

```bash
vpn-router [update|start|stop|status|restart|update-repo]
```

* **update** — перечитать маршруты из файлов/конфига и записать их в `current_routes.txt`, затем применить в таблицу `1000` (если VPN-интерфейс активен).
* **start** — применить маршруты из `current_routes.txt` в таблицу `1000`.
* **stop** — очистить таблицу маршрутов `1000` и удалить правило `prio 1995`.
* **status** — показать состояние таблицы маршрутов и правила (`ip rule show | grep 1995`, `ip route show table 1000`).
* **restart** — выполнить `stop`, затем `start`.
* **update-repo** — выполнить `git pull` в каталоге `/opt/etc/ip-address`, затем автоматически запустить `vpn-router update`.

## Автообновления

Маршруты в репозитории `ip-address` обновляются самостоятельно с помощью cron-задания (например, раз в день выполняется `git pull` в `/opt/etc/ip-address`).

Пример cron-записи:

```
0 3 * * * cd /opt/etc/ip-address && git pull
0 4 * * * /opt/bin/vpn-router update
```

---

### Примечания

* Сам бинарник **не выполняет `git clone` и `git pull`**. Репозиторий с адресами клонируется один раз при установке, а обновляется через cron или через `vpn-router update-repo`.
* Для ручного обновления можно использовать:

  ```bash
  vpn-router update-repo
  ```
