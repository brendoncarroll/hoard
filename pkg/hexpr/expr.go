package hexpr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"strings"

	"github.com/blobcache/glfs"
	"github.com/gotvc/got/pkg/gotfs"

	"github.com/brendoncarroll/hoard/pkg/hcorpus"
	"github.com/brendoncarroll/hoard/pkg/labels"
)

type GLFSExpr = glfs.Ref

func NewGLFS(ref glfs.Ref) Expr {
	return Expr{
		GLFS: &ref,
	}
}

type GotFSExpr struct {
	Root gotfs.Root `json:"root"`
	Path string     `json:"path"`
}

type ListExpr = []hcorpus.ID

type SetExpr = []hcorpus.ID

type QueryExpr = struct {
	Index string       `json:"index"`
	Query labels.Query `json:"query"`
}

type Expr struct {
	GLFS  *GLFSExpr  `json:"glfs,omitempty"`
	GotFS *GotFSExpr `json:"gotfs,omitempty"`

	List *ListExpr `json:"list,omitempty"`
	Set  *SetExpr  `json:"set,omitempty"`

	Query *QueryExpr `json:"query,omitempty"`
	Eval  *Expr      `json:"eval,omitempty"`
}

func Marshal(e Expr) []byte {
	data, err := json.Marshal(e)
	if err != nil {
		panic(err)
	}
	return data
}

func ParseExpr(x []byte) (*Expr, error) {
	var e Expr
	if err := json.Unmarshal(x, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

func (e Expr) IsMutable() bool {
	switch {
	case e.GLFS != nil:
		return false
	case e.GotFS != nil:
		return false
	case e.Query != nil:
		return true
	default:
		panic(e)
	}
}

type Value struct {
	Type string
	Data io.ReaderAt
}

func (v *Value) NewReader() io.ReadSeeker {
	return io.NewSectionReader(v.Data, 0, math.MaxInt64)
}

func (v *Value) ForEach(fn func(ID hcorpus.ID) error) error {
	switch {
	case v.Type == "List[ID]" || v.Type == "Set[ID]":
		var id hcorpus.ID
		r := v.NewReader()
		for {
			_, err := io.ReadFull(r, id[:])
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}
			if err := fn(id); err != nil {
				return err
			}
		}
		return nil
	case strings.HasPrefix(v.Type, "List"):
		fallthrough
	case strings.HasPrefix(v.Type, "Set"):
		r := io.NewSectionReader(v.Data, 0, math.MaxInt64)
		dec := json.NewDecoder(r)
		for dec.More() {
			var id hcorpus.ID
			if err := dec.Decode(&id); err != nil {
				return err
			}
			if err := fn(id); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("cannot iterate over value of type %v", v.Type)
	}
}

type Evaluator struct {
	GetExpr         func(context.Context, hcorpus.ID) (*Expr, error)
	GetQueryBackend func(string) labels.QueryBackend

	OpenGLFS  func(glfs.Ref) (io.ReaderAt, error)
	OpenGotFS func(gotfs.Root, string) (io.ReaderAt, error)
}

func (ev *Evaluator) Eval(ctx context.Context, x Expr) (*Value, error) {
	return ev.eval(ctx, hcorpus.Hash(Marshal(*x.Eval)), x)
}

func (ev *Evaluator) EvalID(ctx context.Context, id hcorpus.ID) (*Value, error) {
	e, err := ev.GetExpr(ctx, id)
	if err != nil {
		return nil, err
	}
	return ev.eval(ctx, id, *e)
}

func (ev *Evaluator) eval(ctx context.Context, id hcorpus.ID, x Expr) (*Value, error) {
	switch {
	case x.GLFS != nil:
		rc, err := ev.OpenGLFS(*x.GLFS)
		if err != nil {
			return nil, err
		}
		return &Value{Data: rc}, nil
	case x.GotFS != nil:
		// TODO: handle directories
		r, err := ev.OpenGotFS(x.GotFS.Root, x.GotFS.Path)
		if err != nil {
			return nil, err
		}
		return &Value{Data: r}, nil
	case x.Query != nil:
		qb := ev.GetQueryBackend(x.Query.Index)
		resultSet, err := labels.DoQuery(ctx, qb, x.Query.Query)
		if err != nil {
			return nil, err
		}
		rs := io.NewSectionReader(&idStream{ids: resultSet.IDs}, 0, int64(len(resultSet.IDs)*32))
		return &Value{Type: "List[ID]", Data: rs}, nil
	case x.Eval != nil:
		v, err := ev.Eval(ctx, *x.Eval)
		if err != nil {
			return nil, err
		}
		data, err := ioutil.ReadAll(io.NewSectionReader(v.Data, 0, math.MaxInt64))
		if err != nil {
			return nil, err
		}
		x2, err := ParseExpr(data)
		if err != nil {
			return nil, err
		}
		return ev.eval(ctx, hcorpus.Hash(data), *x2)
	default:
		return nil, errors.New("empty expression")
	}
}

type idStream struct {
	ids []hcorpus.ID
}

func (s *idStream) ReadAt(p []byte, offset int64) (n int, err error) {
	for i := int(offset) / 32; i < len(s.ids) && n < len(p); i++ {
		n2 := copy(p[n:], s.ids[i][:])
		n += n2
	}
	if n < len(p) {
		err = io.EOF
	}
	return n, err
}
