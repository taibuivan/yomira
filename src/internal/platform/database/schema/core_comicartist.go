package schema

// ComicArtistTable represents the 'core.comicartist' table
type ComicArtistTable struct {
	Table    string
	ComicID  string
	ArtistID string
}

// ComicArtist is the schema definition for core.comicartist
var ComicArtist = ComicArtistTable{
	Table:    "core.comicartist",
	ComicID:  "comicid",
	ArtistID: "artistid",
}
