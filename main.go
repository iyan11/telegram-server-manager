package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"golang.org/x/text/encoding/charmap"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"unicode/utf8"
)

type Command struct {
	Command     string `json:"command"`
	Description string `json:"description"`
	Script      string `json:"script"`
}

func main() {
	// Загружаем переменные из файла .env
	if err := godotenv.Load(); err != nil {
		log.Panic("Error loading .env file")
	}

	// Получаем токен и разрешенный ID пользователя из .env
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Panic("TELEGRAM_BOT_TOKEN is not set in .env file")
	}

	// Получаем токен и разрешенный ID пользователя из .env
	serverName := os.Getenv("SERVER_NAME")
	if serverName == "" {
		log.Panic("SERVER_NAME is not set in .env file")
	}

	allowedUserIDStr := os.Getenv("ALLOWED_TELEGRAM_USER_ID")
	if allowedUserIDStr == "" {
		log.Panic("ALLOWED_TELEGRAM_USER_ID is not set in .env file")
	}

	otherCommands := os.Getenv("OTHER_COMMANDS")
	if otherCommands == "" {
		log.Panic("OTHER_COMMANDS is not set in .env file")
	}

	allowedUserID, err := strconv.Atoi(allowedUserIDStr)
	if err != nil {
		log.Panic("Invalid ALLOWED_TELEGRAM_USER_ID in .env file: must be an integer")
	}

	// Создаем бота
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Загружаем команды из файла commands.json
	commands, err := loadCommands("commands/commands.json")
	if err != nil {
		log.Panicf("Failed to load commands: %v", err)
	}

	// Устанавливаем подсказки для команд
	err = setBotCommands(bot, commands)
	if err != nil {
		log.Printf("Failed to set bot commands: %v", err)
	}

	// Получаем обновления от Telegram
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := bot.GetUpdatesChan(updateConfig)

	for update := range updates {
		if update.Message == nil { // Игнорируем любые обновления, которые не являются сообщениями
			continue
		}

		// Проверяем, что сообщение отправлено разрешенным пользователем
		if update.Message.From.ID != int64(allowedUserID) {
			log.Printf("Unauthorized user: %d", update.Message.From.ID)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Access denied.")
			_, err := bot.Send(msg)
			if err != nil {
				log.Printf("Error sending message: %v", err)
			}
			continue
		}

		log.Printf("[Received] From: %d, Command: %s", update.Message.From.ID, update.Message.Text)

		switch update.Message.Text {
		case "/start":
			sendMessage(bot, update.Message.Chat.ID, "Добро пожаловать! Это бот управления сервером "+serverName+".\n Разработано @kivpa \n Используйте команды, чтобы начать.", false)
			sendHelpMessage(bot, update.Message.Chat.ID, commands)
		case "/help":
			sendHelpMessage(bot, update.Message.Chat.ID, commands)
		default:
			// Если команда не найдена в JSON, выполняем как shell-команду
			if !handleCustomCommand(bot, update.Message.Chat.ID, update.Message.Text, commands) {
				if otherCommands == "yes" {
					output := executeCommand(update.Message.Text)
					if output != nil {
						sendMessage(bot, update.Message.Chat.ID, string(output), false)
					}
				}
			}
		}
	}
}

func loadCommands(filePath string) ([]Command, error) {
	var commands []Command
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Printf("Error closing file: %v", err)
		}
	}(file)

	if err := json.NewDecoder(file).Decode(&commands); err != nil {
		return nil, err
	}
	return commands, nil
}

func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string, markdown bool) {
	// Проверяем, что строка валидна в UTF-8
	if !utf8.ValidString(text) {
		text = string([]rune(text)) // Преобразуем строку в валидный формат
	}
	msg := tgbotapi.NewMessage(chatID, text)
	if markdown {
		msg.ParseMode = "Markdown"
	}
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func sendHelpMessage(bot *tgbotapi.BotAPI, chatID int64, commands []Command) {
	helpText := "Доступные команды:\n"
	helpText += "- `/start`: Стартовое сообщение\n"
	helpText += "- `/help`: Помощь\n"
	for _, cmd := range commands {
		helpText += "- `/" + cmd.Command + "`: " + cmd.Description + "\n"
	}
	sendMessage(bot, chatID, helpText, true)
}

func handleCustomCommand(bot *tgbotapi.BotAPI, chatID int64, commandText string, commands []Command) bool {
	for _, cmd := range commands {
		if commandText == "/"+cmd.Command {
			scriptPath := filepath.Join("commands", cmd.Script)
			output := executeCommand(scriptPath)
			if output == nil {
				sendMessage(bot, chatID, "Ошибка выполнения команды.", false)
			} else {
				sendMessage(bot, chatID, string(output), false)
			}
			return true
		}
	}
	return false // Команда не найдена в JSON
}

func executeCommand(command string) []byte {
	var cmd *exec.Cmd

	// Проверяем операционную систему
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command) // Для Windows
	} else {
		cmd = exec.Command("bash", "-c", command) // Для Unix-подобных систем
	}

	// Выполняем команду и получаем результат
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error executing command: %v", err)
		return []byte(fmt.Sprintf("Error: %s", err.Error()))
	}

	// Конвертируем вывод в UTF-8, если он в другой кодировке
	if !utf8.Valid(output) {
		decodedOutput, _ := charmap.Windows1251.NewDecoder().Bytes(output)
		return decodedOutput
	}
	return output
}

func setBotCommands(bot *tgbotapi.BotAPI, commands []Command) error {
	var botCommands []tgbotapi.BotCommand

	// Добавляем команды /start и /help в начало
	botCommands = append(botCommands, tgbotapi.BotCommand{
		Command:     "start",
		Description: "Стартовое сообщение",
	})
	botCommands = append(botCommands, tgbotapi.BotCommand{
		Command:     "help",
		Description: "Помощь",
	})

	// Добавляем остальные команды из файла commands.json
	for _, cmd := range commands {
		botCommands = append(botCommands, tgbotapi.BotCommand{
			Command:     cmd.Command,
			Description: cmd.Description,
		})
	}

	// Формируем запрос для установки команд
	config := tgbotapi.NewSetMyCommands(botCommands...)
	_, err := bot.Request(config)
	return err
}
