package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	http.HandleFunc("/api/upload", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			// handle GET request
		case "POST":
			uploadFile(w, r)
		default:
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/file/", streamFileHandler)
	fmt.Println("trying to run the server, possibly running")
	err := http.ListenAndServe(":5300", nil)
	if err != nil {
		fmt.Println("error occured runnnig the server")
		fmt.Println(err)
	}
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(200 << 20) // Limit 10MB
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Unable to retrieve file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fmt.Printf("File received: %s, Size: %d bytes\n", header.Filename, header.Size)

	uploadDir := "uploads"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.Mkdir(uploadDir, 0755)
	}

	filePath := filepath.Join(uploadDir, header.Filename)
	destFile, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Unable to create file", http.StatusInternalServerError)
		return
	}
	defer destFile.Close()

	// Stream file contents to the destination
	written, err := io.Copy(destFile, file)
	if err != nil {
		http.Error(w, "Error writing file", http.StatusInternalServerError)
		return
	}

	// Respond to the client
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"success": true, "filename": "%s", "size": %d}`, header.Filename, written)
}

func streamFileHandler(w http.ResponseWriter, r *http.Request) {
	// Get the file path from the request URL (e.g. /uploads/myfile.txt)
	log.Println(r.URL.Path)
	filePath := filepath.Join("./uploads", r.URL.Path[len("/file/"):])
	log.Println(filePath)

	// Open the file for reading
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	// Get the file's stats to set proper headers
	fileInfo, err := file.Stat()
	if err != nil {
		http.Error(w, "Unable to retrieve file info", http.StatusInternalServerError)
		return
	}

	// Set the appropriate headers for inline display
	// Dynamically set the Content-Type based on file extension, if needed
	switch ext := filepath.Ext(fileInfo.Name()); ext {
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".mp4", ".mov":
		w.Header().Set("Content-Type", "video/mp4")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".pdf":
		w.Header().Set("Content-Type", "application/pdf")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	// Set Content-Disposition to inline so the file is displayed in the browser
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s", fileInfo.Name()))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// Stream the file in chunks
	_, err = io.Copy(w, file)
	if err != nil {
		http.Error(w, "Failed to send file", http.StatusInternalServerError)
	}
}