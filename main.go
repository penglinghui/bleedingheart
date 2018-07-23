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

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p-host"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-peer"
	"github.com/libp2p/go-libp2p-peerstore"
	"github.com/multiformats/go-multiaddr"
)

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

func handleStream(s net.Stream) {
	log.Println("Got a new stream!")

	// Create a buffer stream for non blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

	go readData(rw)
	go writeData(rw)

	// stream 's' will stay open until you close it (or the other side closes it).
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

func main() {

	masterID := "QmSmz6YKmwWnMFqz1ugvm1Ban5sco8bor5VW7nUVUYdLyr"
	master := "/ip4/178.128.12.42/tcp/8964/ipfs/QmSmz6YKmwWnMFqz1ugvm1Ban5sco8bor5VW7nUVUYdLyr"
	sourcePort := flag.Int("sp", 8964, "Source port number")
	flag.Parse()
	prvKey := loadPrivKey("key")
	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *sourcePort))
	host, err := libp2p.New(
		context.Background(),
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(prvKey),
	)

	if err != nil {
		panic(err)
	}

	if host.ID().Pretty() == masterID {
		fmt.Println("This is master", master)
		host.SetStreamHandler("/chat/1.0.0", handleStream)
		for {
			time.Sleep(3 * time.Second)
			for i,p := range host.Peerstore().Peers() {
				fmt.Println(i,p, host.Peerstore().PeerInfo(p))
			}

		}
		// Hang forever
		<-make(chan struct{})

	} else {

		// Add destination peer multiaddress in the peerstore.
		// This will be used during connection and stream creation by libp2p.
		peerID := addAddrToPeerstore(host, master)

		fmt.Println("This node's multiaddress: ")
		// IP will be 0.0.0.0 (listen on any interface) and port will be 0 (choose one for me).
		// Although this node will not listen for any connection. It will just initiate a connect with
		// one of its peer and use that stream to communicate.
		fmt.Printf("%s/ipfs/%s\n", sourceMultiAddr, host.ID().Pretty())

		// Start a stream with peer with peer Id: 'peerId'.
		// Multiaddress of the destination peer is fetched from the peerstore using 'peerId'.
		s, err := host.NewStream(context.Background(), peerID, "/chat/1.0.0")

		if err != nil {
			// panic(err)
			fmt.Println(err)

			if err := host.Connect(context.Background(), host.Peerstore().PeerInfo(peerID)); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("Connected to ", peerID)
			}
		} else {

			// Create a buffered stream so that read and writes are non blocking.
			rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

			// Create a thread to read and write data.
			go writeData(rw)
			go readData(rw)
		}



		for {
			time.Sleep(3 * time.Second)
			for i,p := range host.Peerstore().Peers() {
				fmt.Println(i,p, host.Peerstore().PeerInfo(p))
			}

		}
		// Hang forever.
		select {}

	}
}
