package p2p

import (
	"context"
	"fmt"

	"github.com/YogeshUpdhyay/ypoker/internal/constants"
	"github.com/YogeshUpdhyay/ypoker/internal/utils"
	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"
	peerStore "github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	relayv2 "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	ma "github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"
)

type P2PTransport struct {
	host             host.Host
	ListenAddr       string
	RelayAddr        string
	addPeer          chan libp2pnetwork.Stream
	identityFilePath string
}

func (t *P2PTransport) ListenAndAccept(ctx context.Context, serverName string, password string) error {
	edPrivKey, err := getIdentityKey(ctx, password, t.identityFilePath)
	if err != nil {
		return err
	}

	relayInfo, err := getRelayInfo(ctx, t.RelayAddr)
	if err != nil {
		return err
	}

	// start listening for peers
	h, err := libp2p.New(
		libp2p.Identity(edPrivKey),
		libp2p.EnableRelay(),
		libp2p.EnableAutoRelayWithStaticRelays([]peerStore.AddrInfo{*relayInfo}),
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%s", t.ListenAddr)),
	)
	if err != nil {
		return err
	}
	t.host = h

	appConfig := utils.GetAppConfig()
	log.WithContext(ctx).Infof("adding handler for %s", appConfig.StreamProtocol)
	h.SetStreamHandler(protocol.ID(appConfig.StreamProtocol), func(s libp2pnetwork.Stream) {
		log.WithContext(ctx).Infof("incoming stream from %s adding to peer list", s.Conn().RemotePeer())
		t.addPeer <- s
	})

	// connecting to relay explicitly
	if err := h.Connect(ctx, *relayInfo); err != nil {
		log.WithContext(ctx).Infof("failed to connect to relay: %v", err)
		return err
	}
	log.WithContext(ctx).Infof("✅ Connected to relay: %s", relayInfo.ID)

	connectToRelay(ctx, h, relayInfo)

	log.WithField(constants.ServerName, fmt.Sprintf("%s@%s", serverName, h.ID().String())).Infof("server listening at %v", t.GetMyFullAddr())

	return nil
}

func (t *P2PTransport) GetMyFullAddr() []string {
	var addrs []string
	for _, addr := range t.host.Addrs() {
		// Append /p2p/<peer-id> to each base address
		fullAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", t.host.ID().String()))
		relayAddr := fmt.Sprintf("%s/p2p-circuit/p2p/%s", t.RelayAddr, t.host.ID().String())
		addrWithID := addr.Encapsulate(fullAddr)
		addrs = append(addrs, addrWithID.String(), relayAddr)
	}

	return addrs
}

func getIdentityKey(ctx context.Context, decryptionKey, identityFilePath string) (crypto.PrivKey, error) {
	encryption := DefaultEncryption{}
	edPrivKey, err := encryption.LoadAndDecryptKey(decryptionKey, identityFilePath)
	if err != nil {
		log.WithContext(ctx).Errorf("error loading identity key: %v", err)
		return nil, err
	}
	log.WithContext(ctx).Info("key decrytion success")
	return edPrivKey, err
}

func getRelayInfo(ctx context.Context, relayAddr string) (*peerStore.AddrInfo, error) {
	relayMultiAddr, err := ma.NewMultiaddr(relayAddr)
	if err != nil {
		log.Infof("invalid relay multiaddr: %v", err)
		return nil, err
	}
	relayInfo, err := peerStore.AddrInfoFromP2pAddr(relayMultiAddr)
	if err != nil {
		log.Infof("invalid relay info: %v", err)
		return nil, err
	}

	log.WithContext(ctx).Infof("using relay info %v", relayInfo)
	return relayInfo, err
}

func connectToRelay(ctx context.Context, h host.Host, relayInfo *peerStore.AddrInfo) {
	// request reservation on the relay (consult your go-libp2p version API for exact call)
	if _, err := relayv2.Reserve(context.Background(), h, *relayInfo); err != nil {
		log.WithContext(ctx).Infof("❌ Relay reservation failed: %v", err)
	} else {
		log.WithContext(ctx).Info("✅ Relay reservation created manually")
	}

	for _, c := range h.Network().Conns() {
		log.WithContext(ctx).Infof("Connected to %s via %s\n", c.RemotePeer(), c.RemoteMultiaddr())
	}
}
