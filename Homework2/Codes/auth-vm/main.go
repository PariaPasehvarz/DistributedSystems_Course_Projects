package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"` 
}

type AuthArgs struct {
	Username string
	Password string
}

type AuthReply struct {
	Success bool
}

type AuthService struct{}

func loadUsersFromFile() ([]User, error) {
	fileData, err := os.ReadFile("users.json")
	if err != nil {
		return nil, err
	}

	var users []User
	err = json.Unmarshal(fileData, &users)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (t *AuthService) Login(args *AuthArgs, reply *AuthReply) error {
	log.Printf("Received JSON-RPC authentication request for user: %s", args.Username)

	users, err := loadUsersFromFile()
	if err != nil {
		log.Printf("File System Read Error: %v", err)
		return fmt.Errorf("internal authentication store failure")
	}

	inputHash := fmt.Sprintf("%x", sha256.Sum256([]byte(args.Password)))

	for _, user := range users {
		if user.Username == args.Username {
			reply.Success = (user.Password == inputHash)
			return nil
		}
	}

	reply.Success = false
	return nil
}

func main() {
	auth := new(AuthService)
	rpc.Register(auth)

	listener, err := net.Listen("tcp", ":8082")
	if err != nil {
		log.Fatalf("Failed to bind JSON-RPC listener port: %v", err)
	}
	defer listener.Close()
	log.Println("Secure JSON-RPC Auth Server active on port 8082 (File-Backed)...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Network entry mapping failure: %v", err)
			continue
		}
		go rpc.ServeCodec(jsonrpc.NewServerCodec(conn))
	}
}