package components

import (
	"context"
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/YogeshUpdhyay/ypoker/internal/constants"
	"github.com/YogeshUpdhyay/ypoker/internal/db"
	"github.com/YogeshUpdhyay/ypoker/internal/eventbus"
	eventModels "github.com/YogeshUpdhyay/ypoker/internal/eventbus/models"
	"github.com/YogeshUpdhyay/ypoker/internal/p2p"
	p2pModels "github.com/YogeshUpdhyay/ypoker/internal/p2p/models"
	"github.com/YogeshUpdhyay/ypoker/internal/ui/models"
	"github.com/YogeshUpdhyay/ypoker/internal/utils"
	log "github.com/sirupsen/logrus"
)

func NewSideNavBottomBar(ctx context.Context, pendingRequestData binding.UntypedList) *fyne.Container {
	// new chat button
	newChatButton := widget.NewButtonWithIcon(
		"New Chat",
		theme.ContentAddIcon(),
		func() {
			getAddNewChatPopUp(ctx)
		},
	)

	eventbus.Get().Subscribe(constants.EventNewConnectionRequest, func(event eventbus.Event) {
		eventData := event.Payload.(eventModels.NewConnectionRequestEventData)
		pendingRequestData.Prepend(models.PeerData{
			PeerID:   eventData.PeerID,
			Username: eventData.Username,
			Avatar:   eventData.AvatarUrl,
		})
	})

	// bottom layout with new chat button
	bottomLayout := container.NewVBox(
		newChatButton,
		NewPendingRequestsCTA(
			pendingRequestData,
			func(peerID string) {
				handleIncomingRequest(ctx, peerID, constants.RequestStatusAccepted, pendingRequestData)
			},
			func(peerID string) {
				handleIncomingRequest(ctx, peerID, constants.RequestStatusRejected, pendingRequestData)
			},
		),
	)
	return container.NewPadded(
		bottomLayout,
	)
}

func getAddNewChatPopUp(ctx context.Context) {
	log.WithContext(ctx).Info("generating add new chat pop up")

	// modal title
	modalTitle := canvas.NewText("Start a new chat", color.White)
	modalTitle.TextSize = 16
	modalTitle.TextStyle.Bold = true
	modalTitleContainer := container.NewPadded(container.NewCenter(modalTitle))

	// input field for peer ID or address
	input := widget.NewEntry()
	input.SetPlaceHolder("Enter Peer ID or Address")
	inputFieldDimesion := canvas.NewRectangle(color.Transparent)
	inputFieldDimesion.SetMinSize(fyne.NewSize(300, 0))

	// cta row
	ctaRow := container.NewGridWithColumns(2)

	// underlay container with a border
	underLayContainer := canvas.NewRectangle(color.Transparent)
	underLayContainer.SetMinSize(fyne.NewSize(350, 220))

	// spacer rect
	spacerRect := canvas.NewRectangle(color.Transparent)
	spacerRect.SetMinSize(fyne.NewSize(0, 40))

	// create the content for the pop-up
	content := container.NewPadded(
		underLayContainer,
		container.NewCenter(container.NewVBox(
			modalTitleContainer,
			spacerRect,
			container.NewPadded(input, inputFieldDimesion),
			spacerRect,
			ctaRow,
		)),
	)

	// Create the pop-up
	popUp := widget.NewModalPopUp(
		content,
		fyne.CurrentApp().Driver().AllWindows()[0].Canvas(),
	)

	cancelBtn := widget.NewButton("Cancel", func() { popUp.Hide() })
	cancelBtn.Importance = widget.MediumImportance
	ctaRow.Add(cancelBtn)

	connectBtn := widget.NewButton("Connect", func() {
		address := strings.TrimSpace(input.Text)
		if address != "" {
			log.WithContext(ctx).Infof("Starting chat with peer ID: %s", address)
			// logic to initiate chat with the given peer ID goes here
			err := connectToPeer(ctx, address)
			if err != nil {
				log.WithContext(ctx).WithError(err).Errorf("error connecting to peer %s", address)
			}
			popUp.Hide()
		}
	})
	connectBtn.Importance = widget.HighImportance
	ctaRow.Add(connectBtn)

	popUp.Show()
}

