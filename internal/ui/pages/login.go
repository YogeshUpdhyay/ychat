package pages

import (
	"context"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/YogeshUpdhyay/ypoker/internal/constants"
	"github.com/YogeshUpdhyay/ypoker/internal/db"
	"github.com/YogeshUpdhyay/ypoker/internal/p2p"
	"github.com/YogeshUpdhyay/ypoker/internal/ui/assets"
	"github.com/YogeshUpdhyay/ypoker/internal/ui/forms"
	"github.com/YogeshUpdhyay/ypoker/internal/ui/router"
	"github.com/YogeshUpdhyay/ypoker/internal/utils"
	log "github.com/sirupsen/logrus"
)

type Login struct {
}

func (l *Login) OnShow(_ context.Context) {
	// Logic to execute when the login page is shown
}

func (l *Login) OnHide(_ context.Context) {
	// Logic to execute when the login page is hidden
}

func (l *Login) Content(ctx context.Context) fyne.CanvasObject {
	// underlay container with a border
	underLayContainer := canvas.NewRectangle(color.Transparent)
	underLayContainer.SetMinSize(fyne.NewSize(350, 220))
	underLayContainer.CornerRadius = 10
	underLayContainer.StrokeColor = color.White
	underLayContainer.StrokeWidth = 2

	// logo
	logo := canvas.NewImageFromResource(assets.ResourceYChatPng)
	logo.FillMode = canvas.ImageFillContain
	logo.SetMinSize(fyne.NewSize(100, 100))

	// form elements and data bindings
	loginForm := forms.NewLoginForm()

	username := widget.NewEntry()
	username.SetPlaceHolder("Username")
	username.Bind(loginForm.Username)

	password := widget.NewPasswordEntry()
	password.SetPlaceHolder("Password")
	password.Bind(loginForm.Password)

	submit := widget.NewButton("Submit", func() {
		// getting data from the form
		username, password, err := loginForm.GetData()
		if err != nil {
			log.Errorf("error getting login data: %v", err)
			return
		}

		// fetching user from the database
		userMetadata := db.UserMetadata{}
		db.Get().First(&userMetadata)
		if userMetadata.Username == constants.Empty || userMetadata.Username != username {
			log.Errorf("user not found")
			// add how ui will show this error
			return
		}

		// starting the server
		appConfig := utils.GetAppConfig()
		serverConfig := p2p.ServerConfig{
			ListenAddr: appConfig.Port,
			Version:    appConfig.Version,
			ServerName: appConfig.Name,
			RelayAddr:  appConfig.RelayAddr,
		}
		server := p2p.NewServer(serverConfig)
		go server.Start(ctx, password)

		router.GetRouter().Navigate(ctx, constants.ChatRoute)
	})
	submit.Alignment = widget.ButtonAlignCenter

	// vbox size
	vBoxsize := canvas.NewRectangle(color.Transparent)
	vBoxsize.SetMinSize(fyne.NewSize(300, 125))

	login := container.NewCenter(
		container.NewStack(
			underLayContainer,
			container.NewCenter(
				container.NewPadded(
					container.NewVBox(
						vBoxsize,
						logo,
						username,
						password,
						submit,
						vBoxsize,
					),
				),
			),
		),
	)

	return login
}
