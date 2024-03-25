package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/kwaaka-team/orders-core/config/general"
	"github.com/kwaaka-team/orders-core/pkg/order"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func Run(request events.APIGatewayProxyRequest) error {
	ctx := context.Background()
	opts, err := general.LoadConfig(ctx)
	if err != nil {
		return err
	}

	bot := tgbotapi.BotAPI{
		Token: opts.TelegramErrorsBotToken,
	}

	orderCli, err := order.NewClient()
	if err != nil {
		return err
	}
	defer orderCli.Close(ctx)

	update := tgbotapi.Update{}
	err = json.Unmarshal([]byte(request.Body), &update)
	if err != nil {
		log.Println("Error unmarshalling update:", err)
		return err
	}

	if update.Message.IsCommand() && update.Message.Command() == "errors" {
		startDate := time.Now().AddDate(0, 0, -1)

		startDateStr := startDate.Format("2006-01-02")

		failedOrders, err := orderCli.GetFailedOrders(ctx, update.Message.Text)

		var messageText string
		if len(failedOrders) > 0 {
			messageText = fmt.Sprintf("Список ошибочных заказов для ресторана %d за последние сутки (%s):\n", update.Message.Text, startDateStr)
			for _, order := range failedOrders {
				orderInfo := fmt.Sprintf("ID: %d, Статус: %s, Создан: %s, Агрегатор:%s, Состав:%", order.ID, order.Status, order.CreatedAt.Format("2006-01-02 15:04:05"), order.DeliveryService, order.Products)
				messageText += "- " + orderInfo + "\n"
			}
		} else {
			messageText = fmt.Sprintf("За последние сутки (%s) ошибок в заказах для ресторана %d не найдено.", startDateStr, update.Message.Text)
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, messageText)
		if _, err := bot.Send(msg); err != nil {
			log.Println("Error sending message:", err)
			return err
		}

		return err
	}

	return nil
}

func main() {
	lambda.Start(Run)
}
