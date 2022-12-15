package main

import (
	"fmt"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers"
	"github.com/PaulSonOfLars/gotgbot/v2/ext/handlers/filters/message"
	"github.com/joeyave/cartoon-bot/controller"
	"os"
)

func main() {
	bot, err := gotgbot.NewBot(os.Getenv("BOT_TOKEN"), &gotgbot.BotOpts{})
	if err != nil {
		panic("failed to create new bot: " + err.Error())
	}

	botController := &controller.BotController{}

	// Create updater and dispatcher.
	updater := ext.NewUpdater(&ext.UpdaterOpts{
		ErrorLog: nil,
		DispatcherOpts: ext.DispatcherOpts{
			Error: botController.Error,
			Panic: func(b *gotgbot.Bot, ctx *ext.Context, r interface{}) {
				err, ok := r.(error)
				if ok {
					botController.Error(bot, ctx, err)
				} else {
					botController.Error(bot, ctx, fmt.Errorf("panic: %s", fmt.Sprint(r)))
				}
			},
			MaxRoutines: 0,
		},
	})
	dispatcher := updater.Dispatcher

	dispatcher.AddHandlerToGroup(handlers.NewMessage(message.Photo, botController.Photo), 1)
	dispatcher.AddHandlerToGroup(handlers.NewMessage(message.All, botController.Start), 1)

	// Start receiving updates.
	err = updater.StartPolling(bot, &ext.PollingOpts{DropPendingUpdates: true})
	if err != nil {
		panic("failed to start polling: " + err.Error())
	}
	fmt.Printf("%s has been started...\n", bot.User.Username)

	// Idle, to keep updates coming in, and avoid bot stopping.
	updater.Idle()
}
