package taggers

import (
	"io"
	"strconv"

	dtag "github.com/dhowden/tag"
)

func ParseCommonAudio(r io.ReadSeeker, tags []Tag) ([]Tag, error) {
	md, err := dtag.ReadFrom(r)
	if err != nil {
		return nil, err
	}

	t1 := []Tag{
		{"tag_format", []byte(md.Format())},
		{"title", []byte(md.Title())},
		{"album", []byte(md.Album())},
		{"artist", []byte(md.Artist())},
		{"album_artist", []byte(md.AlbumArtist())},
		{"composer", []byte(md.Composer())},
		{"genre", []byte(md.Genre())},
	}

	trackN, _ := md.Track()
	if trackN > 0 {
		t1 = append(t1, Tag{"track", []byte(strconv.Itoa(trackN))})
	}
	for _, t := range t1 {
		if len(t.Value) != 0 {
			tags = append(tags, t)
		}
	}

	return tags, nil
}
