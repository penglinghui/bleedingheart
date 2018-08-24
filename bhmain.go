/*
*
* The MIT License (MIT)
*
* Copyright (c) 2018 Wang Gonging
*
* Permission is hereby granted, free of charge, to any person obtaining a copy
* of this software and associated documentation files (the "Software"), to deal
* in the Software without restriction, including without limitation the rights
* to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
* copies of the Software, and to permit persons to whom the Software is
* furnished to do so, subject to the following conditions:
*
* The above copyright notice and this permission notice shall be included in
* all copies or substantial portions of the Software.
*
* THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
* IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
* FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
* AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
* LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
* OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
* THE SOFTWARE.
*
 */

package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"path"
	"strings"
	"sync"


	ggio "github.com/gogo/protobuf/io"
	ctxio "github.com/jbenet/go-context/io"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-peerstore"
	"github.com/multiformats/go-multiaddr"
)

const (
	confDirName = ".bh"
	masterID = "QmSAdkJ5ZvLz4syZMox6WK4RUk1xVTQ56HLVkWDqctoiFR"
	master = "/ip4/142.93.16.125/tcp/5564/ipfs/"+masterID
)

type BhStream struct {
	sync.RWMutex
	s net.Stream
	r ggio.ReadCloser
	w ggio.WriteCloser
}

type StreamManager struct {
	sync.RWMutex
	streamMap map[peer.ID] *BhStream
}

var (
	confDir		= path.Join(getHomeDir(), confDirName)
	g_ThisHost	host.Host
	g_MyAddr        multiaddr.Multiaddr
	g_Model		*Model
	g_StreamManager *StreamManager
	g_MasterID      peer.ID
	g_IsMaster	bool
)

func loadPrivKey(filename string) crypto.PrivKey {

	var keyLoaded = false
	var b []byte
	var prvKey crypto.PrivKey
	s, err := ioutil.ReadFile(filename)
	if (err == nil) {
		b, err = base64.StdEncoding.DecodeString(string(s))
		if (err == nil) {
			prvKey, err = crypto.UnmarshalPrivateKey(b)
			if (err == nil) {
				keyLoaded = true
			}
		}
		fmt.Println("Loaded private key file: ", filename)
	}

	if (!keyLoaded) {
		fmt.Println("Generating new private key file: ", filename)
		var r io.Reader
		r = rand.Reader
		prvKey, _, err = crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
		b, err := crypto.MarshalPrivateKey(prvKey)
		if err != nil {
			panic(err)
		}

		ioutil.WriteFile(filename, []byte(base64.StdEncoding.EncodeToString(b)), 0600)

		if err != nil {
			panic(err)
		}
	}

	return prvKey
}

/*
* addAddrToPeerstore parses a peer multiaddress and adds
* it to the given host's peerstore, so it knows how to
* contact it. It returns the peer ID of the remote peer.
* @credit examples/http-proxy/proxy.go
 */
func addAddrToPeerstore(h host.Host, addr string) peer.ID {
	// The following code extracts target's the peer ID from the
	// given multiaddress
	ipfsaddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(addr)
	pid, err := ipfsaddr.ValueForProtocol(multiaddr.P_IPFS)
	if err != nil {
		log.Fatalln(err)
	}

	peerid, err := peer.IDB58Decode(pid)
	if err != nil {
		log.Fatalln(err)
	}

	// Decapsulate the /ipfs/<peerID> part from the target
	// /ip4/<a.b.c.d>/ipfs/<peer> becomes /ip4/<a.b.c.d>
	targetPeerAddr, _ := multiaddr.NewMultiaddr(
		fmt.Sprintf("/ipfs/%s", peer.IDB58Encode(peerid)))
	targetAddr := ipfsaddr.Decapsulate(targetPeerAddr)

	// We have a peer ID and a targetAddr so we add
	// it to the peerstore so LibP2P knows how to contact it
	h.Peerstore().AddAddr(peerid, targetAddr, peerstore.PermanentAddrTTL)
	return peerid
}

