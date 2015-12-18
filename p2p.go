package registry_p2p

const (
	MODE_IMAGE string = "image"
	MODE_LAYER string = "layer"
)

type Item struct {
	ID       string
	ParentID string
	Type     string //layer or layer_meta
	URL      string
}
