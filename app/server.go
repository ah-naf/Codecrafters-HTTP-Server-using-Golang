package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
)

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	defer l.Close()
	fmt.Println("Server listening on 4221")
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			continue
		}
		// Handle Client Connection
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
    // Ensure we close the connection after we're done
    defer conn.Close()
	
	request, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		fmt.Println("Error reading request. ", err.Error())
		return
	}
	url, method := request.URL.Path, request.Method
	response := ""
	
	if url == "/" {
		response = "HTTP/1.1 200 OK\r\n\r\n"
	} else if url == "/user-agent" {
		response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(request.UserAgent()), request.UserAgent())
		
	} else if strings.Contains(url, "/echo/") {
		message := url[6:]
		encoding := request.Header.Get("Accept-Encoding")
		if strings.Contains(encoding, "gzip") {
			compressedData, err := gzipCompress([]byte(message))
			if err != nil {
				response = "HTTP/1.1 500 Internal Server Error\r\n\r\n"
				conn.Write([]byte(response))
				return
			}
			message = string(compressedData)
			response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Encoding: gzip\r\nContent-Length: %d\r\n\r\n%s", len(message), message)
		} else {
			response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(message), message)
		}
	} else if strings.Contains(url, "/files/") {
		fileName := url[7:]
		dir := os.Args[2]
		if method == "GET" {
			data, err := readFile(dir + fileName)
			if err != nil {
				response = "HTTP/1.1 404 Not Found\r\n\r\n"
			} else {
				response = fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", len(data), data)
			}
		} else if method == "POST" {
			outFile, err := os.Create(dir + fileName)
			if err != nil {
				response = "HTTP/1.1 500 Internal Server Error\r\n\r\n"
				conn.Write([]byte(response))
				return
			}
			defer outFile.Close()

			// Use io.Copy to efficiently stream the request body to the file
			_, err = io.Copy(outFile, request.Body)
			if err != nil {
				response = "HTTP/1.1 500 Internal Server Error\r\n\r\n"
				conn.Write([]byte(response))
				return
			}
			response = "HTTP/1.1 201 Created\r\n\r\n"
		} else {
			response = "HTTP/1.1 404 Not Found\r\n\r\n"
		}
	} else {
		response = "HTTP/1.1 404 Not Found\r\n\r\n"
	}
	conn.Write([]byte(response))
}

func readFile(filePath string) (string, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return "", err
    }
    defer file.Close()

    var builder strings.Builder
    reader := bufio.NewReader(file)
    buffer := make([]byte, 1024)
    
    for {
        n, err := reader.Read(buffer)
        if n > 0 {
            builder.Write(buffer[:n])
        }
        if err == io.EOF {
            break
        }
        if err != nil {
            return "", err
        }
    }

    return builder.String(), nil
}

// Compress data using gzip
func gzipCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	_, err := gzipWriter.Write(data)
	if err != nil {
		return nil, err
	}
	err = gzipWriter.Close() // Make sure to close the writer to flush the data
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
