package main
import (
	"context"
	"flag"
	"fmt"
	"io"
	mrand "math/rand"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-crypto"
	"github.com/multiformats/go-multiaddr"
)

func main1() {
	sourcePort := flag.Int("sp", 8964, "Source port number")
	flag.Parse()

	var r io.Reader
	r = mrand.New(mrand.NewSource(int64(*sourcePort)))
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)

	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *sourcePort))
	host, err := libp2p.New(
		context.Background(),
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(prvKey),
	)

	if err != nil {
		panic(err)
	}

	var d *dht.IpfsDHT
	d, err = dht.New(context.Background(), host)

	fmt.Println(d)
	fmt.Printf("DHT created @/ip4/0.0.0.0/tcp/%d/ipfs/%s", *sourcePort, host.ID().Pretty())
	select {}
}