func handleNewMessage(ctx context.Context, s net.Stream, r ggio.ReadCloser, w ggio.WriteCloser) {

	//mPeer := s.Conn().RemotePeer()

	for {
		pmes := new(BhMessage)
		fmt.Println("Waiting for new message...")
		switch err := r.ReadMsg(pmes); err {
		case io.EOF:
			s.Close()
			fmt.Println("Tearing down stream - EOF")
			return
		case nil:
		default:
			s.Reset()
			fmt.Println("Tearing down stream - nil")
			return
		}

		fmt.Println("Got a message from", s.Conn().RemotePeer(), ". Type: ", pmes.GetType())
		if (BhMessage_BH_PING == pmes.GetType()) {
			t := BhMessage_BH_PEERS
			rpmes := &BhMessage {
					Type: &t,
			}
			rpmes.Peers = PeerInfosToBhPeers(
					peerstore.PeerInfos(
						g_ThisHost.Peerstore(),
						g_ThisHost.Peerstore().Peers()))
			fmt.Println("Send BH_PEERS message back")
			if err := w.WriteMsg(rpmes); err != nil {
				fmt.Println("Failed", err)
				//s.Reset()
				continue
			}

			t1 := BhMessage_BH_INDEX
			rpmes1 := &BhMessage {
					Type: &t1,
					Updated: g_Model.model.Updated,
				}
			rpmes1.Files = g_Model.GetLocalFiles()
			fmt.Println("Send BH_INDEX message back")
			if err := w.WriteMsg(rpmes1); err != nil {
				fmt.Println("Failed", err)
				continue
			}
			continue
		}
		if (BhMessage_BH_PEERS == pmes.GetType()) {
			peers := BhPeersToPeerInfos(pmes.GetPeers())
			for i,p := range peers {
				fmt.Println(i, p.ID, p.Addrs)
				if p.ID != g_ThisHost.ID() {
					g_ThisHost.Peerstore().AddAddrs(p.ID, p.Addrs, peerstore.AddressTTL)
				}
			}
			continue
		}
		if (BhMessage_BH_INDEX == pmes.GetType()) {
			for _,f := range pmes.GetFiles() {
				f.Dump()
			}
			if pmes.GetUpdated() != 0 {
				*g_Model.model.Updated = pmes.GetUpdated()
				g_Model.model.GlobalFiles = pmes.GetFiles()
				fmt.Println("Received global index @", pmes.GetUpdated())
				g_Model.UpdateGlobal(false)
			} else {
				fmt.Println("Updated is 0. Index discarded")
			}
			continue
		}
		if (BhMessage_BH_REQUEST == pmes.GetType()) {
			t := BhMessage_BH_RESPONSE
			rpmes := &BhMessage {
					Type: &t,
				}
			g_Model.BuildResponse(pmes, rpmes)
			if err := w.WriteMsg(rpmes); err != nil {
				fmt.Println("Failed", err)
			}
			continue
		}
		if (BhMessage_BH_RESPONSE == pmes.GetType()) {
			g_Model.WriteBlock(pmes.GetBlockData())
			continue
		}
	}
}

func handleStream(s net.Stream) {
	log.Println("Got a new stream!", s.Conn().RemoteMultiaddr(), s.Protocol())
	ctx := context.Background() // TODO change to some timeout
	cr := ctxio.NewReader(ctx, s)
	cw := ctxio.NewWriter(ctx, s)
	r := ggio.NewDelimitedReader(cr, net.MessageSizeMax)
	w := ggio.NewDelimitedWriter(cw)
	go handleNewMessage(ctx, s, r, w)
}

func NewBhStream(peerID peer.ID) (*BhStream, error) {
	fmt.Println("NewBhStream ", peerID)
	s, err := g_ThisHost.NewStream(context.Background(), peerID, "/chat/1.0.0")
	if err != nil {
		fmt.Println("g_ThisHost.NewStream failed", err)
		return nil,err
	}
	ctx := context.Background()
	cr := ctxio.NewReader(ctx, s)
	cw := ctxio.NewWriter(ctx, s)
	r := ggio.NewDelimitedReader(cr, net.MessageSizeMax)
	w := ggio.NewDelimitedWriter(cw)
	go handleNewMessage(ctx, s, r, w)
	return &BhStream{
		s:s,
		r:r,
		w:w,
	}, nil
}

func (m *StreamManager)GetStream(peerID peer.ID) (*BhStream, error) {
	m.Lock()
	defer m.Unlock()
	var bs *BhStream
	var err error
	bs = m.streamMap[peerID]
	if bs == nil {
		bs, err = NewBhStream(peerID)
		if err != nil {
			fmt.Println("NewBhStream failed with ", err)
			return nil,err
		}
		m.streamMap[peerID] = bs
		fmt.Println("Created BhStream: ", bs)
	}
	return bs, nil
}

