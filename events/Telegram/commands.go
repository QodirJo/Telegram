package telegram

import (
	"TelegramBot/events"
	"TelegramBot/lib/e"
	storage "TelegramBot/storage"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/tealeg/xlsx"
)

const (
	RunCmd   = "/rnd"
	HelpCmd  = "/help"
	StartCmd = "/start"
)

// A map to keep track of user states
var userStates = make(map[int]bool)

func (p *Processor) doCmd(text string, chatID int, username string) error {
	text = strings.TrimSpace(text)
	log.Printf("got new command '%s' from user %s", text, username)

	switch text {
	case StartCmd:
		userStates[chatID] = true // Set the user state to waiting for STIR number
		return p.requestSTIRNumber(chatID)
	case RunCmd:
		return p.SendRandom(chatID, username)
	case HelpCmd:
		return p.SendHelp(chatID)
	default:
		return p.tg.SendMessage(chatID, msgUnknownCommand)
	}
}

func (p *Processor) requestSTIRNumber(chatID int) error {
	// Request STIR number from user
	return p.tg.SendMessage(chatID, "Please enter your STIR number (9-digit number):")
}

func (p *Processor) handleSTIRNumber(stir string, chatID int) error {
	// Validate if the input is a 9-digit number
	if matched, _ := regexp.MatchString(`^\d{9}$`, stir); !matched {
		return p.tg.SendMessage(chatID, "Malumot xato kiritilgan.")
	}

	filePath := "/home/ubuntu/Telegram/ex.xlsm" // Update this path
	file, err := xlsx.OpenFile(filePath)
	if err != nil {
		return p.tg.SendMessage(chatID, "Error opening the file.")
	}

	sheet := file.Sheets[0]
	found := false
	for _, row := range sheet.Rows {
		cell := row.Cells[0]
		if cell.String() == stir {
			found = true

			// Extract data from cells
			orgName := row.Cells[1].String()  // Adjust index based on your file's column structure
			oked := row.Cells[2].String()     // Adjust index based on your file's column structure
			okedName := row.Cells[3].String() // Adjust index based on your file's column structure
			region := row.Cells[4].String()   // Adjust index based on your file's column structure
			district := row.Cells[5].String() // Adjust index based on your file's column structure

			response := fmt.Sprintf(
				"СТИР: %s\nТашкилотингиз номи: %s\nЖойлашган худудингиз: %s\nМанзилингиз: %s\nОКЭД рақами: %s\nОКЭД номи: %s",
				stir, orgName, region, district, oked, okedName,
			)

			userStates[chatID] = false // Reset the user state
			return p.tg.SendMessage(chatID, response)
		}
	}

	if !found {
		return p.tg.SendMessage(chatID, "STIR number not found.")
	}
	return nil
}

func (p *Processor) ProcessMessage(event events.Event) error {
	meta, err := meta(event)
	if err != nil {
		return e.Wrap("can't process message", err)
	}

	if userStates[meta.ChatId] {
		// If waiting for STIR number, handle it
		return p.handleSTIRNumber(event.Text, meta.ChatId)
	}

	if event.Text == StartCmd {
		// If the command is /start, handle it accordingly
		return p.doCmd(event.Text, meta.ChatId, meta.UserName)
	}

	// If not a known command, treat it as STIR number
	return p.handleSTIRNumber(event.Text, meta.ChatId)
}

func (p *Processor) SendHelp(chatID int) error {
	return p.tg.SendMessage(chatID, msgHelp)
}

func (p *Processor) SendHello(chatID int) error {
	return p.tg.SendMessage(chatID, msgHello)
}

func (p *Processor) SendRandom(chatID int, username string) (err error) {
	defer func() { err = e.WrapIfErr("can't do command: can't send random", err) }()

	page, err := p.storage.PickRandom(username)
	if err != nil && err != storage.ErrNoSavedPages {
		return err
	}
	if errors.Is(err, storage.ErrNoSavedPages) {
		return p.tg.SendMessage(chatID, msgNoSavedPages)
	}
	if err := p.tg.SendMessage(chatID, page.URL); err != nil {
		return err
	}

	return p.storage.Remove(page)
}
