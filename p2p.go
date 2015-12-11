package registry_p2p

const (
	MODE_IMAGE string = "image"
	MODE_LAYER string = "layer"
)

type Item struct {
	ID   string
	Type string //image, layer or metadata
	URL  string
}
