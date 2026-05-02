package p2p

import (
	"context"
	"path/filepath"
	"time"

	"github.com/YogeshUpdhyay/ypoker/internal/constants"
	"github.com/YogeshUpdhyay/ypoker/internal/db"
	p2pModels "github.com/YogeshUpdhyay/ypoker/internal/p2p/models"
	"github.com/YogeshUpdhyay/ypoker/internal/utils"
	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"
)

type Server struct {
	ServerConfig
	handler    Handler
	transport  *P2PTransport
	peers      map[string]*Peer
	addPeer    chan libp2pnetwork.Stream
	deletePeer chan peer.ID
	msgCh      chan *p2pModels.Envelope
}

type ServerConfig struct {
	ListenAddr       string
	RelayAddr        string
	Version          string
	ServerName       string
	IdentityFilePath string
}

func (cfg *ServerConfig) ApplyDefaults() {
	if cfg.IdentityFilePath == "" {
		cfg.IdentityFilePath = filepath.Join(constants.ApplicationDataDir, constants.ApplicationIdentityFileName)
	}

	if cfg.Version == "" {
		cfg.Version = "yoker v0.1-alpha"
	}

	if cfg.ServerName == "" {
		cfg.ServerName = "yoker"
	}

	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":3000"
	}

	if cfg.RelayAddr == constants.Empty {
		cfg.RelayAddr = constants.DefaultRelayAddr
	}
}

var server *Server

func GetServer() *Server {
	return server
}

func NewServer(cfg ServerConfig) *Server {
	// apply default values if not set
	cfg.ApplyDefaults()

	server = &Server{
		ServerConfig: cfg,
		handler:      &DefaultHandler{},
		peers:        make(map[string]*Peer),
		addPeer:      make(chan libp2pnetwork.Stream),
		deletePeer:   make(chan peer.ID),
		msgCh:        make(chan *p2pModels.Envelope),
	}

	server.transport = &P2PTransport{
		ListenAddr: server.ListenAddr,
		RelayAddr:  server.RelayAddr,
		addPeer:    server.addPeer,

		// internals
		identityFilePath: server.IdentityFilePath,
	}

	return server
}

func (s *Server) Start(ctx context.Context, password string) {
	log.Info("starting server")
	go s.loop(ctx)
	if err := s.transport.ListenAndAccept(ctx, s.ServerName, password); err != nil {
		log.Fatal("error starting the server")
	}
}

func (s *Server) loop(ctx context.Context) {
	for {
		select {
		case conn := <-s.addPeer:
			log.WithContext(ctx).Info("adding new peer connection")
			peerId := s.handleNewStream(ctx, conn)
			log.WithContext(ctx).Infof("new player connected %s", peerId)
		case msg := <-s.msgCh:
			s.handler.HandleMessage(ctx, msg)
		case peerId := <-s.deletePeer:
			// connection is closed removing from peers
			delete(s.peers, string(peerId.String()))
			log.Infof("player disconnected %s", peerId.String())
		}
	}
}

