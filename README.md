# go-data-keeper 
**go-data-keeper** is a client-server system that allows users to securely store usernames, passwords, binary data, and other private information.

2) Кроссплатформенность CLI (Windows/Linux/macOS)
Сейчас в cmd/client/cli/root.go дефолтные пути жёстко Windows-специфичные (D:\prog\data\...). Это ломает требование “клиент запускается на Windows/Linux/Mac”.
Что нужно сделать:
Дефолтные пути через os.UserConfigDir() / os.UserHomeDir() + подкаталог (gophkeeper/state.json, gophkeeper/vault.json).
Оставить флаги --state/--vault, но дефолты сделать переносимыми.
3) Версия и дата сборки клиента
В CLI сейчас нет команды/флага version (в коде клиента я не нашёл переменных Version/BuildDate и т.п.). По ТЗ пользователь должен уметь получить “версию и дату сборки бинарника”.
Что нужно сделать:
Команду version (или --version).
Переменные, заполняемые через -ldflags (Version, Commit, BuildDate).