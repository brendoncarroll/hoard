package hindex

import (
	"bytes"
	"context"
	"sort"
	"strings"

	"github.com/brendoncarroll/go-state/cadata"
	"github.com/gotvc/got/pkg/gotfs"
	"github.com/gotvc/got/pkg/gotkv"
	"github.com/pkg/errors"

	"github.com/brendoncarroll/hoard/pkg/hcorpus"
	"github.com/brendoncarroll/hoard/pkg/taggers"
	"github.com/brendoncarroll/hoard/pkg/tagging"
)

type Fingerprint = hcorpus.Fingerprint

// Root is the root of an index
// The index is structured like this:
/*
f/
	<tag_key>/
		<entity> -> <tag_value>
		<entity> -> <tag_value>
		...
	<tag_key2>/
		...
i/
	<tag_key>/
		<tag_value><entity> -> ""
		<tag_value><entity> -> ""
		...
	<tag_key2>/
		...
*/
type Root = gotkv.Root

type Operator struct {
	gotkv gotkv.Operator
}

func New() *Operator {
	return &Operator{
		gotkv: gotkv.NewOperator(
			gotkv.WithAverageSize(gotfs.DefaultAverageBlobSizeMetadata),
			gotkv.WithMaxSize(1<<16),
		),
	}
}

func (o *Operator) NewQueryBackend(s cadata.Store, root Root) QueryBackend {
	return QueryBackend{
		op:   o,
		s:    s,
		root: root,
	}
}

func (o *Operator) NewEmpty(ctx context.Context, s cadata.Store) (*Root, error) {
	return o.gotkv.NewEmpty(ctx, s)
}

func (o *Operator) AddTags(ctx context.Context, s cadata.Store, root Root, fp Fingerprint, tags []taggers.Tag) (*Root, error) {
	b := o.gotkv.NewBuilder(s)
	sort.SliceStable(tags, func(i, j int) bool {
		return tags[i].Key < tags[j].Key
	})
	var lastKey []byte
	for i, tag := range tags {
		if err := checkTag(tag); err != nil {
			return nil, err
		}
		forwardEnt := makeForwardEntry(tag, fp)
		// before
		var start []byte
		if i > 0 {
			start = gotkv.KeyAfter(lastKey)
		}
		it := o.gotkv.NewIterator(s, root, gotkv.Span{
			Start: start,
			End:   forwardEnt.Key,
		})
		if err := gotkv.CopyAll(ctx, b, it); err != nil {
			return nil, err
		}
		// insert
		if err := b.Put(ctx, forwardEnt.Key, forwardEnt.Value); err != nil {
			return nil, err
		}
		lastKey = forwardEnt.Key
	}
	for _, tag := range tags {
		inverseEnt := makeInverseEntry(tag, fp)
		// before
		start := gotkv.KeyAfter(lastKey)
		beforeIt := o.gotkv.NewIterator(s, root, gotkv.Span{
			Start: start,
			End:   inverseEnt.Key,
		})
		if err := gotkv.CopyAll(ctx, b, beforeIt); err != nil {
			return nil, err
		}
		// insert
		if err := b.Put(ctx, inverseEnt.Key, inverseEnt.Value); err != nil {
			return nil, err
		}
		lastKey = inverseEnt.Key
	}
	// after
	afterIt := o.gotkv.NewIterator(s, root, gotkv.Span{
		Start: gotkv.KeyAfter(lastKey),
		End:   nil,
	})
	if err := gotkv.CopyAll(ctx, b, afterIt); err != nil {
		return nil, err
	}
	return b.Finish(ctx)
}

func (o *Operator) Search(ctx context.Context, s cadata.Store, root Root, query tagging.Query) (*tagging.ResultSet, error) {
	qb := o.NewQueryBackend(s, root)
	return tagging.DoQuery(ctx, qb, query)
}

func checkTag(t tagging.Tag) error {
	if strings.Contains(t.Key, "\x00") {
		return errors.Errorf("tag key cannot contain NULL byte")
	}
	if strings.Contains(t.Value, "\x00") {
		return errors.Errorf("tag value cannot contain NULL byte")
	}
	return nil
}

func makeForwardKey(out []byte, tagKey []byte, fp Fingerprint) []byte {
	out = append(out, 'f')
	out = append(out, 0x00)
	out = append(out, tagKey...)
	out = append(out, 0x00)
	out = append(out, fp[:]...)
	return out
}

func makeForwardEntry(tag tagging.Tag, fp Fingerprint) gotkv.Entry {
	return gotkv.Entry{
		Key:   makeForwardKey(nil, []byte(tag.Key), fp),
		Value: []byte(tag.Value),
	}
}

func parseForwardKey(x []byte) ([]byte, Fingerprint, error) {
	parts := bytes.SplitN(x, []byte{0x00}, 3)
	if len(parts) != 3 {
		return nil, Fingerprint{}, errors.Errorf("invalid forward key: %q", x)
	}
	dirBytes := parts[0]
	tagKeyBytes := parts[1]
	fpBytes := parts[2]
	if len(dirBytes) != 1 || dirBytes[0] != 'f' {
		return nil, Fingerprint{}, errors.Errorf("incorrect key direction identifier: %q", dirBytes)
	}
	fp := Fingerprint{}
	if n := copy(fp[:], fpBytes); n < len(fp) {
		return nil, Fingerprint{}, errors.Errorf("too short to be fingerprint")
	}
	return tagKeyBytes, fp, nil
}

func parseForwardEntry(ent gotkv.Entry) (Fingerprint, []byte, []byte, error) {
	key, fp, err := parseForwardKey(ent.Key)
	if err != nil {
		return Fingerprint{}, nil, nil, err
	}
	return fp, key, ent.Value, nil
}

func makeInverseKey(out []byte, tag tagging.Tag, fp Fingerprint) []byte {
	out = append(out, 'i')
	out = append(out, 0x00)
	out = append(out, []byte(tag.Key)...)
	out = append(out, 0x00)
	out = append(out, []byte(tag.Value)...)
	out = append(out, 0x00)
	out = append(out, fp[:]...)
	return out
}

func makeInverseEntry(tag tagging.Tag, fp Fingerprint) gotkv.Entry {
	return gotkv.Entry{
		Key:   makeInverseKey(nil, tag, fp),
		Value: fp[:],
	}
}

func parseInverseKey(x []byte) (key, value []byte, _ *Fingerprint, _ error) {
	parts := bytes.SplitN(x, []byte{0x00}, 4)
	if len(parts) != 4 {
		return nil, nil, nil, errors.Errorf("invalid inverse key: %q", x)
	}
	dirBytes := parts[0]
	tagKey := parts[1]
	tagValue := parts[2]
	fpBytes := parts[3]
	if len(dirBytes) != 1 || dirBytes[0] != 'i' {
		return nil, nil, nil, errors.Errorf("incorrect key direction identifier: %q", dirBytes)
	}
	fp := Fingerprint{}
	if n := copy(fp[:], fpBytes); n < len(fp) {
		return nil, nil, nil, errors.Errorf("too short to be fingerprint")
	}
	return tagKey, tagValue, &fp, nil
}
