package schema

// CoreComicRelationTable represents the 'core.comicrelation' table
type CoreComicRelationTable struct {
	Table        string
	FromComicID  string
	ToComicID    string
	RelationType string
}

// CoreComicRelation is the schema definition for core.comicrelation
var CoreComicRelation = CoreComicRelationTable{
	Table:        "core.comicrelation",
	FromComicID:  "fromcomicid",
	ToComicID:    "tocomicid",
	RelationType: "relationtype",
}
