# TSM - telegram server manager

## Оглавление

- [Описание](#описание)
- [Установка](#установка)
- [Команды](#команды)
- [Контакты](#контакты)

## Описание
Не большой и очень легковесный скрипт, который позволяет управлять сервером через telegram.

В commands.json можно настраивать команды, которые будут выполняться на сервере.

Пример:

```json
[
    {
        "command" : "reboot",
        "description": "Перезагрузка сервера",
        "script" : "reboot.sh"
    },
    {
        "command" : "shutdown",
        "description": "Выключение сервера",
        "script" : "shutdown.sh"
    }
]
```
Поле script указывает путь к скрипту, который будет выполняться на сервере.

## Установка

```bash
git clone https://github.com/iyan11/telegram-server-manager.git
cd telegram-server-manager
go build -o tsm
./tsm
```

## Команды

- `/start`: Стартовое сообщение
- `/help`: Помощь

## Контакты

- Telegram: [@kivpa](https://t.me/kivpa)
- GitHub: [iyan11](https://github.com/iyan11)
- Email: [serverscrazy@gmail.com](mailto:serverscrazy@gmail.com)