package main

import (
	// "bufio"
	"encoding/json"
	"fmt"
	"log"
	"mods/receiver"
	"mods/seeder"
	"mods/tracker"
	"net"
	"net/http"
	"os"
	"strings"
	// "time"
)
type ProgressInfo struct {
	ItemId    string `json:"itemId"`
	Status    string `json:"status"`
	BytesDown int64  `json:"bytesDown"`
	TSize     int64  `json:"tSize"`
	PeerNum   int    `json:"peerNum"`
}

var tIP string

var defPath string

var pInfos[]*ProgressInfo

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

func receive(writer http.ResponseWriter,fname string) {
	// tIP := tracker.SearchTrackers()
	// fmt.Println(tIP)
	// if tIP == ""{
	// 	writer.Write([]byte("{\"result\":0}"))
	// 	return
	// }
	nRecv, err := receiver.NewReceiver(fname, tIP)
	if err != nil{
		writer.Write([]byte("{\"result\":0}"))
		return
	}
	data, _ := json.Marshal(*(nRecv.Info))
	writer.Write(data)
	pInfos = append(pInfos, (*ProgressInfo)(nRecv.Info))
	if err != nil {
		log.Fatal(err)
	}
	go nRecv.StartDownload()
}


func share(writer http.ResponseWriter, fname string) {
	file, err := os.Open("./share/" + fname)
	if err != nil {
		log.Fatal(err)
	}
	sed := seeder.NewSeeder(file, tIP)
	data, _ := json.Marshal(*(sed.Info))
	writer.Write(data)
	pInfos = append(pInfos, (*ProgressInfo)(sed.Info))
	go sed.StartShare()
}

func progress() []byte {
	data, err := json.Marshal(pInfos)
	if err != nil {
		log.Fatal(err)
	}
	return data
}

func queryHandle(writer http.ResponseWriter, req *http.Request)  {
	msg := req.URL.Query()["msg"][0]
	// fmt.Println(msg)
	msgSplit := strings.Fields(msg)
	switch msgSplit[0] {
	case "share":
		share(writer, msgSplit[1])
	case "receive":
		receive(writer, msgSplit[1])
	case "progress":
		data := progress()
		writer.Write(data)
	case "path":
		msg := fmt.Sprintf("{\"path\":\"%s\"}", defPath)
		writer.Write([]byte(msg))
	}
	
}

func main() {
	defPath, _ = os.Getwd()
	myIP := getLocalIP()
	if myIP == "" {
		log.Fatal(fmt.Errorf("Not connected to network."))
	}
	tIP = tracker.SearchTrackers()
	if tIP == "" {
		tker := tracker.Tracker{
			TInfos: make(map[string]tracker.TrackInfo),
		}
		go tker.StartTracking()
		tIP = myIP + ":49718"
		fmt.Println("Started Tracking...")
	}

	fileServer := http.FileServer(http.Dir("./web"))
	http.HandleFunc("/protoMsg", queryHandle)
	http.Handle("/", fileServer)
	http.ListenAndServe("localhost:43480", nil)
	// termReader := bufio.NewReader(os.Stdin)
	// 	cmd, err := termReader.ReadString('\n')
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	cmdSplit := strings.Split(cmd, " ")
	// 	cmdSplit[1] = strings.TrimSpace(cmdSplit[1])
	// 	if len(cmdSplit) != 2 {
	// 		fmt.Println("usage: share <filename>")
	// 		fmt.Println("usage: receive <filename>")
	// 	}

	// 	switch cmdSplit[0] {
	// 	case "share":
	// 		share(cmdSplit[1])
	// 	case "receive":
	// 		receive(cmdSplit[1])
	// 	}
}