package hoard

type VolumeSpec struct {
	Cell  CellSpec
	Store StoreSpec
}

type CellSpec struct {
	HTTP *HTTPCellSpec `json:"http,omitempty"`
}

type HTTPCellSpec struct {
	URL string
}

type StoreSpec struct {
	LocalDir *string `json:"local_dir,omitempty"`
}
