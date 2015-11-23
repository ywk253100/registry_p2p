package bittorrent

type BitTorrent interface {
	CreateTorrent(path, torrentPath string, trackers []string) error
	Download(torrents, configs map[string]string) error
}
