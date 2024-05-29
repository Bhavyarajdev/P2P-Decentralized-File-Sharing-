package seeder

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"mods/chunkrw"
	// "mods/tracker"
	"net"
	"os"
	"strconv"
	"strings"
	// "time"
)

func getLocalIP() string {
	// All Address for this device
	localIP := ""
	addrs, _ := net.InterfaceAddrs()
	for i, j := 0, len(addrs)-1; i < j; i, j = i+1, j-1 {
		addrs[i], addrs[j] = addrs[j], addrs[i]
	}
	
	for _, addr := range addrs {
		// fmt.Println(addr)
		ipnet, ok := addr.(*net.IPNet)
		// fmt.Println(ok, ipnet.IP.String())
		if ok && !ipnet.IP.IsLoopback() {
			ipv4 := ipnet.IP.To4()
			if !ipv4.IsPrivate() && ipv4 != nil {
				break
			}
			if ipv4 != nil {
				localIP = ipnet.IP.String()
				break
			}
		}
	}
	return localIP
}

func getListener(myIP string) (*net.TCPListener, string, error) {
	rNum := rand.Intn(1000) + 50000
	randPort := fmt.Sprintf(":%d", rNum)
	me, err := net.ResolveTCPAddr("tcp4", myIP+randPort)
	if err != nil {
		log.Fatal(err)
	}
	listener, err := net.ListenTCP("tcp4", me)
	if err != nil && strings.Contains(err.Error(), "already in use") {
		fmt.Println("Trying Again...")
		listener, port, err := getListener(myIP)
		return listener, port, err
	}
	if err != nil {
		return listener, "", err
	}
	return listener, randPort, nil
}

func divideInChunks(fileSize int64) (size uint, total uint) {
	approxSize := fileSize / 250
	if approxSize < 8192 {
		approxSize = 8192
		return uint(approxSize), uint(fileSize / approxSize)
	}
	if approxSize > 2097152 {
		approxSize = 2097152
		return uint(approxSize), uint(fileSize / approxSize)
	}
	return uint(approxSize), uint(fileSize / approxSize)
}

func AddMe(tIP string, fname string, port string) string {
	tAddr, _ := net.ResolveUDPAddr("udp4", tIP)
	tConn, err := net.DialUDP("udp4", nil, tAddr)
	if err != nil {
		log.Fatal(err)
	}
	tConn.Write([]byte("add:" + fname + port))
	buff := make([]byte, 16)
	n, _ := tConn.Read(buff)
	fmt.Println(string(buff[:n]))
	return string(buff[:n])
}

func TrackMe(fsinfo FileShareInfo, fname string, tIP string, port string) {
	tAddr, _ := net.ResolveUDPAddr("udp4", tIP)
	tConn, err := net.DialUDP("udp4", nil, tAddr)
	if err != nil {
		log.Fatal(err)
	}
	msg := []byte("start:")
	msg2 := []byte("^" + port)
	// tConn.Write([]byte("start:" + fname))
	// time.Sleep(1 * time.Second)
	data, _ := json.Marshal(fsinfo)
	msg = append(msg, data...)
	msg = append(msg, msg2...)
	tConn.Write(msg)
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

type Seeder struct {
	FInfo   FileShareInfo
	Info    *ProgressInfo
	cReader chunkrw.FileChunkReader
	trackerIP string
}

func NewSeeder(file *os.File, tIP string) *Seeder {
	finfo, _ := file.Stat()
	fsInfo := FileShareInfo{}
	fsInfo.Name = finfo.Name()
	fsInfo.Size = finfo.Size()
	fsInfo.Mode = uint32(finfo.Mode())
	fsInfo.ChunkSize, fsInfo.TotalChunks = divideInChunks(fsInfo.Size)
	return &Seeder {
		FInfo: fsInfo,
		Info: &ProgressInfo{
			ItemId:    fsInfo.Name,
			Status:    "Sharing",
			BytesDown: fsInfo.Size,
			TSize:     fsInfo.Size,
			PeerNum:   0,
		},
		cReader: chunkrw.FileChunkReader{
			File:      file,
			ChunkSize: fsInfo.ChunkSize,
		},
		trackerIP: tIP,
	}
}

func (sed *Seeder) StartShare() {
	myIP := getLocalIP()
	listner, port, err := getListener(myIP)
	// fmt.Println(port)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Sharing %s at: %s\n", sed.FInfo.Name, listner.Addr())

	// Get Tracked
	// sed.trackerIP = tracker.SearchTrackers()
	// fmt.Println(sed.trackerIP)
	// if sed.trackerIP == "" {
	// 	tker := tracker.Tracker{
	// 		TInfos: make(map[string]tracker.TrackInfo),
	// 	}
	// 	tker.Track(tracker.FileShareInfo(sed.FInfo), myIP + port)
	// 	go tker.StartTracking()
	// 	sed.trackerIP = myIP + ":49718"
	// 	fmt.Println("Started Tracking at: " + sed.trackerIP)
	// } else {
	if AddMe(sed.trackerIP, sed.FInfo.Name, port) == "false" {
		fmt.Println("Tracker Tracking: ", sed.FInfo.Name)
		TrackMe(sed.FInfo, sed.FInfo.Name, sed.trackerIP, port)
	}
	// }

	for {
		fmt.Println("Started Loop")
		conn, err := listner.Accept()
		fmt.Println("Listening at...", conn.RemoteAddr())
		if err != nil {
			log.Fatal(err)
		}
		sed.Info.PeerNum += 1
		go sed.shareHandler(conn)
	}
}

func (sed *Seeder) shareHandler(conn net.Conn) {
	rBuff := make([]byte, 128)
	wBuff := make([]byte, sed.cReader.ChunkSize)
	// Communication Loop
	for {
		n, err := conn.Read(rBuff)
		if err != nil {
			if err == io.EOF {
				fmt.Printf("Connection lost with %v\n", conn.RemoteAddr())
				return
			}
			log.Fatal(err)
		}
		req := string(rBuff[:n])
		// fmt.Println(req)
		str := strings.Split(req, ":")
		switch str[0] {
		case "chunk":
			cNum, err := strconv.ParseUint(str[1], 10, 0)
			if err != nil {
				log.Fatal(err)
			}
			n, err := sed.cReader.ReadChunk(uint(cNum), wBuff)
			if err != nil {
				log.Fatal(err)
			}
			// fmt.Printf("Read %d (%d)\n", cNum, n)
			_, err = conn.Write(wBuff[:n])
			if err != nil {
				log.Fatal(err)
			}
		case "quit":
			sed.Info.PeerNum -= 1
			conn.Close()
			return
		}
	}
}
