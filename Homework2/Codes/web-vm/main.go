package main

import (
	"bytes"
	"encoding/json"
	"flag" 
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"runtime"
	"strconv"
	"sync"
	"time"
)

var (
	authVMIP   string
	fileVMIP   string
	pubSubVMIP string
)

var memoryLeakSilo []byte
var stateMutex sync.Mutex

type AuthArgs struct{ Username, Password string }
type AuthReply struct{ Success bool }

type FileArgs struct {
	Folder   string
	FileName string
}
type FileReply struct{ Data []byte }

type MemoryEvent struct {
	EventType   string    `json:"event_type"`
	Service     string    `json:"service"`
	MemoryMB    uint64    `json:"memory_mb"`
	ThresholdMB uint64    `json:"threshold_mb"`
	Timestamp   time.Time `json:"timestamp"`
}

func init() {
	flag.StringVar(&authVMIP, "auth-ip", "192.168.1.102", "Network IPv4 address of the VM2 Auth Server")
	flag.StringVar(&fileVMIP, "file-ip", "192.168.1.103", "Network IPv4 address of the VM3 File Server")
	flag.StringVar(&pubSubVMIP, "pubsub-ip", "192.168.1.104", "Network IPv4 address of the VM4 Pub/Sub Broker")
}

func main() {
	flag.Parse()

	log.Printf("Initializing Gateway Routing Configurations...")
	log.Printf("-> Auth Server target (VM2): %s:8082", authVMIP)
	log.Printf("-> File Server target (VM3): %s:8083", fileVMIP)
	log.Printf("-> PubSub Server target (VM4): %s:8084", pubSubVMIP)

	go startMemoryMonitor()

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", handleLoginRoute)
	http.HandleFunc("/dashboard", handleDashboardRoute)
	http.HandleFunc("/view-file", handleFileViewerRoute)
	http.HandleFunc("/consume-memory", handleConsumeMemory)

	log.Println("Web Server running on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}

func handleLoginRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tmpl, err := template.ParseFiles("templates/login.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	conn, err := net.Dial("tcp", authVMIP+":8082")
	if err != nil {
		http.Error(w, "Authentication Infrastructure Unreachable", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	client := jsonrpc.NewClient(conn)
	args := &AuthArgs{Username: username, Password: password}
	var reply AuthReply

	err = client.Call("AuthService.Login", args, &reply)
	if err != nil || !reply.Success {
		tmpl, _ := template.ParseFiles("templates/login.html")
		w.WriteHeader(http.StatusUnauthorized)
		tmpl.Execute(w, "Access Denied: Invalid Credentials Configuration")
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func handleDashboardRoute(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/dashboard.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func handleFileViewerRoute(w http.ResponseWriter, r *http.Request) {
	folder := r.URL.Query().Get("folder")
	name := r.URL.Query().Get("name")

	if folder == "" || name == "" {
		http.Error(w, "Missing URL query parameters", http.StatusBadRequest)
		return
	}

	client, err := rpc.Dial("tcp", fileVMIP+":8083")
	if err != nil {
		http.Error(w, "File Microservice Unreachable", http.StatusInternalServerError)
		return
	}
	defer client.Close()

	args := &FileArgs{Folder: folder, FileName: name}
	var reply FileReply

	err = client.Call("FileService.FetchFile", args, &reply)
	if err != nil {
		http.Error(w, "Remote File System Storage Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if folder == "images" {
		w.Header().Set("Content-Type", "image/png")
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Write(reply.Data)
}

func handleConsumeMemory(w http.ResponseWriter, r *http.Request) {
	mbStr := r.URL.Query().Get("mb")
	mb, err := strconv.Atoi(mbStr)
	if err != nil || mb <= 0 {
		w.Write([]byte("Provide a valid positive integer value"))
		return
	}

	stateMutex.Lock()
	leakAllocation := make([]byte, mb*1024*1024)
	for i := range leakAllocation {
		leakAllocation[i] = 1
	}
	memoryLeakSilo = append(memoryLeakSilo, leakAllocation...)
	stateMutex.Unlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	currentAllocMB := m.Alloc / (1024 * 1024)

	w.Write([]byte(fmt.Sprintf("Allocated %d MB. Current Heap: %d MB\n", mb, currentAllocMB)))
}

func startMemoryMonitor() {
	ticker := time.NewTicker(2 * time.Second)
	var threshold uint64 = 300

	for range ticker.C {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		currentAllocMB := m.Alloc / (1024 * 1024)

		if currentAllocMB > threshold {
			log.Printf("ALERT: Memory utilization breached threshold! Current: %d MB", currentAllocMB)
			
			event := MemoryEvent{
				EventType:   "HIGH_MEMORY_USAGE",
				Service:     "web-server",
				MemoryMB:    currentAllocMB,
				ThresholdMB: threshold,
				Timestamp:   time.Now(),
			}
			
			payload, _ := json.Marshal(event)
			
			go func() {
				resp, err := http.Post("http://"+pubSubVMIP+":8084/alerts", "application/json", bytes.NewBuffer(payload))
				if err == nil {
					resp.Body.Close()
				}
			}()
		}
	}
}