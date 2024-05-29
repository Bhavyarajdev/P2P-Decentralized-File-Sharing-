package tracker

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type FileShareInfo struct {
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	ChunkSize   uint   `json:"chunksize"`
	TotalChunks uint   `json:"totalchunks"`
	Mode        uint32 `json:"mode"`
}

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

// func allLocalIPs(aLocalIP string) []string {
// 	var localIPs [256]string
// 	ipSplit := strings.Split(aLocalIP, ".")
// 	joined := ""
// 	for i := 0; i < 3; i++ {
// 		joined += ipSplit[i] + "."
// 	}
// 	for i := 0; i <= 255; i++ {
// 		localIPs[i] = fmt.Sprintf("%s%d", joined, i)
// 	}
// 	return localIPs[:]
// }

func SearchTrackers() string {
	for i := 0; i < 3; i++ {
		fmt.Println("Searching Tracker")
		buff := make([]byte, 16)
		bAddr, err := net.ResolveUDPAddr("udp4", "255.255.255.255:49718")
		if err != nil {
			log.Fatal(err)
		}
		conn, err := net.ListenUDP("udp4", nil)
		if err != nil {
			log.Fatal(err)
		}
		conn.WriteToUDP([]byte("tracking"), bAddr)
		conn.SetReadDeadline(time.Now().Add(4 * time.Second))
		n, tAddr, err := conn.ReadFromUDP(buff)
		if err != nil {
			if strings.Contains(err.Error(), "i/o timeout") {
				continue
			} else {
				log.Fatal(err)
			}
		} 
		if string(buff[:n]) == "true" {
			return tAddr.String()
		}
	}
		
	return ""
}

type TrackInfo struct {
	FInfo FileShareInfo
	FSeeds []string
}

type Tracker struct {
	TInfos map[string]TrackInfo
	mux sync.Mutex
}

func (t *Tracker) Track(fsinfo FileShareInfo, ip string) {
	fmt.Println("Tracking:", fsinfo.Name)
	trackinfo := TrackInfo{
		FInfo: fsinfo,
		FSeeds: make([]string, 0),
	}
	fname := fsinfo.Name
	t.mux.Lock()
	trackinfo.FSeeds = append(trackinfo.FSeeds, ip)
	t.TInfos[fname] = trackinfo
	t.mux.Unlock()
	fmt.Println(t.TInfos[fname])
}

func (t *Tracker) Add(fname string, ip string) error {
	t.mux.Lock()
	trackinfo, isOk := t.TInfos[fname]
	if !isOk {
		t.mux.Unlock()
		return fmt.Errorf("no file")
	}
	trackinfo.FSeeds = append(trackinfo.FSeeds, ip)
	t.TInfos[fname] = trackinfo
	t.mux.Unlock()
	return nil
}

func (t *Tracker) StartTracking() {
	myIP := getLocalIP()
	fmt.Println(myIP)
	me, _ := net.ResolveUDPAddr("udp4", ":49718")
	conn, err := net.ListenUDP("udp4", me)
	if err != nil {
		log.Fatal(err)
	}
	buff := make([]byte, 2048)
	mux := sync.Mutex{}
	for {
		mux.Lock()
		n, sender, err := conn.ReadFromUDP(buff)
		mux.Unlock()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Request: %s\n", string(buff[:n]))
		go t.handleSender(*conn, sender, string(buff[:n]), &mux)
	}
}

func (t *Tracker) handleSender(conn net.UDPConn, sender *net.UDPAddr, msg string, mux *sync.Mutex) {
	msgSplit := strings.SplitN(msg, ":", 2)
	fmt.Println(msgSplit)
	switch msgSplit[0] {
	case "tracking":
		conn.WriteToUDP([]byte("true"), sender)
	case "info":
		fname := msgSplit[1]
		fmt.Println(fname)
		var trackinfo TrackInfo
		t.mux.Lock()
		trackinfo = t.TInfos[fname]
		t.mux.Unlock()
		fmt.Println(trackinfo)
		data, _ := json.Marshal(trackinfo)
		conn.WriteToUDP(data, sender)
	case "tracked":
		fname := msgSplit[1]
		fmt.Println(fname)
		t.mux.Lock()
		_, isOk := t.TInfos[fname]
		fmt.Println(t.TInfos)
		fmt.Println(isOk)
		t.mux.Unlock()
		msg := fmt.Sprint(isOk)
		conn.WriteToUDP([]byte(msg), sender)
	case "add":
		split := strings.Split(msgSplit[1], ":")
		fname := split[0]
		port := ":" + split[1]
		fmt.Println(fname)
		err := t.Add(fname, sender.IP.String() + port)
		if err != nil {
			conn.WriteToUDP([]byte("false"), sender)
		} else {
			conn.WriteToUDP([]byte("true"), sender)
		}
	case "start":
		split := strings.Split(msgSplit[1], "^")
		port := split[1]
		fmt.Println(port)
		// fmt.Println("inStart")
		data := []byte(split[0])
		var fsinfo FileShareInfo
		json.Unmarshal(data, &fsinfo)
		t.Track(fsinfo, sender.IP.String() + port)
		fmt.Println(t.TInfos)
	default:
		fmt.Println("Gone in Default!!")
	}
}
