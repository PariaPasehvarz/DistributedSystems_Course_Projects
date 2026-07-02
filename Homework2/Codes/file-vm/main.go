package main

import (
	"errors"
	"log"
	"net"
	"net/rpc"
	"os"
	"path/filepath"
)

type FileArgs struct {
	Folder   string 
	FileName string 
}

type FileReply struct {
	Data []byte
}

type FileService struct{}

func (f *FileService) FetchFile(args *FileArgs, reply *FileReply) error {
	log.Printf("Received request for folder: %s, file: %s", args.Folder, args.FileName)

	var baseDir string
	if args.Folder == "files" {
		baseDir = "./files"
	} else if args.Folder == "images" {
		baseDir = "./images"
	} else {
		return errors.New("invalid folder category: must be 'images' or 'files'")
	}

	safePath := filepath.Join(baseDir, filepath.Base(args.FileName))
	
	data, err := os.ReadFile(safePath)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("requested file not found in " + args.Folder)
		}
		return err
	}

	reply.Data = data
	return nil
}

func main() {
	fileService := new(FileService)
	rpc.Register(fileService)

	listener, err := net.Listen("tcp", ":8083")
	if err != nil {
		log.Fatalf("Error starting File RPC server: %v", err)
	}
	defer listener.Close()
	log.Println("Flexible File Server running on port 8083 using Native Go-RPC...")

	rpc.Accept(listener)
}