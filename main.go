package main

import (
	"flag"
	"log"
	files "TelegramBot/storage/Files"
	tgClient "TelegramBot/clients/telegram"
	"TelegramBot/consumer/event-consumer"
	"TelegramBot/events/Telegram"
)

const (
	tgBotHost   = "api.telegram.org"
	storagePath = "files_storage"
	botchSize   = 100
)

func main() {

	eventsProcessor := telegram.New(
		tgClient.New(tgBotHost, mustToken()),
		files.New(storagePath),
	)

	log.Print("service started")

	consumer := event_consumer.New(eventsProcessor, eventsProcessor, botchSize)

	if err := consumer.Start(); err != nil {
		log.Fatal("service stopped ", err)
	}

}

func mustToken() string {
	token := flag.String(
		"tg-bot-token",
		"",
		"token for access telegram bot",
	)

	flag.Parse()
	if *token == "" {
		log.Fatal("token is not specified")
	}

	return *token
}


//Start for

//	first:     go build
//   second:  ./TelegramBot -tg-bot-token '7439680908:AAEHRuKIaQpPSPnOPKA58hs4X5r1scTBEX8'