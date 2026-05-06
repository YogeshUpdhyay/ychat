package components

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	log "github.com/sirupsen/logrus"
)

type Avatar struct {
	widget.BaseWidget
	URL string
}

func NewAvatar(url string) *Avatar {
	avatar := &Avatar{URL: url}
	avatar.ExtendBaseWidget(avatar)
	return avatar
}

func (a *Avatar) CreateRenderer() fyne.WidgetRenderer {
	if a.URL == "" || strings.Contains(a.URL, "avatar.iran.liara.run") {
		return newAvatarRenderer(theme.AccountIcon())
	}

	imageResource, err := fyne.LoadResourceFromURLString(a.URL)
	if err != nil {
		log.Infof("error loading resource %s", err.Error())
		imageResource = theme.AccountIcon()
	}
	return newAvatarRenderer(imageResource)
}

func newAvatarRenderer(imageResource fyne.Resource) fyne.WidgetRenderer {
	image := canvas.NewImageFromResource(imageResource)
	image.SetMinSize(fyne.NewSize(50, 50))
	image.FillMode = canvas.ImageFillContain

	return widget.NewSimpleRenderer(
		image,
	)
}