func connectToPeer(ctx context.Context, address string) error {
	server := p2p.GetServer()
	if server == nil {
		return nil
	}

	_, err := server.Connect(ctx, address)
	if err != nil {
		log.WithContext(ctx).WithError(err).Errorf("error connecting to peer %s", address)
		return err
	}

	log.WithContext(ctx).Infof("successfully connected to peer %s", address)

	return nil
}

func handleIncomingRequest(ctx context.Context, peerID, decision string, pendingRequestData binding.UntypedList) {
	log.WithContext(ctx).Infof("handling incoming request from peer %s with decision %s", peerID, decision)

	appConfig := utils.GetAppConfig()
	server := p2p.GetServer()
	if server == nil {
		log.WithContext(ctx).Error("server not initialized")
		return
	}

	dbConnReq := db.ConnectionRequests{}
	tx := db.Get().
		Where(&db.ConnectionRequests{
			PeerID: peerID,
			Status: constants.RequestStatusAwaitingDecision,
		}).
		First(&dbConnReq)
	if tx.Error != nil {
		log.WithContext(ctx).WithError(tx.Error).Errorf("error fetching connection request from peer %s", peerID)
		return
	}

	peer := server.GetPeerFromPeerID(ctx, peerID)
	if peer == nil && dbConnReq.Address != constants.Empty {
		log.WithContext(ctx).Infof("peer %s is not connected, reconnecting using %s", peerID, dbConnReq.Address)
		var err error
		peer, err = server.Connect(ctx, fmt.Sprintf("%s/p2p/%s", dbConnReq.Address, dbConnReq.PeerID))
		if err != nil {
			log.WithContext(ctx).WithError(err).Errorf("error reconnecting to peer %s", peerID)
			return
		}
	}
	if peer == nil {
		log.WithContext(ctx).Errorf("peer %s is not connected and no address is available", peerID)
		return
	}

	envelope := &p2pModels.Envelope{
		Type: p2pModels.MsgTypeHandshakeReject,
	}

	if decision == constants.RequestStatusAccepted {
		// fetching usermetadata
		userMetadata := db.UserMetadata{}
		tx := db.Get().First(&userMetadata)
		if tx.Error != nil {
			log.WithContext(ctx).WithError(tx.Error).Info("error fetching metadata")
			return
		}

		envelope.Type = p2pModels.MsgTypeHandshakeAck
		envelope.Payload = utils.MarshalPayload(p2pModels.HandShake{
			Version: appConfig.Version,
			UserInfo: p2pModels.UserInfo{
				Username:  userMetadata.Username,
				AvatarUrl: userMetadata.AvatarUrl,
			},
		})
	}

	err := peer.Send(envelope)
	if err != nil {
		log.WithContext(ctx).WithError(err).Infof("errro sending hanshake reponse to peer")
		return
	}
	log.WithContext(ctx).Info("handshake decision sent to peer")

	dbConnReq.Status = decision
	tx = db.Get().Save(&dbConnReq)
	if tx.Error != nil {
		log.WithContext(ctx).WithError(tx.Error).Errorf("error updating connection request status for peer %s", peerID)
		return
	}

	if decision == constants.RequestStatusAccepted {
		log.WithContext(ctx).Infof("connection request from peer %s accepted", peerID)
		peerInfo := db.PeerInfo{
			PeerID:    dbConnReq.PeerID,
			Username:  dbConnReq.Username,
			AvatarUrl: dbConnReq.AvatarUrl,
			Address:   dbConnReq.Address,
			Status:    constants.ConnectionStateInactive,
		}
		tx := db.Get().Create(&peerInfo)
		if tx.Error != nil {
			log.WithContext(ctx).WithError(tx.Error).Errorf("error creating peer info for peer %s", peerID)
			return
		}
		log.WithContext(ctx).Infof("peer info for peer %s created successfully", peerID)

		eventbus.Get().Publish(constants.EventThreadListUpdated, nil)
	}

	// removing the peer id whose decision is taken from the
	// pending requests list
	pendingPeerReqList, _ := pendingRequestData.Get()
	pendingPeerReqList = removePeerByID(pendingPeerReqList, peerID)
	_ = pendingRequestData.Set(pendingPeerReqList)
}

func removePeerByID(list []any, peerID string) []any {
	newList := make([]any, 0, len(list))
	for _, item := range list {
		if pd, ok := item.(models.PeerData); ok {
			if pd.PeerID != peerID {
				newList = append(newList, pd)
			}
		}
	}
	return newList
}
