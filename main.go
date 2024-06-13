package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func listFiles() []os.DirEntry {
	files, err := os.ReadDir("./transferir")
	if err != nil {
		log.Fatal("Unable to read transferir directory:", err)
	}
	return files
}

func displayFiles(files []os.DirEntry) {
	fmt.Println("\nArquivos encontrados:")
	for _, file := range files {
		if !file.IsDir() {
			filePath := filepath.Join("./transferir", file.Name())
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				continue
			}
			fileSize := fileInfo.Size()
			fmt.Printf("%s (%d bytes)\n", file.Name(), fileSize)
		}
	}
}

func confirmTransfer() bool {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Confirmar arquivos (s/n): \n")
	scanner.Scan()
	input := scanner.Text()
	return input == "s" || input == "S"
}

var server *http.Server 
var serverRunning bool  

func startServer(stopChan chan struct{}) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		files := listFiles()
		fmt.Fprintf(w, "<html><body style='text-align:center;'>")
		fmt.Fprintf(w, "<h1 style='font-size:60px;'>Arquivos no Servidor</h1>")
		fmt.Fprintf(w, "<ul style='font-size:55px; list-style-position: inside;'>")
	
		for _, file := range files {
			if !file.IsDir() {
				filePath := filepath.Join("./transferir", file.Name())
				fileInfo, err := os.Stat(filePath)
				if err != nil {
					continue
				}
				fileSize := fileInfo.Size()
				fmt.Fprintf(w, "<li><a href='/download?file=%s'>%s</a> (%d bytes)</li>", file.Name(), file.Name(), fileSize)
			}
		}
	
		fmt.Fprintf(w, "</ul>")
		fmt.Fprintf(w, "</body></html>")
	})

	mux.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		fileName := r.URL.Query().Get("file")
		if fileName == "" {
			http.Error(w, "File parameter is missing", http.StatusBadRequest)
			return
		}

		filePath := filepath.Join("./transferir", fileName)
		file, err := os.Open(filePath)
		if err != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		defer file.Close()

		w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
		w.Header().Set("Content-Type", "application/octet-stream")
		http.ServeFile(w, r, filePath)
	})

	mux.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Servidor será desligado em breve.")
		stopChan <- struct{}{}
	})

	server = &http.Server{
		Addr:    ":8080", 
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Could not listen on :8080: %v\n", err)
		}
	}()

	serverRunning = true
	return server
}

func closeServer(server *http.Server) {
	if !serverRunning {
		fmt.Println("O servidor não está rodando.")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Servidor forçado a desligar: %v", err)
	}
	fmt.Println("Servidor desligado.")
	serverRunning = false
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func displayMenu() int {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("\n1. Enviar Arquivos")
	fmt.Println("2. Fechar Servidor")
	fmt.Println("3. Sair")
	fmt.Print("Escolha uma opção: ")
	scanner.Scan()
	input := scanner.Text()
	option, err := strconv.Atoi(input)
	if err != nil {
		fmt.Println("Por favor, insira um número válido.")
		return displayMenu()
	}
	return option
}

func main() {
	for {
		option := displayMenu()
		switch option {
		case 1:
			if serverRunning {
				fmt.Println("\nO servidor já está rodando.")
			} else {
				files := listFiles()
				displayFiles(files)
				if confirmTransfer() {
					stopChan := make(chan struct{})
					startServer(stopChan) 
					ip := getLocalIP()
					fmt.Printf("\nServidor rodando em http://%s:8080/\n", ip) 
				} else {
					fmt.Println("Transferência cancelada.")
				}
			}
		case 2:
			fmt.Println("\nFechando o servidor...")
			closeServer(server)
		case 3:
			fmt.Println("\nSaindo...")
			os.Exit(0)
		default:
			fmt.Println("\nOpção inválida. Por favor, tente novamente.")
		}
	}
}


