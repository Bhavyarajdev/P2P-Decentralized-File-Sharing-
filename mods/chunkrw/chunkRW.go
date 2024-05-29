package chunkrw

import (
	"fmt"
	"os"
	"sync"
)

type FileChunkReader struct {
	*os.File
	ChunkSize uint
	rMux      sync.Mutex
}

func NewFileChunkReader(path string, ChunkSize uint) (*FileChunkReader, error) {
	file, err := os.Open(path)
	if err != nil {
		return &FileChunkReader{}, err
	}
	return &FileChunkReader{
		File:      file,
		ChunkSize: ChunkSize,
	}, nil
}

func (cR *FileChunkReader) ReadChunk(num uint, buff []byte) (n int, err error) {
	if len(buff) < int(cR.ChunkSize) {
		return 0, fmt.Errorf("small buffer")
	}
	cR.rMux.Lock()
	_, err = cR.Seek(int64(cR.ChunkSize*num), 0)
	if err != nil {
		return 0, err
	}
	n, err = cR.Read(buff[:cR.ChunkSize])
	// fmt.Printf("Chunk %d: %s\n", num, string(buff))
	cR.rMux.Unlock()
	return n, err
}

type FileChunkWriter struct {
	*os.File
	ChunkSize   uint
	TotalChunks uint
	wMux        sync.Mutex
}

func NewFileChunkWriter(path string, ChunkSize uint, total uint) (*FileChunkWriter, error) {
	flags := os.O_WRONLY
	file, err := os.OpenFile(path, flags, os.FileMode(0600))
	if err != nil {
		return &FileChunkWriter{}, err
	}
	return &FileChunkWriter{
		File:        file,
		ChunkSize:   ChunkSize,
		TotalChunks: total,
	}, nil
}

func (cW *FileChunkWriter) WriteChunk(num uint, buff []byte) (err error) {
	writeLen := len(buff)
	if writeLen > int(cW.ChunkSize) {
		writeLen = int(cW.ChunkSize)
	}
	cW.wMux.Lock()
	_, err = cW.Seek(int64(cW.ChunkSize*num), 0)
	if err != nil {
		return err
	}
	_, err = cW.Write(buff[:writeLen])
	
	cW.wMux.Unlock()
	return err
}
