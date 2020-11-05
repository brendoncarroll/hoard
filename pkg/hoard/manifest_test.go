package hoard

import (
	"encoding/json"
	"testing"

	"github.com/brendoncarroll/hoard/pkg/hoardfile"
	"github.com/brendoncarroll/hoard/pkg/hoardproto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshal(t *testing.T) {
	mf := Manifest{
		Manifest: hoardproto.Manifest{
			File: hoardfile.File{},
		},
	}
	data, err := json.Marshal(mf)
	t.Log(string(data))
	require.Nil(t, err)

	mf2 := Manifest{}
	err = json.Unmarshal(data, &mf2)
	require.Nil(t, err)

	assert.Equal(t, mf, mf2)
}
