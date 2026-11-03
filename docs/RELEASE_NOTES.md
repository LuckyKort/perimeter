**Установка и обновление**

Уже стоит Kordon 1.1 или новее - обновитесь из меню (пункт «Обновить до …») или командой `kordon update`. Первая установка:

macOS и Linux:
```bash
curl -fsSL https://raw.githubusercontent.com/ef1m03/kordon/main/scripts/install.sh | sh
```

Windows (PowerShell):
```powershell
irm https://raw.githubusercontent.com/ef1m03/kordon/main/scripts/install.ps1 | iex
```

Подробный гайд: [README](https://github.com/ef1m03/kordon#readme). Файлы ниже - для ручной установки.

| Файл | Для чего |
|---|---|
| `Kordon-macos.zip` | macOS 11+, Apple Silicon и Intel |
| `kordon-windows-amd64.exe` | Windows 10/11 x64 |
| `kordon-windows-arm64.exe` | Windows на ARM |
| `kordon-linux-*.tar.gz` | Linux x64 / ARM64 |
| `SHA256SUMS.txt` | контрольные суммы |

Сделано в [Efimov Dev](https://efimovdev.ru/kordon/?utm_source=kordon&utm_medium=release).
