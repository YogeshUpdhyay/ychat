package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	relay "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	identify "github.com/libp2p/go-libp2p/p2p/protocol/identify"

	log "github.com/sirupsen/logrus"
)

const IdentityFilePath = ".yoker/relayID.key"

func getIdentityKey(ctx context.Context, filePath string) (libp2pcrypto.PrivKey, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.WithContext(ctx).Infof("error reading id key %s", err.Error())
		return nil, err
	}

	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		log.WithContext(ctx).Infof("error decoding id key %s", err.Error())
		return nil, err
	}

	priv, err := libp2pcrypto.UnmarshalPrivateKey(decoded)
	if err != nil {
		log.WithContext(ctx).Infof("error unmarshal id key %s", err.Error())
		return nil, err
	}

	return priv, err
}

func generateAndWriteIDKey(ctx context.Context, filePath string) (libp2pcrypto.PrivKey, error) {
	priv, _, err := libp2pcrypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		log.WithContext(ctx).Infof("error creating a priv key %s", err.Error())
		return nil, err
	}

	privBytes, err := libp2pcrypto.MarshalPrivateKey(priv)
	if err != nil {
		log.WithContext(ctx).Infof("error marshalling a priv key %s", err.Error())
		return nil, err
	}

	encoded := base64.StdEncoding.EncodeToString(privBytes)
	if err := os.MkdirAll(filepath.Dir(filePath), 0700); err != nil {
		log.WithContext(ctx).Infof("error creating id key dir %s", err.Error())
		return nil, err
	}
	if err := os.WriteFile(filePath, []byte(encoded), 0600); err != nil {
		log.WithContext(ctx).Infof("error writing id key %s", err.Error())
		return nil, err
	}

	return priv, nil
}

// check if the identity file exists
func identityFileExists() bool {
	_, err := os.Stat(IdentityFilePath)
	return !errors.Is(err, os.ErrNotExist)
}

func main() {
	ctx := context.Background()

	var key libp2pcrypto.PrivKey
	var err error
	if identityFileExists() {
		key, err = getIdentityKey(ctx, IdentityFilePath)
	} else {
		key, err = generateAndWriteIDKey(ctx, IdentityFilePath)
	}

	if err != nil {
		log.WithContext(ctx).Infof("error preparing the id file %s", err.Error())
		panic(err)
	}

	relayOpts := []relay.Option{
		relay.WithLimit(&relay.RelayLimit{
			Duration: time.Hour,
			Data:     1 << 25, // 32 MiB per direction.
		}),
	}

	// relay host
	relayHost, err := libp2p.New(
		libp2p.Identity(key),
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/4001"),
		libp2p.EnableRelayService(relayOpts...),
	)
	if err != nil {
		log.Fatalf("failed to create host: %v", err)
	}

	relayHost.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(_ network.Network, conn network.Conn) {
			fmt.Printf("🔗 Peer connected: %s (%s)\n", conn.RemotePeer(), conn.RemoteMultiaddr())
		},
		DisconnectedF: func(_ network.Network, conn network.Conn) {
			fmt.Printf("❌ Peer disconnected: %s\n", conn.RemotePeer())
		},
	})

	log.Infof("Relay Node started with PeerID: %s", relayHost.ID())

	_, err = relay.New(relayHost, relayOpts...)
	if err != nil {
		log.Fatalf("failed to enable relay service: %v", err)
	}
	log.Infof("✅ Relay service enabled")

	identify.NewIDService(relayHost)

	log.Info("🚀 Relay Node is running. Share these addresses with clients:")
	for _, addr := range relayHost.Addrs() {
		fullAddr := fmt.Sprintf("%s/p2p/%s", addr, relayHost.ID().String())
		log.Info(fullAddr)
	}
	log.Info("Example relay address peers will use:")
	log.Infof("/ip4/<relay-ip>/tcp/4001/p2p/%s/p2p-circuit", relayHost.ID().String())

	select {}
}
