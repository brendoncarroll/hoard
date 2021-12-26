package taggers

import (
	"io"

	"github.com/brendoncarroll/hoard/pkg/tagging"
)

type Tag = tagging.Tag

type TagSet = tagging.TagSet

type TagFunc func(r io.ReadSeeker, tags []Tag) ([]Tag, error)

func SuggestTags(r io.ReadSeeker, tags TagSet) error {
	tfs := []TagFunc{
		ParseCommonAudio,
		ParseFLAC,
	}

	stagingTags := []Tag{}
	for _, tf := range tfs {
		var err error
		if _, err := r.Seek(0, io.SeekStart); err != nil {
			return err
		}
		stagingTagsNew, err := tf(r, stagingTags)
		if err == nil {
			stagingTags = stagingTagsNew
		}
	}

	for _, t := range stagingTags {
		tags[t.Key] = t.Value
	}
	return nil
}
