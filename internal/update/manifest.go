package update

type Artifact struct {
	URL       string `json:"url"`
	Size      int64  `json:"size"`
	SHA256    string `json:"sha256"`
	Signature string `json:"signature,omitempty"`
}

type Manifest struct {
	SchemaVersion int                 `json:"schema_version"`
	Channel       string              `json:"channel"`
	Version       string              `json:"version"`
	ReleaseDate   string              `json:"release_date"`
	KeyID         string              `json:"key_id"`
	Changelog     string              `json:"changelog,omitempty"`
	Artifacts     map[string]Artifact `json:"artifacts,omitempty"`
}
