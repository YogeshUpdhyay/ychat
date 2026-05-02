package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/YogeshUpdhyay/ypoker/internal/constants"
	"github.com/YogeshUpdhyay/ypoker/internal/db"
	"github.com/YogeshUpdhyay/ypoker/internal/eventbus"
	"github.com/YogeshUpdhyay/ypoker/internal/ui"
	"github.com/YogeshUpdhyay/ypoker/internal/utils"
	log "github.com/sirupsen/logrus"
)

func main() {
	ctx := context.Background()

	// initialize application
	initializeApp(ctx)
	uiImpl := ui.DefaultUI{}

	if !identityFileExists() {
		// writing default config file
		log.WithContext(ctx).Info("first startup, creating the application config file")
		if err := utils.WriteAppConfig(); err != nil {
			log.WithContext(ctx).WithError(err).Fatal("failed to write default config file")
		}

		log.WithContext(ctx).Infof("identity not initialized, starting initialization flow")
		uiImpl.StartUI(ctx, false)
		return
	}
	log.WithContext(ctx).Info("identity already present, skipping initialization, starting UI")
	uiImpl.StartUI(ctx, true)
}

// check if the identity file exists
func identityFileExists() bool {
	_, err := os.Stat(fmt.Sprintf("%s/%s", constants.ApplicationDataDir, constants.ApplicationIdentityFileName))
	return !errors.Is(err, os.ErrNotExist)
}

func initializeApp(ctx context.Context) {
	log.SetFormatter(&log.JSONFormatter{})
	// create application directory
	dir := constants.ApplicationDataDir
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.WithContext(ctx).Info("error creating application data directory")
	}

	err := db.Initialize(ctx)
	if err != nil {
		log.WithError(err).Fatal("failed to initialize the database")
	}

	eventbus.New()
	log.WithContext(ctx).Info("application initialized successfully")
}
