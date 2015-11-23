package p2p_lib

import (
	"p2p_lib/bittorrent"
)

func Download(client bittorrent.BitTorrent, torrents, configs map[string]string) (err error) {
	err = client.Download(torrents, configs)
	return
}
