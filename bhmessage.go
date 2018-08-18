package main

import (
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

func PeerInfoToBhPeer(p peerstore.PeerInfo) *BhMessage_Peer {
	m := new(BhMessage_Peer)
	m.Addrs = make([][]byte, len(p.Addrs))
	for i, maddr := range p.Addrs {
		m.Addrs[i] = maddr.Bytes()
	}
	s := string(p.ID)
	m.Id = &s
	return m
}

func PeerInfosToBhPeers(peers []peerstore.PeerInfo) []*BhMessage_Peer {
	bhpeers := make([]*BhMessage_Peer, len(peers))
	for i, p := range peers {
		bhpeers[i] = PeerInfoToBhPeer(p)
	}
	return bhpeers
}

func (bhpeer *BhMessage_Peer) Addresses() []ma.Multiaddr {
	if bhpeer == nil {
		return nil
	}
	maddrs := make([]ma.Multiaddr, 0, len(bhpeer.Addrs))
	for _, addr := range bhpeer.Addrs {
		maddr, err := ma.NewMultiaddrBytes(addr)
		if err != nil {
			continue
		}
		maddrs = append(maddrs, maddr)
	}
	return maddrs
}

func BhPeerToPeerInfo(bhpeer *BhMessage_Peer) *peerstore.PeerInfo {
	return &peerstore.PeerInfo {
		ID:	peer.ID(bhpeer.GetId()),
		Addrs:	bhpeer.Addresses(),
	}
}

func BhPeersToPeerInfos(bhpeers []*BhMessage_Peer) []*peerstore.PeerInfo {
	peers := make([]*peerstore.PeerInfo, 0, len(bhpeers))
	for _, bhpeer := range bhpeers {
		peers = append(peers, BhPeerToPeerInfo(bhpeer))
	}
	return peers
}
