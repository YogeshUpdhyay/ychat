package ui

import (
	"context"

	"fyne.io/fyne/v2"
	fyneApp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"

	"github.com/YogeshUpdhyay/ypoker/internal/constants"
	"github.com/YogeshUpdhyay/ypoker/internal/ui/pages"
	"github.com/YogeshUpdhyay/ypoker/internal/ui/router"
	"github.com/YogeshUpdhyay/ypoker/internal/utils"
)

type UI interface {
	StartUI() error
}

type DefaultUI struct{}

// initialize the UI and start the application
func (ui *DefaultUI) StartUI(ctx context.Context, isIdentityInitialized bool) error {
	app := fyneApp.New()
	appConfig := utils.GetAppConfig()

	app.Settings().SetTheme(theme.DarkTheme())
	window := app.NewWindow(appConfig.Name)
	window.Resize(fyne.NewSize(constants.WindowWidth, constants.WindowHeight))
	rootCanvas := window.Canvas()

	// get router
	router := router.NewRouter(ctx, rootCanvas)

	// registering pages to router
	router.Register(ctx, constants.LoginRoute, &pages.Login{})
	router.Register(ctx, constants.ChatRoute, &pages.Chat{})
	router.Register(ctx, constants.RegisterRoute, &pages.Register{})
	router.Navigate(ctx, constants.RegisterRoute)
	if isIdentityInitialized {
		router.Navigate(ctx, constants.LoginRoute)
	}

	// content is set by the router
	window.ShowAndRun()

	return nil
}
