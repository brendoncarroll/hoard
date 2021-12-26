package taggers

import (
	"fmt"
	"io"
	"strings"

	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/meta"
)

func ParseFLAC(r io.ReadSeeker, tags []Tag) ([]Tag, error) {
	stream, err := flac.Parse(r)
	if err != nil {
		return nil, err
	}

	// Stream info
	tags = append(tags, []Tag{
		{"bits_per_sample", []byte(fmt.Sprint(stream.Info.BitsPerSample))},
		{"channels", []byte(fmt.Sprint(stream.Info.NChannels))},
		{"sample_rate", []byte(fmt.Sprint(stream.Info.SampleRate))},
	}...)

	// Tags
	for _, block := range stream.Blocks {
		switch block.Body.(type) {
		case *meta.VorbisComment:
			vc := block.Body.(*meta.VorbisComment)
			tags = fromVorbis(vc, tags)
		}
	}

	return tags, nil
}

func fromVorbis(comment *meta.VorbisComment, tags []Tag) []Tag {
	for _, vtag := range comment.Tags {
		key := vtag[0]
		value := []byte(vtag[1])
		key = strings.ToLower(key)
		tags = append(tags, Tag{Key: key, Value: value})
	}
	return tags
}
