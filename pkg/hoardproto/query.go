package hoardproto

type QueryReq struct {
	HasTags map[string]string `json:"has_tags"`
	Limit   int               `json:"limit"`
	Hops    int               `json:"hops"`
}

type QueryRes struct {
	Manifests []Manifest `json:"manifests"`
}
