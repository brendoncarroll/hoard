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
		{"tag_format", string(md.Format())},
		{"title", md.Title()},
		{"album", md.Album()},
		{"artist", md.Artist()},
		{"album_artist", md.AlbumArtist()},
		{"composer", md.Composer()},
		{"genre", md.Genre()},
	}

	trackN, _ := md.Track()
	if trackN > 0 {
		t1 = append(t1, Tag{"track", strconv.Itoa(trackN)})
	}

	for _, t := range t1 {
		if t.Value != "" {
			tags = append(tags, t)
		}
	}

	return tags, nil
}
