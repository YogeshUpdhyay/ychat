package pages

import (
	"context"
	"fmt"
	"image/color"
	"net/url"
	"time"

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

type Register struct {
}

func (l *Register) OnShow(_ context.Context) {
	// Logic to execute when the login page is shown
}

func (l *Register) OnHide(_ context.Context) {
	// Logic to execute when the login page is hidden
}

func (l *Register) Content(ctx context.Context) fyne.CanvasObject {
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
	registerForm := forms.NewRegisterForm()

	username := widget.NewEntry()
	username.SetPlaceHolder("Username")
	username.Bind(registerForm.Username)

	password := widget.NewPasswordEntry()
	password.SetPlaceHolder("Password")
	password.Bind(registerForm.Password)

	confirmPassword := widget.NewPasswordEntry()
	confirmPassword.SetPlaceHolder("Confirm Password")
	confirmPassword.Bind(registerForm.ConfirmPassword)

	submit := widget.NewButton("Submit", func() {
		// validate and get data from the form
		username, password, _, err := registerForm.GetData()
		if err != nil {
			log.Errorf("form validation failed: %v", err)
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
		server.InitializeIdentityFlow(ctx, password)
		go server.Start(ctx, password)
		// wait for server to start
		time.Sleep(2 * time.Second)
		log.WithContext(ctx).Infof("serever started with addresses: %+v", server.GetMyFullAddr())

		// storing user name to the metadata table
		userMetadata := db.UserMetadata{}
		db.Get().First(&userMetadata, username)
		if userMetadata.Username != constants.Empty {
			log.Errorf("user already exists with username: %s", username)
			// handle this error in the ui as well
			return
		}

		// update usermetadata
		userMetadata.Username = username
		userMetadata.AvatarUrl = fmt.Sprintf(constants.AvatarUrlTemplate, url.QueryEscape(username))
		userMetadata.LastLoginTs = time.Now()
		userMetadata.CreateTs = time.Now()
		userMetadata.PeerID = server.GetNodeID(ctx)
		db.Get().Create(&userMetadata)
		log.WithContext(ctx).Infof("user metadata saved successfully for user: %s", username)

		router.GetRouter().Navigate(ctx, constants.ChatRoute)
	})
	submit.Alignment = widget.ButtonAlignCenter

	// vbox size
	vBoxsize := canvas.NewRectangle(color.Transparent)
	vBoxsize.SetMinSize(fyne.NewSize(300, 125))

	register := container.NewCenter(
		container.NewStack(
			underLayContainer,
			container.NewCenter(
				container.NewPadded(
					container.NewVBox(
						vBoxsize,
						logo,
						username,
						password,
						confirmPassword,
						submit,
						vBoxsize,
					),
				),
			),
		),
	)

	return register
}
