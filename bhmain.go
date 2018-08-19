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
	"os"
	"time"

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

var g_ThisHost host.Host

func check(err error) {
	if (err != nil) {
		panic(err)
	}
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
				fmt.Println("Failed", err);
				//s.Reset()
				continue
			}
			continue
		}
		if (BhMessage_BH_PEERS == pmes.GetType()) {
			peers := BhPeersToPeerInfos(pmes.GetPeers())
			for i,p := range peers {
				fmt.Println(i, p.ID, p.Addrs)
			}
			continue
		}

		// TODO: update the peer on valid msgs only

/*		handler := handlerForMsgType(pmes.GetType())
		if handler == nil {
			s.Reset()
			return
		}

		rpmes, err := handler(ctx, mPeer, pmes)
		if err != nil {
			s.Reset()
			return
		}

		if rpmes == nil {
			continue
		}

		if err := w.WriteMsg(rpmes); err != nil {
			s.Reset()
			return
		}
		*/
	}
}

func handleStream(s net.Stream) {
	log.Println("Got a new stream!")
	ctx := context.Background() // TODO change to some timeout
	cr := ctxio.NewReader(ctx, s)
	cw := ctxio.NewWriter(ctx, s)
	r := ggio.NewDelimitedReader(cr, net.MessageSizeMax)
	w := ggio.NewDelimitedWriter(cw)
	go handleNewMessage(ctx, s, r, w)
}

/*
function sendRequest(ctx context.Context, p peer.ID, pmes *BHMessage) (*BHMessage, error)
{
	ms, err := messageSenderForPeer(p)
	if err != nil {
		return nil, err
	}

	rpmes, err := ms.SendRequest(ctx, pmes)
	if err != nil {
		return nil, err
	}

	return rpmes, nil
}

function sendMessage(ctx, context.Context, p peer.ID, pmes *BHMessage) error {
	ms, err := messageSenderForPeer(p)
	if err != nil {
		return err
	}

	if err := ms.SendMessage(ctx, pmes); err != nil {
		return err
	}

	return nil
}

func readData(rw *bufio.ReadWriter) {
	for {
		str, _ := rw.ReadString('\n')

		if str == "" {
			return
		}
		if str != "\n" {
			// Green console colour: 	\x1b[32m
			// Reset console colour: 	\x1b[0m
			fmt.Printf("\x1b[32m%s\x1b[0m> ", str)
		}

	}
}

func writeData(rw *bufio.ReadWriter) {
	stdReader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		sendData, err := stdReader.ReadString('\n')

		if err != nil {
			panic(err)
		}

		rw.WriteString(fmt.Sprintf("%s\n", sendData))
		rw.Flush()
	}

}
*/
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
	}

	if (!keyLoaded) {
		fmt.Println("Generating new private key ...")
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

func bhmain() {

	masterID := "QmSAdkJ5ZvLz4syZMox6WK4RUk1xVTQ56HLVkWDqctoiFR"
	master := "/ip4/142.93.16.125/tcp/5564/ipfs/"+masterID
	sourcePort := flag.Int("sp", 5564, "Source port number")
	flag.Parse()
	prvKey := loadPrivKey("key")
	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *sourcePort))
	host, err := libp2p.New(
		context.Background(),
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(prvKey),
	)
	g_ThisHost = host

	if err != nil {
		panic(err)
	}

	fmt.Println("--- start ----")
	if host.ID().Pretty() == masterID {
		fmt.Println("This is master", master)
		host.SetStreamHandler("/chat/1.0.0", handleStream)
		dumploop(host)
	} else {

		// Add destination peer multiaddress in the peerstore.
		// This will be used during connection and stream creation by libp2p.
		peerID := addAddrToPeerstore(host, master)

		fmt.Printf("My address: ")
		// IP will be 0.0.0.0 (listen on any interface) and port will be 0 (choose one for me).
		// Although this node will not listen for any connection. It will just initiate a connect with
		// one of its peer and use that stream to communicate.
		fmt.Printf("%s/ipfs/%s\n", sourceMultiAddr, host.ID().Pretty())

		// Start a stream with peer with peer Id: 'peerId'.
		// Multiaddress of the destination peer is fetched from the peerstore using 'peerId'.
		s, err := host.NewStream(context.Background(), peerID, "/chat/1.0.0")

		if err != nil {

			fmt.Println(err)

		} else {

			// Send ping message to server
			ctx := context.Background() // TODO change to some timeout
			cr := ctxio.NewReader(ctx, s)
			cw := ctxio.NewWriter(ctx, s)
			r := ggio.NewDelimitedReader(cr, net.MessageSizeMax)
			w := ggio.NewDelimitedWriter(cw)
			go handleNewMessage(ctx, s, r, w)

			for {
				time.Sleep(3 * time.Second)
				t := BhMessage_BH_PING
				pmes := &BhMessage {
					Type: &t,
				}
				fmt.Println("Sending BH_PING message to server")
				w.WriteMsg(pmes)
				// dump(host)
			}
		}
	}
}



func dump (h host.Host) {
	for i,p := range h.Peerstore().Peers() {
		fmt.Println(i,p, h.Peerstore().Addrs(p))
		for i,a := range h.Peerstore().Addrs(p) {
			fmt.Println(i, a)
		}
	}
}

func dumploop (h host.Host) {
//	for {
//		time.Sleep(3 * time.Second)
//		dump(h)
//	}

	stdReader := bufio.NewReader(os.Stdin)
	for {
		input, err := stdReader.ReadString('\n')
		if err != nil {
			panic(err)
		}

		if (input == "l\n") {
			dump(h)
			continue
		}
		if (input == "q\n") {
			fmt.Println("Quiting ...")
			os.Exit(0)
		} else {
			fmt.Print("Unknown command: "+input)
		}
	}
}

