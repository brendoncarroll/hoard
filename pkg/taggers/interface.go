package taggers

import "io"

type Tag struct {
	Key, Value string
}

type TagSet map[string]string

type TagFunc func(r io.ReadSeeker, tags []Tag) ([]Tag, error)

func SuggestTags(r io.ReadSeeker, tags TagSet) error {
	tfs := []TagFunc{
		ParseCommonAudio,
		ParseFLAC,
	}

	stagingTags := []Tag{}
	for _, tf := range tfs {
		var err error
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
