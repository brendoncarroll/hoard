package hidx_audio

import (
	"context"
	"errors"
	"io"
	"strconv"

	"github.com/brendoncarroll/hoard/pkg/hexpr"
	"github.com/brendoncarroll/hoard/pkg/labels"
	dtag "github.com/dhowden/tag"
)

type Tag = labels.Pair

func IndexID3v1(ctx context.Context, e hexpr.Expr, cv hexpr.Value) ([]Tag, error) {
	if e.IsMutable() {
		return nil, nil
	}
	return ParseID3v1(nil, cv.NewReader())
}

func IndexID3v2(ctx context.Context, e hexpr.Expr, cv hexpr.Value) ([]Tag, error) {
	if e.IsMutable() {
		return nil, nil
	}
	return ParseID3v2(nil, cv.NewReader())
}

func IndexFLAC(ctx context.Context, e hexpr.Expr, cv hexpr.Value) ([]Tag, error) {
	if e.IsMutable() {
		return nil, nil
	}
	return ParseFLAC(nil, cv.NewReader())
}

func ParseID3v1(out []Tag, r io.ReadSeeker) ([]Tag, error) {
	md, err := dtag.ReadID3v1Tags(r)
	if err != nil {
		if errors.Is(err, dtag.ErrNotID3v1) || errors.Is(err, dtag.ErrNoTagsFound) {
			return nil, nil
		}
		return nil, err
	}
	return addID3(out, md)
}

func ParseID3v2(out []Tag, r io.ReadSeeker) ([]Tag, error) {
	md, err := dtag.ReadID3v2Tags(r)
	if err != nil {
		if errors.Is(err, dtag.ErrNoTagsFound) {
			return nil, nil
		}
		return nil, err
	}
	return addID3(out, md)
}

func addID3(out []Tag, md dtag.Metadata) ([]Tag, error) {
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
			out = append(out, t)
		}
	}
	return out, nil
}