func (m *StreamManager)SendMessage(peerID peer.ID, pmes *BhMessage) error {
	bs,err := m.GetStream(peerID)
	if err != nil {
		return err
	}
	if err = bs.SendMessage(pmes); err != nil {
		fmt.Println("SendMessage failed:", err)
		m.CloseStream(peerID)
	}
	return err
}

func (m *StreamManager)CloseStream(peerID peer.ID) {
	m.Lock()
	bs := m.streamMap[peerID]
	if bs != nil {
		fmt.Println("Setting streamMap to nil for peer", peerID)
		bs.s.Close()
		delete(m.streamMap,peerID)
	}
	m.Unlock()
}

func (bs *BhStream)SendMessage(pmes *BhMessage) error {
	return bs.w.WriteMsg(pmes)
}

func ping() {
	t := BhMessage_BH_PING
	pmes := &BhMessage {
		Type: &t,
	}
	fmt.Println("Sending BH_PING message to server ...")
	if err := g_StreamManager.SendMessage(g_MasterID, pmes); err != nil {
		fmt.Println("SendMessage failed with", err)
	}
}

func bhmain() {

	flag.Parse()
	ensureDir(confDir)
	prvKey := loadPrivKey(path.Join(confDir, "key"))
	myId,_ := peer.IDFromPublicKey(prvKey.GetPublic())
	g_IsMaster = (myId.Pretty() == masterID)
	var sourcePort = 5564
	if !g_IsMaster {
		var i big.Int
		i.SetUint64(10000)
		randomNumber, _ := rand.Int(rand.Reader, &i)
		sourcePort = 5564 + int(randomNumber.Int64())
	}

	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", sourcePort))
	host, err := libp2p.New(
		context.Background(),
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(prvKey),
	)
	g_ThisHost = host
	g_MyAddr = sourceMultiAddr

	g_StreamManager = &StreamManager{
		streamMap: make(map[peer.ID]*BhStream),
	}

	if err != nil {
		panic(err)
	}
	root := path.Join(confDir, "bh")
	ensureDir(root)
	InitModel(root)

	fmt.Println("--- start ----")
	if g_IsMaster {
		fmt.Println("This is master", master)
		host.SetStreamHandler("/chat/1.0.0", handleStream)
		g_MasterID = host.ID()
		cmdLoop()
	} else {
		fmt.Printf("My address: ")
		// IP will be 0.0.0.0 (listen on any interface) and port will be 0 (choose one for me).
		// Although this node will not listen for any connection. It will just initiate a connect with
		// one of its peer and use that stream to communicate.
		fmt.Printf("%s/ipfs/%s\n", g_MyAddr, host.ID().Pretty())
		// Add destination peer multiaddress in the peerstore.
		// This will be used during connection and stream creation by libp2p.
		g_MasterID = addAddrToPeerstore(g_ThisHost, master)
		cmdLoop()
	}
}

func dumpHost (h host.Host) {
	for i,p := range h.Peerstore().Peers() {
		fmt.Println(i,p, h.Peerstore().Addrs(p))
		for i,a := range h.Peerstore().Addrs(p) {
			fmt.Println(i, a)
		}
	}
}

func cmdLoop () {

	stdReader := bufio.NewReader(os.Stdin)
	for {
		input, err := stdReader.ReadString('\n')
		if err != nil {
			panic(err)
		}
		input = strings.TrimRight(input, "\r\n")

		if (input == "l") {
			dumpHost(g_ThisHost)
			continue
		}
		if (input == "r") {
			g_Model.Refresh()
			continue
		}
		if (input == "d") {
			g_Model.Dump()
			continue
		}
		if (input == "ping") {
			ping()
			continue
		}
		if (input == "p") {
			g_Model.puller()
			continue
		}
		if (input == "h") || (input == "?") {
			fmt.Println(
`l - dumpHost
r - Refresh
d - Dump
ping - ping
p - Puller
h,? - help
q - quit`)
			continue
		}
		if (input == "q") {
			fmt.Println("Quiting ...")
			os.Exit(0)
		} else {
			fmt.Print("Unknown command: "+input)
		}
	}
}

func ensureDir(dir string) {
	fi, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0700)
		fatalErr(err)
	} else if fi.Mode()&0077 != 0 {
		err := os.Chmod(dir, 0700)
		fatalErr(err)
	}
}

func getHomeDir() string {
	ex, err := os.Executable()
	if err != nil { panic(err) }
	dir := path.Dir(ex)
	return dir
/*
    if runtime.GOOS == "windows" {
        home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
        if home == "" {
            home = os.Getenv("USERPROFILE")
        }
        return home
    }
    return os.Getenv("HOME")
    */
}
