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
	http.HandleFunc("/streamfile/", chunkTransferEncoding)
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

	fileInfo, err := file.Stat()
	if err != nil {
		http.Error(w, "Unable to retrieve file info", http.StatusInternalServerError)
		return
	}

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

	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s", fileInfo.Name()))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// Stream the file in chunks
	_, err = io.Copy(w, file)
	if err != nil {
		http.Error(w, "Failed to send file", http.StatusInternalServerError)
	}
}

func chunkTransferEncoding(w http.ResponseWriter, r *http.Request) {
	// Get the file path from the request URL
	log.Println(r.URL.Path)
	filePath := filepath.Join("./uploads", r.URL.Path[len("/streamfile/"):])
	log.Println(filePath)

	// Open the file for reading
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		http.Error(w, "Unable to retrieve file info", http.StatusInternalServerError)
		return
	}

	// Set headers based on the file type
	switch ext := filepath.Ext(filePath); ext {
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

	// Extract the "Range" header
	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		// No range header: serve the whole file
		w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s", fileInfo.Name()))
		http.ServeContent(w, r, fileInfo.Name(), fileInfo.ModTime(), file)
		return
	}

	// Parse the range header
	start, end, err := parseRange(rangeHeader, fileInfo.Size())
	if err != nil {
		http.Error(w, "Invalid Range header", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	// Set headers for partial content
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s", fileInfo.Name()))
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileInfo.Size()))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", end-start+1))
	w.WriteHeader(http.StatusPartialContent)

	// Seek to the start of the range
	_, err = file.Seek(start, io.SeekStart)
	if err != nil {
		http.Error(w, "Error seeking file", http.StatusInternalServerError)
		return
	}

	// Stream the requested range
	buffer := make([]byte, 8*1024) // 8 KB buffer
	toRead := end - start + 1
	for toRead > 0 {
		n, err := file.Read(buffer)
		if n > int(toRead) {
			n = int(toRead)
		}
		if n > 0 {
			_, writeErr := w.Write(buffer[:n])
			if writeErr != nil {
				http.Error(w, "Error writing to response", http.StatusInternalServerError)
				return
			}
			toRead -= int64(n)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			http.Error(w, "Error reading file", http.StatusInternalServerError)
			return
		}
	}
}

func parseRange(rangeHeader string, fileSize int64) (int64, int64, error) {
	// Example: "Range: bytes=0-1024"
	var start, end int64
	_, err := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid range format")
	}

	// Adjust end if not specified
	if end == 0 || end >= fileSize {
		end = fileSize - 1
	}

	// Validate range
	if start > end || start < 0 || end >= fileSize {
		return 0, 0, fmt.Errorf("invalid range values")
	}

	return start, end, nil
}