func (s *Server) Connect(ctx context.Context, remoteAddr string) (*Peer, error) {
	// parse the multiaddress
	maddr, err := ma.NewMultiaddr(remoteAddr)
	if err != nil {
		log.Errorf("invalid multiaddress: %s", err)
		return nil, err
	}

	// extract peer info from multiaddr (contains ID + address)
	peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		log.Errorf("invalid peer info from multiaddr: %s", err)
		return nil, err
	}

	// add peer to peerstore so we can dial it
	if err := s.transport.host.Connect(ctx, *peerInfo); err != nil {
		log.Errorf("error connecting to peer: %s", err)
		return nil, err
	}

	// ✅ Wait for identify to finish
	for i := 0; i < 10; i++ {
		if protocols, _ := s.transport.host.Peerstore().GetProtocols(peerInfo.ID); len(protocols) > 0 {
			log.WithContext(ctx).Infof("protocols %v", protocols)
			break
		}
		time.Sleep(1 * time.Second)
	}

	// open a stream to the peer using your protocol
	appConfig := utils.GetAppConfig()
	log.WithContext(ctx).Infof("starting new stream  for %s and %s", peerInfo.ID.String(), appConfig.StreamProtocol)
	stream, err := s.transport.host.NewStream(ctx, peerInfo.ID, protocol.ID(appConfig.StreamProtocol))
	if err != nil {
		log.Errorf("error opening stream to %s: %s", peerInfo.ID, err)
		return nil, err
	}

	peer := &Peer{
		conn:   stream,
		Status: constants.ConnectionStateActive,
		PeerID: stream.Conn().RemotePeer().String(),
	}

	// send handshake message
	if err := s.sendHandshake(ctx, peer); err != nil {
		log.Errorf("error sending handshake: %s", err)
		return nil, err
	}
	log.WithContext(ctx).Infof("connection request sucess %s at %s adding to peers", peerInfo.ID, remoteAddr)

	// adding this pending connection to db
	connectionRequest := db.ConnectionRequests{
		PeerID:  stream.Conn().RemotePeer().String(),
		Status:  constants.RequestStatusSent,
		Address: peer.GetMultiAddrs(ctx).String(),
	}
	db.Get().Create(&connectionRequest)

	s.addPeer <- stream

	return peer, nil
}

func (s *Server) GetMyFullAddr() []string {
	return s.transport.GetMyFullAddr()
}

func (s *Server) InitializeIdentityFlow(ctx context.Context, password string) error {
	encryption := DefaultEncryption{}
	idKey, err := encryption.GenerateIdentityKey()
	if err != nil {
		log.WithContext(ctx).Errorf("error generating identity key %s", err.Error())
		return err
	}

	err = encryption.EncryptAndSaveIdentityKey(
		password,
		idKey,
		s.IdentityFilePath,
	)
	if err != nil {
		log.WithContext(ctx).Errorf("error saving identity key %s", err.Error())
		return err
	}

	log.WithContext(ctx).Infof("identity key saved successfully at %s", s.IdentityFilePath)
	return nil
}

func (s *Server) GetPeerFromPeerID(ctx context.Context, peerID string) *Peer {
	peer, ok := s.peers[peerID]
	if !ok {
		log.WithContext(ctx).Infof("peer not found, peer id: %s", peerID)
	}
	return peer
}

func (s *Server) GetNodeID(ctx context.Context) string {
	return s.transport.host.ID().String()
}

func (s *Server) sendHandshake(ctx context.Context, p *Peer) error {
	// fetching usermetadata
	userMetadata := db.UserMetadata{}
	db.Get().First(&userMetadata)

	handshakeEnvelope := p2pModels.Envelope{
		Type: p2pModels.MsgTypeHandshake,
		From: s.transport.host.ID().String(),
		Payload: utils.MarshalPayload(p2pModels.HandShake{
			Version: utils.DefaultAppConfig().Version,
			UserInfo: p2pModels.UserInfo{
				Username:  userMetadata.Username,
				AvatarUrl: userMetadata.AvatarUrl,
			},
		}),
	}

	if err := p.Send(&handshakeEnvelope); err != nil {
		log.WithContext(ctx).Infof("error sending hanshake payload %v", err)
		return err
	}
	log.WithContext(ctx).Info("hanshake sent successfully.")
	return nil
}

func (s *Server) handleNewStream(ctx context.Context, conn libp2pnetwork.Stream) string {
	// updating peer list
	peerId := string(conn.Conn().RemotePeer().String())
	peer := Peer{
		conn:   conn,
		Status: constants.ConnectionStateActive,
		PeerID: peerId,
	}

	// starting read loop
	s.peers[peerId] = &peer
	go peer.ReadLoop(ctx, s.msgCh, s.deletePeer)
	return peerId
}
