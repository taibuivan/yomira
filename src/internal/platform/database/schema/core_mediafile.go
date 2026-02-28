package schema

// CoreMediaFileTable represents the 'core.mediafile' table
type CoreMediaFileTable struct {
	Table         string
	ID            string
	StorageBucket string
	StorageKey    string
	SHA256        string
	SizeBytes     string
	MimeType      string
	CreatedAt     string
}

// CoreMediaFile is the schema definition for core.mediafile
var CoreMediaFile = CoreMediaFileTable{
	Table:         "core.mediafile",
	ID:            "id",
	StorageBucket: "storagebucket",
	StorageKey:    "storagekey",
	SHA256:        "sha256",
	SizeBytes:     "sizebytes",
	MimeType:      "mimetype",
	CreatedAt:     "createdat",
}

func (t CoreMediaFileTable) Columns() []string {
	return []string{
		t.ID, t.StorageBucket, t.StorageKey, t.SHA256, t.SizeBytes, t.MimeType, t.CreatedAt,
	}
}
