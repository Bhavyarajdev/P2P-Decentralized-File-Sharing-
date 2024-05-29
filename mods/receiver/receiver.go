package receiver

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mods/chunkrw"
	"mods/tracker"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

func group(totalChunks int) [][2]int {
	if totalChunks == 0 {
		return [][2]int{{0, 0}}
	}
	need := make([][2]int, 0)
	tgroups := totalChunks / 20
	// extra := totalChunks % 20
	// fmt.Println(tgroups)
	for i := 0; i < tgroups; i++ {
		need = append(need, [2]int{20 * i, 20*(i+1) - 1})
	}
	need = append(need, [2]int{20 * (tgroups), totalChunks})
	return need
}

func fileOpenCreate(name string, mode uint32, size int64) (path string) {
	path = "./receive/" + name
	flags := os.O_CREATE | os.O_TRUNC | os.O_WRONLY
	fMode := fs.FileMode(mode)
	file, err := os.OpenFile(path, flags, fMode)
	if err != nil {
		log.Fatal(err)
	}
	err = file.Truncate(size)
	if err != nil {
		log.Fatal(err)
	}
	file.Close()
	return path
}

// File share info for Peers
type FileShareInfo struct {
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	ChunkSize   uint   `json:"chunksize"`
	TotalChunks uint   `json:"totalchunks"`
	Mode        uint32 `json:"mode"`
}

// ProgressInfo for GUI
type ProgressInfo struct {
	ItemId    string `json:"itemId"`
	Status    string `json:"status"`
	BytesDown int64  `json:"bytesDown"`
	TSize     int64  `json:"tSize"`
	PeerNum   int    `json:"peerNum"`
}

type Receiver struct {
	FInfo     FileShareInfo
	Info      *ProgressInfo
	cWriter   *chunkrw.FileChunkWriter
	Needed    [][2]int
	Seeds     []string
	trackerIP string
}

func NewReceiver(fname, tIP string) (*Receiver, error) {
	trackinfo := GetSeeds(tIP, fname)
	// fmt.Println(trackinfo)
	if len(trackinfo.FSeeds) <= 0 {
		return nil, fmt.Errorf("no file")
	}
	path := fileOpenCreate(fname, trackinfo.FInfo.Mode, trackinfo.FInfo.Size)
	cW, _ := chunkrw.NewFileChunkWriter(path, trackinfo.FInfo.ChunkSize, trackinfo.FInfo.TotalChunks)
	return &Receiver{
		FInfo: FileShareInfo(trackinfo.FInfo),
		Info: &ProgressInfo{
			ItemId:    trackinfo.FInfo.Name,
			Status:    "Downloading",
			BytesDown: 0,
			TSize:     trackinfo.FInfo.Size,
			PeerNum:   0,
		},
		cWriter:   cW,
		Needed:    group(int(trackinfo.FInfo.TotalChunks)),
		Seeds:     trackinfo.FSeeds,
		trackerIP: tIP,
	}, nil
}

func GetSeeds(tIP string, fname string) tracker.TrackInfo {
	tAddr, _ := net.ResolveUDPAddr("udp4", tIP)
	buff := make([]byte, 2048)
	var trackinfo tracker.TrackInfo
	tConn, _ := net.DialUDP("udp4", nil, tAddr)
	tConn.Write([]byte("info:" + fname))
	n, err := tConn.Read(buff)
	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal(buff[:n], &trackinfo)
	fmt.Println(trackinfo)
	return trackinfo
}

func connReadBytes(conn net.TCPConn, nBytes int, rBuff []byte) (n int) {
	read := 0
	for read < nBytes {
		conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, err := conn.Read(rBuff[read:])
		if err != nil {
			if err == io.EOF {
				fmt.Println("Connection lost with", conn.RemoteAddr())
				break
			}
			if strings.Contains(err.Error(), "i/o timeout") {
				if read == 0 {
					continue
				}
				return read
			}
			log.Fatal(err)
		}
		// fmt.Printf("Read (%d)\n", n)
		read += n
	}
	return read
}

func (r *Receiver) commHandler(conn net.TCPConn, mux *sync.Mutex, wG *sync.WaitGroup) {
	for {
		mux.Lock()
		if len(r.Needed) <= 0 {
			mux.Unlock()
			break
		}
		need := (r.Needed)[0]
		(r.Needed) = (r.Needed)[1:]
		mux.Unlock()
		fmt.Println(r.Needed)
		for i := need[0]; i <= need[1]; i++ {
			rBuff := make([]byte, r.FInfo.ChunkSize)
			msg := fmt.Sprintf("chunk:%d", i)
			conn.Write([]byte(msg))
			n := connReadBytes(conn, int(r.cWriter.ChunkSize), rBuff)
			fmt.Printf("Write %d (%d)\n", i, n)
			r.cWriter.WriteChunk(uint(i), rBuff[:n])
			mux.Lock()
			r.Info.BytesDown += int64(n)
			mux.Unlock()
		}
	}
	mux.Lock()
	msg := "quit"
	r.Info.PeerNum -= 1
	mux.Unlock()
	conn.Write([]byte(msg))
	time.Sleep(1 * time.Second)
	conn.Close()
	wG.Done()
}

func (r *Receiver) StartDownload() {
	mutex := sync.Mutex{}
	wGroup := sync.WaitGroup{}
	for i := 0; i < len(r.Seeds); i++ {
		sAddr, _ := net.ResolveTCPAddr("tcp4", r.Seeds[i])
		conn, _ := net.DialTCP("tcp4", nil, sAddr)
		r.Info.PeerNum += 1
		wGroup.Add(1)
		fmt.Printf("Connected and downloading with: %s\n", sAddr.String())
		go r.commHandler(*conn, &mutex, &wGroup)
		if r.Info.PeerNum == 4 {
			break
		}
	}
	wGroup.Wait()
	r.Info.Status = "Downloaded"
}
