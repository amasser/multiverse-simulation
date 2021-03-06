package network

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/iotaledger/hive.go/crypto"
)

// region Peer /////////////////////////////////////////////////////////////////////////////////////////////////////////

type Peer struct {
	ID        PeerID
	Neighbors map[PeerID]*Connection
	Socket    chan interface{}
	Node      Node

	startOnce      sync.Once
	shutdownOnce   sync.Once
	shutdownSignal chan struct{}
}

func NewPeer(node Node) (peer *Peer) {
	peer = &Peer{
		ID:        NewPeerID(),
		Neighbors: make(map[PeerID]*Connection),
		Socket:    make(chan interface{}, 1024),
		Node:      node,

		shutdownSignal: make(chan struct{}, 1),
	}

	return
}

func (p *Peer) SetupNode(consensusWeightDistribution *ConsensusWeightDistribution) {
	p.Node.Setup(p, consensusWeightDistribution)
}

func (p *Peer) Start() {
	p.startOnce.Do(func() {
		go p.run()
	})
}

func (p *Peer) Shutdown() {
	p.shutdownOnce.Do(func() {
		close(p.shutdownSignal)
	})
}

func (p *Peer) ReceiveNetworkMessage(message interface{}) {
	p.Socket <- message
}

func (p *Peer) GossipNetworkMessage(message interface{}) {
	for _, neighborConnection := range p.Neighbors {
		neighborConnection.Send(message)
	}
}

func (p *Peer) String() string {
	return fmt.Sprintf("Peer%d", p.ID)
}

func (p *Peer) run() {
	for {
		select {
		case <-p.shutdownSignal:
			return
		case networkMessage := <-p.Socket:
			p.Node.HandleNetworkMessage(networkMessage)
		}
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region PeerID ///////////////////////////////////////////////////////////////////////////////////////////////////////

type PeerID int64

var peerIDCounter int64

func NewPeerID() PeerID {
	return PeerID(atomic.AddInt64(&peerIDCounter, 1) - 1)
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region Connection ///////////////////////////////////////////////////////////////////////////////////////////////////

type Connection struct {
	Socket       chan<- interface{}
	NetworkDelay time.Duration
	PacketLoss   float64
}

func (c *Connection) Send(message interface{}) {
	if crypto.Randomness.Float64() <= c.PacketLoss {
		return
	}

	go func() {
		time.Sleep(c.NetworkDelay)

		c.Socket <- message
	}()
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
