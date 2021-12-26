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

type OID = hcorpus.Fingerprint

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

func (o *Operator) AddTags(ctx context.Context, s cadata.Store, root Root, fp OID, tags []taggers.Tag) (*Root, error) {
	muts := make([]gotkv.Mutation, 2*len(tags))
	for i, tag := range tags {
		if err := checkTag(tag); err != nil {
			return nil, err
		}
		forwardEnt := makeForwardEntry(tag, fp)
		muts[2*i] = gotkv.Mutation{
			Span:    gotkv.SingleKeySpan(forwardEnt.Key),
			Entries: []gotkv.Entry{forwardEnt},
		}
		inverseEnt := makeInverseEntry(tag, fp)
		muts[2*i+1] = gotkv.Mutation{
			Span:    gotkv.SingleKeySpan(inverseEnt.Key),
			Entries: []gotkv.Entry{inverseEnt},
		}
	}
	sort.Slice(muts, func(i, j int) bool {
		return bytes.Compare(muts[i].Span.Start, muts[j].Span.Start) < 0
	})
	return o.gotkv.Mutate(ctx, s, root, muts...)
}

func (o *Operator) GetTags(ctx context.Context, s cadata.Store, root Root, oid OID) (ret []tagging.Tag, _ error) {
	span := gotkv.PrefixSpan(makeForwardKey(nil, oid, nil))
	if err := o.gotkv.ForEach(ctx, s, root, span, func(ent gotkv.Entry) error {
		_, key, value, err := parseForwardEntry(ent)
		if err != nil {
			return err
		}
		ret = append(ret, tagging.Tag{
			Key:   string(key),
			Value: append([]byte{}, value...),
		})
		return nil
	}); err != nil {
		return nil, err
	}
	return ret, nil
}

func (o *Operator) GetTagValue(ctx context.Context, s cadata.Store, root Root, fp OID, tagKey string) ([]byte, error) {
	key := makeForwardKey(nil, fp, []byte(tagKey))
	return o.gotkv.Get(ctx, s, gotkv.Root(root), key)
}

func (o *Operator) ForEach(ctx context.Context, s cadata.Store, root Root, fn func(OID, []tagging.Tag) error) error {
	span := prefixSpan(gotkv.TotalSpan(), []byte{'f', 0x00})
	var currentFP OID
	var tags []tagging.Tag
	return o.gotkv.ForEach(ctx, s, root, span, func(ent gotkv.Entry) error {
		fp, key, value, err := parseForwardEntry(ent)
		if err != nil {
			return err
		}
		if fp != currentFP {
			if currentFP != (OID{}) {
				if err := fn(currentFP, tags); err != nil {
					return err
				}
			}
			currentFP = fp
			tags = tags[:0]
		}
		tags = append(tags, tagging.Tag{
			Key:   string(key),
			Value: append([]byte{}, value...),
		})
		return nil
	})
}

func (o *Operator) ForEachTagKey(ctx context.Context, s cadata.Store, root Root, fn func(string) error) error {
	span := prefixSpan(gotkv.TotalSpan(), []byte{'i', 0x00})
	var lastKey []byte
	return o.gotkv.ForEach(ctx, s, root, span, func(ent gotkv.Entry) error {
		_, key, _, err := parseInverseEntry(ent)
		if err != nil {
			return err
		}
		if !bytes.Equal(key, lastKey) {
			lastKey = append(lastKey[:0], key...)
			if err := fn(string(key)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (o *Operator) ForEachTagValue(ctx context.Context, s cadata.Store, root Root, tagKey string, fn func([]byte) error) error {
	prefix := []byte{'i', 0x00}
	prefix = append(prefix, tagKey...)
	prefix = append(prefix, 0x00)
	span := gotkv.PrefixSpan(prefix)
	return o.gotkv.ForEach(ctx, s, root, span, func(ent gotkv.Entry) error {
		_, _, value, err := parseInverseEntry(ent)
		if err != nil {
			return err
		}
		return fn(value)
	})
}

func (o *Operator) Search(ctx context.Context, s cadata.Store, root Root, query tagging.Query) (*tagging.ResultSet, error) {
	qb := o.NewQueryBackend(s, root)
	return tagging.DoQuery(ctx, qb, query)
}

func checkTag(t tagging.Tag) error {
	if strings.Contains(t.Key, "\x00") {
		return errors.Errorf("tag key cannot contain NULL byte")
	}
	if bytes.Contains(t.Value, []byte("\x00")) {
		return errors.Errorf("tag value cannot contain NULL byte")
	}
	return nil
}

func makeForwardKey(out []byte, fp OID, tagKey []byte) []byte {
	out = append(out, 'f')
	out = append(out, 0x00)
	out = append(out, fp[:]...)
	out = append(out, tagKey...)
	return out
}

func makeForwardEntry(tag tagging.Tag, fp OID) gotkv.Entry {
	return gotkv.Entry{
		Key:   makeForwardKey(nil, fp, []byte(tag.Key)),
		Value: []byte(tag.Value),
	}
}

func parseForwardKey(x []byte) ([]byte, OID, error) {
	parts := bytes.SplitN(x, []byte{0x00}, 2)
	if len(parts) != 2 {
		return nil, OID{}, errors.Errorf("invalid forward key: %q", x)
	}
	dirBytes := parts[0]
	if len(dirBytes) != 1 || dirBytes[0] != 'f' {
		return nil, OID{}, errors.Errorf("incorrect key direction identifier: %q", dirBytes)
	}
	if len(parts[1]) < len(OID{}) {
		return nil, OID{}, errors.Errorf("too short to be fingerprint")
	}
	fp := hcorpus.FPFromBytes(parts[1])
	tagKeyBytes := parts[1][32:]
	return tagKeyBytes, fp, nil
}

func parseForwardEntry(ent gotkv.Entry) (OID, []byte, []byte, error) {
	key, fp, err := parseForwardKey(ent.Key)
	if err != nil {
		return OID{}, nil, nil, err
	}
	return fp, key, ent.Value, nil
}

func makeInverseKey(out []byte, tag tagging.Tag, fp OID) []byte {
	out = append(out, 'i')
	out = append(out, 0x00)
	out = append(out, []byte(tag.Key)...)
	out = append(out, 0x00)
	out = append(out, []byte(tag.Value)...)
	out = append(out, 0x00)
	out = append(out, fp[:]...)
	return out
}

func makeInverseEntry(tag tagging.Tag, fp OID) gotkv.Entry {
	return gotkv.Entry{
		Key:   makeInverseKey(nil, tag, fp),
		Value: fp[:],
	}
}

func parseInverseKey(x []byte) (key, value []byte, _ *OID, _ error) {
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
	fp := hcorpus.FPFromBytes(fpBytes)
	return tagKey, tagValue, &fp, nil
}

func parseInverseEntry(ent gotkv.Entry) (_ *OID, key, value []byte, _ error) {
	key, value, fp, err := parseInverseKey(ent.Key)
	if err != nil {
		return nil, nil, nil, err
	}
	return fp, key, value, nil
}
