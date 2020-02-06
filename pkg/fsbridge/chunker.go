package fsbridge

import (
	"io"
)

type Chunker interface {
	ForEachChunk(func(Chunk) error) error
}

type Chunk struct {
	Offset int64
	Data   []byte
}

type FixedSizeChunker struct {
	r    io.Reader
	size int
}

func NewFixedSizeChunker(r io.Reader, chunkSize int) *FixedSizeChunker {
	if chunkSize == 0 {
		panic("cannot make 0 sized chunks")
	}
	return &FixedSizeChunker{r: r, size: chunkSize}
}

func (c *FixedSizeChunker) ForEachChunk(fn func(Chunk) error) error {
	buf := make([]byte, c.size)
	offset := int64(0)

	for {
		n, err := io.ReadFull(c.r, buf)
		switch err {
		case nil, io.EOF, io.ErrUnexpectedEOF:
		default:
			return err
		}
		chunk := Chunk{
			Offset: offset,
			Data:   buf[:n],
		}
		if err2 := fn(chunk); err2 != nil {
			return err
		}

		offset += int64(n)
		switch err {
		case io.EOF, io.ErrUnexpectedEOF:
			return nil
		}
	}
}
