package telegram

import (
	"errors"
	"fmt"
	"github.com/tealeg/xlsx"
	"log"
	"regexp"
	"strings"
	"sync"
	storage "TelegramBot/storage"
	"TelegramBot/events"
	"TelegramBot/lib/e"
)

const (
	RunCmd   = "/rnd"
	HelpCmd  = "/help"
	StartCmd = "/start"
)

var (
	userStates    = make(map[int]bool)
	excelIndex    = make(map[string][]string)
	indexMutex    = sync.RWMutex{}
	excelFilePath = "./ex.xlsm" // Update this path
)

func init() {
	go loadExcelData()
}

func loadExcelData() {
	file, err := xlsx.OpenFile(excelFilePath)
	if err != nil {
		log.Fatalf("Error opening the file: %v", err)
		return
	}

	indexMutex.Lock()
	defer indexMutex.Unlock()

	for _, row := range file.Sheets[0].Rows {
		stir := row.Cells[0].String()
		var rowData []string
		for _, cell := range row.Cells {
			rowData = append(rowData, cell.String())
		}
		excelIndex[stir] = rowData
	}
}

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
	return p.tg.SendMessage(chatID, "Please enter your STIR number (9-digit number):")
}

func (p *Processor) handleSTIRNumber(stir string, chatID int) error {
	if matched, _ := regexp.MatchString(`^\d{9}$`, stir); !matched {
		return p.tg.SendMessage(chatID, "Malumot xato kiritilgan.")
	}

	indexMutex.RLock()
	defer indexMutex.RUnlock()

	rowData, ok := excelIndex[stir]
	if !ok {
		return p.tg.SendMessage(chatID, "Malumot topilmadi")
	}

	orgName := rowData[1]
	oked := rowData[2]
	okedName := rowData[3]
	region := rowData[4]
	district := rowData[5]

	response := fmt.Sprintf(
		"СТИР: %s\nТашкилотингиз номи: %s\nЖойлашган худудингиз: %s\nМанзилингиз: %s\nОКЭД рақами: %s\nОКЭД номи: %s",
		stir, orgName, region, district, oked, okedName,
	)

	userStates[chatID] = false // Reset the user state
	return p.tg.SendMessage(chatID, response)
}

func (p *Processor) ProcessMessage(event events.Event) error {
	meta, err := meta(event)
	if err != nil {
		return e.Wrap("can't process message", err)
	}

	if userStates[meta.ChatId] {
		return p.handleSTIRNumber(event.Text, meta.ChatId)
	}

	if event.Text == StartCmd {
		return p.doCmd(event.Text, meta.ChatId, meta.UserName)
	}

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