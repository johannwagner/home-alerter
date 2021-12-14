package telegram

import (
	"bytes"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/johannwagner/home-alerter/alertmanager"
	"strings"
	"text/template"
)

type Telegram struct {
	bot    *tgbotapi.BotAPI
	chatId int64
}

func New(botToken string, chatId int64) (*Telegram, error) {
	bot, err := tgbotapi.NewBotAPI(botToken)

	if err != nil {
		return nil, err
	}

	t := Telegram{bot: bot, chatId: chatId}
	return &t, nil
}

func (t *Telegram) WriteMessage(alerts []*alertmanager.TriggeredAlert) error {
	if len(alerts) == 0 {
		message := "Alles paletti. Hier gibts nichts mehr zu sehen."
		m := tgbotapi.NewMessage(t.chatId, message)
		m.ParseMode = tgbotapi.ModeMarkdown
		_, err := t.bot.Send(m)
		return err
	}

	messageParts := []string{"Hier is irgendwas gerade eher uncool:", ""}

	for _, alert := range alerts {
		tmpl, err := template.New("description").Parse(alert.Rule.Description)
		if err != nil {
			return err
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

	m := tgbotapi.NewMessage(t.chatId, finishedMessage)
	m.ParseMode = tgbotapi.ModeMarkdown
	_, err := t.bot.Send(m)
	return err
}
