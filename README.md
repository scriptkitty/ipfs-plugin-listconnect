# ipfs-plugin-listconnect. A plugin for IPFS which periodically connects to a given list of peers.

## Behavior

This plugin will read a ```list-connect.peers``` file (to be placed in the same directory as where ```ipfs daemon``` is started) and connect to every peer in that file.
Each line should be either a multiaddr or a peerID, the IDs will periodically resolved via the DHT.
Example:

    /ip4/78.47.197.225/tcp/4001/p2p/12D3KooWKQow8TEU4aqjVu7sZhXhWfdvQr6gAt3QgjPsxZZkEFhH
    QmcfFmzSDVbwexQ9Au2pt5YEXHK5xajwgaU6PpkbLWerMa
    ...
  
To build the plugin you have to set ```export IPFS_VERSION=<absolute path of IPFS source>``` you want to bind agains, then ```make build``` and ```make install```.
Alternatively, simply build and move the resulting ```listconnect.so``` to your IPFS plugin path, per default ```~/.ipfs/plugins``` (create the plugins directoy if it doesn't exist).
