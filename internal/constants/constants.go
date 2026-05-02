package constants

const (
	Empty                       = ""
	PeerID                      = "peerID"
	ApplicationDataDir          = ".yoker"
	ApplicationIdentityFileName = "identitypi.key"
	ApplicationConfigFileName   = "config.yml"
	ApplicationDBFilleName      = "yoker.db"
	DatabasePathDefault         = ApplicationDataDir + "/" + ApplicationDBFilleName
	DefaultRelayAddr            = "/dns4/0.tcp.in.ngrok.io/tcp/19307/p2p/12D3KooWHNfGzXZ9cUbJ2pAP7nkFEPzWgNWj91LfU2NMNasbZPRf"

	Ping = "PING"
	Pong = "PONG"

	ServerName = "serverName"

	ServerNameDefault     = "yoker alpha"
	ServerPortDefault     = "9000"
	ServerVersionDefault  = "yoker1.0.0"
	StreamProtocolDefault = "/ypoker/1.0.0"

	// connection states
	ConnectionStateActive   = "active"
	ConnectionStateInactive = "inactive"
	ConnectionStatePending  = "pending"

	// request status
	RequestStatusSent             = "sent"
	RequestStatusAwaitingDecision = "awaiting_decision"
	RequestStatusAccepted         = "accepted"
	RequestStatusRejected         = "rejected"

	// message statuses
	ToBeSent  = "toBeSent"
	Sending   = "sending"
	Sent      = "sent"
	Delivered = "delivered"
	Read      = "read"
	Received  = "received"

	// events
	EventNewConnectionRequest = "new_connection_request"
	EventThreadListUpdated    = "thread_list_upadted"
	EventNewMessage           = "new_message"

	// dummy
	DummyAvatarUrl    = "https://avatar.iran.liara.run/username?username=dummy&bold=false&length=1"
	AvatarUrlTemplate = "https://avatar.iran.liara.run/username?username=%s&bold=true&length=1"
)
