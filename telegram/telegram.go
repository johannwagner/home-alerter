package telegram

import (
	"bytes"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/johannwagner/home-alerter/alertmanager"
	"math/rand"
	"strings"
	"text/template"
)

type Telegram struct {
	bot    *tgbotapi.BotAPI
	chatId int64
}

var resolvedAlerts = []string{
	"Alles paletti. Hier gibt es nichts mehr zu sehen.",
	"Das wars schon. Danke für ihre Aufmerksamkeit.",
	"Weiterschlafen, hier gibts nichts zu sehen.",
	"Gut gemacht!",
}

var currentAlerts = []string{
	"Hier is irgendwas gerade eher uncool:",
	"Jo, diggi, das is irgendwie blöd hier:",
	"Was ist denn hier los? Schau ma:",
	"Hier schepperts gleich, was is denn los hier?",
}

func GetRandomMessage(messageList []string) string {
	lenMessages := len(messageList)
	randomIndex := rand.Intn(lenMessages)
	return messageList[randomIndex]
}

func New(botToken string, chatId int64) (*Telegram, error) {
	bot, err := tgbotapi.NewBotAPI(botToken)

	if err != nil {
		return nil, err
	}

	t := Telegram{bot: bot, chatId: chatId}
	return &t, nil
}

func (t *Telegram) WriteMessage(knownMessageId *int, alerts []*alertmanager.TriggeredAlert) (*int, error) {
	if len(alerts) == 0 {
		message := GetRandomMessage(resolvedAlerts)
		m := tgbotapi.NewMessage(t.chatId, message)
		m.ParseMode = tgbotapi.ModeMarkdown
		msg, err := t.bot.Send(m)
		return &msg.MessageID, err
	}

	currentAlertMessage := GetRandomMessage(currentAlerts)
	messageParts := []string{currentAlertMessage, ""}

	for _, alert := range alerts {
		tmpl, err := template.New("description").Parse(alert.Rule.Description)
		if err != nil {
			panic(err)
		}

		tmplData := map[string]string{}

		for _, pair := range alert.Metric.Label {
			tmplData[pair.GetName()] = pair.GetValue()
		}

		var descBuffer bytes.Buffer
		err = tmpl.Execute(&descBuffer, tmplData)
		if err != nil {
			panic(err)
		}
		descString := descBuffer.String()

		messageParts = append(
			messageParts,
			descString,
		)

	}

	messageParts = append(messageParts, "", "Schau mal besser nach...")
	finishedMessage := strings.Join(messageParts, "\n")

	if knownMessageId != nil {
		m := tgbotapi.NewEditMessageText(t.chatId, *knownMessageId, finishedMessage)
		m.ParseMode = tgbotapi.ModeMarkdown
		sendMsg, err := t.bot.Send(m)
		return &sendMsg.MessageID, err
	} else {
		m := tgbotapi.NewMessage(t.chatId, finishedMessage)
		m.ParseMode = tgbotapi.ModeMarkdown
		sendMsg, err := t.bot.Send(m)
		return &sendMsg.MessageID, err
	}
}
