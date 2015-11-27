package bittorrent

type BitTorrent interface {
	CreateTorrent(path, torrentPath string, trackers []string) error
	Download(path, torrentPath string, configs map[string]string) error
}
