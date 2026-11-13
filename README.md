<p align="center"><img src="docs/img/icon.png" width="128" alt="Perimeter"></p>

<h1 align="center">Perimeter</h1>

<p align="center"><b>Kill switch для отдельных приложений.</b><br>
VPN отвалился - Claude, ChatGPT и Cursor замирают на месте.<br>
VPN вернулся - продолжают с той же строчки. Остальной интернет даже не заметит.</p>

<p align="center">
  <a href="https://github.com/LuckyKort/perimeter/releases/latest"><b>Скачать</b></a> ·
  <a href="#установка">Установка</a> ·
  <a href="#как-это-работает">Как это работает</a> ·
  <a href="#приватность">Приватность</a>
</p>

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/img/overview-dark.png">
    <img src="docs/img/overview-light.png" width="820" alt="Панель Perimeter: защита включена, внешний IP и DNS в разрешённой стране">
  </picture>
</p>

---

## Зачем

Обычный kill switch в VPN-клиенте рубит интернет целиком: моргнул туннель - отвалились созвон, загрузки и музыка. А сервисам вроде Claude или ChatGPT достаточно одного запроса с «не того» IP, чтобы прилетела блокировка аккаунта.

Perimeter решает ровно эту задачу: следит за вашим внешним IP и **ставит на паузу только выбранные программы**, когда адрес становится не тем. Всё остальное работает как обычно.

## Возможности

<table>
<tr>
<td width="55%">

**❄️ Заморозка, а не убийство**<br>
Программа ставится на паузу вместе со всеми дочерними процессами. VPN моргнул на пару секунд - работа продолжается, несохранённое не теряется. Не вернулся правильный IP за заданное время - программу можно закрыть.

**🎯 Точечно**<br>
Выберите, что защищать: Claude, ChatGPT, Cursor, Codex, Gemini CLI, Copilot, Perplexity, Slack, Telegram - или любую свою программу. Для каждой можно задать свои правила.

**🌍 Белый список**<br>
Разрешайте отдельные IP, подсети или целые страны.

</td>
<td>
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/img/popover-dark.png">
    <img src="docs/img/popover-light.png" width="380" alt="Окно в строке меню">
  </picture>
</td>
</tr>
</table>

**🔍 Не верит на слово.** IP подтверждают минимум два независимых сервиса. Отдельно проверяются IPv6 и DNS - ловятся VPN, которые заворачивают только IPv4. Смена сети замечается мгновенно по системным событиям, а не только по таймеру.

**🏠 Доверенные сети.** Дома можно без VPN, в кафе - только с ним. Сеть узнаётся по роутеру.

**🧩 Расширение для браузера.** Ставит на паузу только вкладки с ИИ и закрывает утечку настоящего IP через WebRTC.

**🔒 Без облака.** Никаких аккаунтов, серверов, ботов и телеметрии. Права администратора не нужны.

<table>
<tr>
<td width="50%">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/img/apps-dark.png">
    <img src="docs/img/apps-light.png" alt="Выбор защищаемых приложений">
  </picture>
  <p align="center"><sub>Выбираете, какие программы защищать</sub></p>
</td>
<td width="50%">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/img/leaks-dark.png">
    <img src="docs/img/leaks-light.png" alt="Проверка утечек IP, IPv6 и DNS">
  </picture>
  <p align="center"><sub>Проверка утечек: IP, IPv6, DNS, прокси</sub></p>
</td>
</tr>
</table>

## Как это работает

| | Состояние | Что происходит |
|---|---|---|
| 🟢 | **Всё в порядке** | Раз в несколько секунд и сразу после смены сети Perimeter сверяет внешний IP у нескольких сервисов. Адрес разрешён - программы работают. |
| 🟠 | **Заморожено** | IP сменился, VPN отвалился или пропал интернет - выбранные программы мгновенно встают на паузу. Идёт обратный отсчёт. |
| 🟢 | **Вернулось** | Разрешённый IP снова на месте - программы размораживаются и продолжают с того же места. |
| 🔴 | **Закрыто** | Если включено закрытие и IP так и не вернулся - программы закрываются. |

> [!TIP]
> При первом запуске текущий IP становится разрешённым - поэтому запускайте Perimeter с включённым VPN.

## Установка

**macOS 11+, Windows 10/11, Linux.** Готовые сборки - на странице [релизов](https://github.com/LuckyKort/perimeter/releases/latest). Или одной строкой:

macOS и Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/LuckyKort/perimeter/main/scripts/install.sh | sh
```

Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/LuckyKort/perimeter/main/scripts/install.ps1 | iex
```

Perimeter сам следит за обновлениями: покажет, что нового, и обновится по кнопке, сверив контрольную сумму.

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/img/settings-dark.png">
    <img src="docs/img/settings-light.png" width="720" alt="Настройки Perimeter">
  </picture>
</p>

## Приватность

Perimeter отправляет в сеть только запросы, без которых не узнать свой IP и страну:

- IPv4: `api.ipify.org`, `ipv4.icanhazip.com`, `v4.ident.me`, `checkip.amazonaws.com`, `ipinfo.io` (список меняется в настройках);
- IPv6: `api6.ipify.org`, `ipv6.icanhazip.com`, `v6.ident.me`;
- страна: `ipinfo.io`, `ipapi.co`, `api.country.is`;
- проверка обновлений на GitHub.

Браузерное расширение общается только с приложением на `127.0.0.1:47843`.

<details>
<summary><b>Командная строка</b></summary>

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

</details>

<details>
<summary><b>Где лежат настройки</b></summary>

Всё настраивается в панели. Те же настройки лежат в `config.yaml` с комментариями, изменения подхватываются без перезапуска:

| Система | Папка |
|---|---|
| macOS | `~/Library/Application Support/Perimeter` |
| Windows | `%AppData%\Perimeter` |
| Linux | `~/.config/perimeter` |

</details>

<details>
<summary><b>Сборка из исходников</b></summary>

Сборка, тесты, выпуск релизов и генерация иконок описаны в [docs/BUILDING.md](docs/BUILDING.md).

</details>

## Лицензия

[GPL-3.0](LICENSE). Perimeter - форк [Kordon](https://github.com/ef1m03/kordon) © Efimov Dev, сделанный из последнего открытого кода (коммит `e01c7c4`, 03.11.2026). Perimeter не связан с Efimov Dev. Подробности и список изменений - в [NOTICE](NOTICE) и [CHANGELOG](CHANGELOG.md).
