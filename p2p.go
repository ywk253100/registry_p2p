package registry_p2p

const (
	ImageMode string = "image"
	LayerMode string = "layer"
)

type Item struct {
	ID   string
	Type string //image, layer or metadata
	URL  string
}
