package server

import "container/list"

type peer interface {
	AddrPeer(p *peer)
	RemovePeer(p *peer)
}

type PeerSet struct {
	list *list.List
	//volatile peer
	leader *Peer
	//volatile
	self *Peer
}

type Peer struct {
	address string
}

func NewPeer() *Peer {
	return &Peer{}
}

func (p *Peer) Address(val string) *Peer {
	p.address = val
	return p
}

func (p *Peer) GetAddress() string {
	return p.address
}
