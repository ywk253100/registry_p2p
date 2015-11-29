package agent

type DownloadTask struct {
	ImageName    string
	ImageID      string
	TmpDir       string
	ImageTarPath string
	Torrents     map[string]string
	Mode         string
}
