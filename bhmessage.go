package bleedingheart

import (
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

func PeerInfoToBhMessagePeer(p peerstore.PeerInfo) *BhMessage_Peer {
	m := new(BhMessage_Peer)
	m.Addrs = make([][]byte, len(p.Addrs))
	for i, maddr := range p.Addrs {
		m.Addrs[i] = maddr.Bytes()
	}
	s := string(p.ID)
	m.Id = &s
	return m
}

func (m *BhMessage_Peer) Addresses() []ma.Multiaddr {
	if m == nil {
		return nil
	}
	maddrs := make([]ma.Multiaddr, 0, len(m.Addrs))
	for _, addr := range m.Addrs {
		maddr, err := ma.NewMultiaddrBytes(addr)
		if err != nil {
			continue
		}
		maddrs = append(maddrs, maddr)
	}
	return maddrs
}

func BhMessagePeerToPeerInfo(m *BhMessage_Peer) *peerstore.PeerInfo {
	return &peerstore.PeerInfo {
		ID:	peer.ID(m.GetId()),
		Addrs:	m.Addresses(),
	}
}
