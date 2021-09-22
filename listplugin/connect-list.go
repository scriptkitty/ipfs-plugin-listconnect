package listplugin

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	core "github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/plugin"
	peer "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

const (
	connectTimeout = 5 * time.Second
	idleInterval   = 10 * time.Second
	lookupTimeout  = 10 * time.Second
	lookupInterval = 120 * time.Second
	peerFilePath   = "list-connect.peers"
)

type ConnectPlugin struct {
	api       *core.IpfsNode
	peerIDMap map[peer.ID]peer.AddrInfo
}

var _ plugin.PluginDaemonInternal = (*ConnectPlugin)(nil)

// We have to satisfy the plugin.Plugin interface, so we need a Name(), Version(), and Init()
func (*ConnectPlugin) Name() string {
	return "connect-to-list-plugin"
}

func (*ConnectPlugin) Version() string {
	return "0.0.1"
}

func (*ConnectPlugin) Init(env *plugin.Environment) error {
	return nil
}

func (cp *ConnectPlugin) Start(ipfsInstance *core.IpfsNode) error {
	fmt.Println("Connect List Plugin started")
	cp.peerIDMap = make(map[peer.ID]peer.AddrInfo)
	cp.api = ipfsInstance
	go cp.forkToBackground()
	return nil
}

// Actual logic goes here. First, we need a wrapper around the Connect() function which takes a list of peerAddrs
func (cp *ConnectPlugin) connectToAll(pctx context.Context) {
	// Connect to the list of peers. Hopefully Connect() skips the ones that we are already connect to
	ctx, cancelFunc := context.WithTimeout(pctx, connectTimeout)
	defer cancelFunc()
	for _, pinfo := range cp.peerIDMap {
		if len(pinfo.Addrs) > 0 {
			fmt.Printf("Connecting to %s\n", pinfo)
			err := cp.api.PeerHost.Connect(ctx, pinfo)
			if err != nil {
				fmt.Println(err)
			}
		}

	}
}

func (cp *ConnectPlugin) readPeersFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// read each line. We can deal with peer IDs or multiaddresses. The latter we just completely enter into the map,
	// the former we enter as well but perform DHT lookups for them lateron
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "/") {
			// It's a multiaddress
			fmt.Printf("Found multiaddress %s\n", line)
			pinfo, err := peer.AddrInfoFromString(scanner.Text())
			if err != nil {
				fmt.Println(err)
				continue
			}
			if _, ok := cp.peerIDMap[pinfo.ID]; !ok {
				cp.peerIDMap[pinfo.ID] = *pinfo
			}

		} else {
			// It's a node ID
			peerID, err := peer.Decode(line)
			if err != nil {
				fmt.Println(err)
				continue
			}
			if _, ok := cp.peerIDMap[peerID]; !ok {
				cp.peerIDMap[peerID] = peer.AddrInfo{
					ID:    peerID,
					Addrs: make([]ma.Multiaddr, 0),
				}
			}

		}

	}

	return nil
}

func (cp *ConnectPlugin) lookupPeerIDs(bctx context.Context) {
	for pid, mapinfo := range cp.peerIDMap {
		if len(mapinfo.Addrs) == 0 {
			ctx, cfunc := context.WithTimeout(bctx, lookupTimeout)
			defer cfunc()
			fmt.Printf("Lookup of %s\n", pid)
			pinfo, err := cp.api.DHTClient.FindPeer(ctx, pid)
			if err != nil {
				fmt.Println(err)
				continue
			}
			cp.peerIDMap[pid] = pinfo
		}
	}
}

// Periodically wake up, read peer multiaddrs from file and try to connect to them.
// Workaround until we have a proper RPC support
func (cp *ConnectPlugin) forkToBackground() {
	bctx := context.Background()

	// Read the peer file once
	err := cp.readPeersFromFile(peerFilePath)
	if err != nil {
		fmt.Println(err)
	}

	connectTimer := time.NewTimer(idleInterval)
	// Initially wait less for the first DHT lookup
	dhtLookupTimer := time.NewTimer(20 * time.Second)
	for {
		select {

		case <-connectTimer.C:
			fmt.Println("Connecting to all nodes with multiaddr...")
			cp.connectToAll(bctx)
			connectTimer.Reset(idleInterval)

		case <-dhtLookupTimer.C:
			go func() {
				fmt.Println("Reading peers from file & Performing DHT lookup...")
				err := cp.readPeersFromFile(peerFilePath)
				if err != nil {
					fmt.Println(err)
				}
				cp.lookupPeerIDs(bctx)
				dhtLookupTimer.Reset(lookupInterval)
			}()
		}
		// dir, _ := os.Getwd()
		// fmt.Println(dir)
	}

}
