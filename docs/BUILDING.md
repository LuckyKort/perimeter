# Сборка и релизы

← [Назад к README](../README.md)

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

Отдельную папку с настройками для экспериментов можно задать переменной `PERIMETER_DIR` - так dev-сборка не тронет рабочий конфиг.

## Обновления и релизы

Perimeter обновляется из [релизов LuckyKort/perimeter](https://github.com/LuckyKort/perimeter/releases): несколько раз в день проверяет новую версию, показывает в окне, что в ней нового, и ставит по кнопке, сверив контрольную сумму. Отключается в настройках («Проверять обновления»).

Выпустить версию: добавьте в `CHANGELOG.md` раздел `## X.Y.Z - дата` и поставьте тег.

```bash
git tag vX.Y.Z && git push origin vX.Y.Z
```

Релизный workflow соберёт приложения, расширение и контрольные суммы и опубликует релиз; описание берётся из этого раздела CHANGELOG. Сборки с версией не вида `X.Y.Z` (например `0.1.1-dev`) не обновляются.

Форк с другим репозиторием передаёт свой при сборке (`REPO=owner/name ./scripts/build-mac.sh`), `REPO=` без значения отключает обновления и ссылки на GitHub. Адрес по умолчанию - в `internal/app/paths.go`.

## Иконки

Иконки рисуются кодом (`internal/ui/icon.go`). После изменения логотипа перегенерируйте PNG:

```bash
go run ./tools/mkicons -assets
```

## Скриншоты

Скриншоты в `docs/img` сняты с живой панели в headless Chrome (светлая и тёмная тема, @2x) и обрамлены в окно macOS: светофор, скругления 26 pt (попап - 18 pt), тень, прозрачный фон вокруг. IP, DNS и адреса роутера перед съёмкой заменены случайными цифрами и размыты прямо в DOM. Если переснимаете - не выкладывайте свои настоящие адреса.
