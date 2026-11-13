<p align="center"><img src="docs/img/icon.png" width="128" alt="Perimeter"></p>

<h1 align="center">Perimeter</h1>

<p align="center"><b>Kill switch для отдельных приложений.</b><br>
Отвалился VPN, сменился IP или пропал интернет - выбранные программы замирают.<br>
Вернулся правильный IP - продолжают с того же места.</p>

---

## Что делает

Обычный kill switch в VPN-клиенте рубит интернет целиком. Perimeter трогает только те программы, которые вы выбрали (Claude, ChatGPT, Cursor, браузер для ИИ и т.д.), остальное работает как обычно.

- **Заморозка, а не убийство.** Программа ставится на паузу (`SIGSTOP` / `NtSuspendProcess`) вместе со всеми дочерними процессами. VPN моргнул на пару секунд - работа продолжается, ничего не потеряно. Если правильный IP не вернулся за заданное время, программа закрывается.
- **Белый список.** Разрешённые IP, подсети и целые страны. Для каждого приложения можно задать свои правила.
- **Надёжная проверка.** IP подтверждают минимум два независимых сервиса. Отдельно проверяются IPv6 и DNS. Смена сети ловится по системным событиям, а не только по таймеру.
- **Доверенные сети.** Дома можно без VPN, в кафе - только с ним. Сеть узнаётся по роутеру.
- **Окно в трее и панель.** Страна и IP, что защищается, история, статистика, проверка утечек.
- **Расширение для браузера.** Ставит на паузу только вкладки с ИИ и закрывает утечку через WebRTC. Работает в связке с приложением.
- **Без облака.** Никаких аккаунтов, ботов и телеметрии - см. [что уходит в сеть](#что-perimeter-отправляет-в-сеть).

macOS 11+, Windows 10/11, Linux. Права администратора не нужны.

## Как это работает

1. Раз в несколько секунд и сразу после смены сети Perimeter спрашивает свой внешний IP у нескольких сервисов и определяет страну.
2. Если IP разрешён - всё в порядке, значок зелёный.
3. Если нет - выбранные программы замораживаются, значок оранжевый, идёт обратный отсчёт.
4. Вернулся разрешённый IP - программы размораживаются. Не вернулся до конца отсчёта - закрываются, значок красный.

При первом запуске текущий IP становится разрешённым, поэтому запускайте Perimeter с включённым VPN.

## Установка

Готовые сборки - на странице [релизов](https://github.com/LuckyKort/perimeter/releases/latest). Или одной строкой.

macOS и Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/LuckyKort/perimeter/main/scripts/install.sh | sh
```

Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/LuckyKort/perimeter/main/scripts/install.ps1 | iex
```

## Сборка

Нужен Go версии из `go.mod`. На macOS окно панели использует WebKit через cgo - хватит Xcode Command Line Tools. Windows и Linux собираются без cgo.

```bash
go build -o perimeter ./cmd/perimeter
./perimeter
```

Готовое приложение для macOS (`dist/Perimeter.app`, Apple Silicon + Intel):

```bash
./scripts/build-mac.sh
```

Windows (кросс-компиляция с Mac или Linux): `./scripts/build-windows.sh`. Расширение: `./scripts/build-extension.sh`.

Проверки:

```bash
go vet ./... && go test ./...
```

### Обновления

Perimeter обновляется из [релизов LuckyKort/perimeter](https://github.com/LuckyKort/perimeter/releases): несколько раз в день проверяет новую версию, показывает в окне, что в ней нового, и ставит по кнопке, сверив контрольную сумму. Отключается в настройках («Проверять обновления»).

Выпустить версию: добавьте в `CHANGELOG.md` раздел `## X.Y.Z - дата` и поставьте тег.

```bash
git tag vX.Y.Z && git push origin vX.Y.Z
```

Релизный workflow соберёт приложения, расширение и контрольные суммы и опубликует релиз; описание берётся из этого раздела CHANGELOG. Сборки с версией не вида `X.Y.Z` (например `0.1.1-dev`) не обновляются.

Форк с другим репозиторием передаёт свой при сборке (`REPO=owner/name ./scripts/build-mac.sh`), `REPO=` без значения отключает обновления и ссылки на GitHub. Адрес по умолчанию - в `internal/app/paths.go`.

Иконки рисуются кодом (`internal/ui/icon.go`). После изменения логотипа перегенерируйте PNG:

```bash
go run ./tools/mkicons -assets
```

## Команды

```
perimeter                 запуск в строке меню / трее
perimeter status          текущий IP, страна и состояние
perimeter allow           разрешить текущий IP
perimeter allow 1.2.3.4   разрешить IP или подсеть (1.2.3.0/24)
perimeter allow NL        разрешить страну
perimeter config          путь к файлу настроек
perimeter autostart on|off
perimeter panel           открыть панель
perimeter headless        работать без значка в трее
perimeter version
```

## Настройки

Всё настраивается в панели. Те же настройки лежат в `config.yaml` с комментариями, изменения подхватываются без перезапуска:

| Система | Папка |
|---|---|
| macOS | `~/Library/Application Support/Perimeter` |
| Windows | `%AppData%\Perimeter` |
| Linux | `~/.config/perimeter` |

Отдельную папку для экспериментов можно задать переменной `PERIMETER_DIR`.

## Что Perimeter отправляет в сеть

Только запросы, без которых не узнать свой IP и страну:

- IPv4: `api.ipify.org`, `ipv4.icanhazip.com`, `v4.ident.me`, `checkip.amazonaws.com`, `ipinfo.io` (список меняется в настройках);
- IPv6: `api6.ipify.org`, `ipv6.icanhazip.com`, `v6.ident.me`;
- страна: `ipinfo.io`, `ipapi.co`, `api.country.is`;
- проверка обновлений на GitHub - только если сборка привязана к репозиторию.

Браузерное расширение общается только с приложением на `127.0.0.1:47843`.

## Лицензия

[GPL-3.0](LICENSE). Perimeter - форк [Kordon](https://github.com/ef1m03/kordon) © Efimov Dev, сделанный из последнего открытого кода (коммит `e01c7c4`, 03.11.2026). Perimeter не связан с Efimov Dev. Подробности и список изменений - в [NOTICE](NOTICE) и [CHANGELOG](CHANGELOG.md).
