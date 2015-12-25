package registry_p2p

const (
	MODE_IMAGE       string = "image"
	MODE_LAYER       string = "layer"
	MODE_MULTI_LAYER string = "multi_layer"
)

type Item struct {
	ID       string
	ParentID string
	Type     string //layer, multi_layer
	URL      string
	LayerNum int
}
