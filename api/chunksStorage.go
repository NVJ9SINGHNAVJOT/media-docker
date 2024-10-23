package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/rs/zerolog/log"
)

// fileStatus holds metadata about the file being uploaded, including its type, status, chunk number, and unique chunk identifier.
type fileStatus struct {
	Type    string `json:"type"`    // The type of the file (e.g., "image", "video").
	Status  string `json:"status"`  // The current status of the file upload (e.g., "start", "uploading", "completed").
	Chunk   int64  `json:"chunk"`   // The current chunk number being processed.
	ChunkId string `json:"chunkId"` // A unique identifier for the chunk, used for tracking uploads.
}

// checkForm validates the form data from the HTTP request and returns the file configuration, file status, and any error encountered.
func checkForm(r *http.Request) (helper.FileConfig, fileStatus, error) {
	// Initialize a fileStatus struct with the type and status from the form values.
	checkFileStatus := fileStatus{
		Type:   r.FormValue("type"),   // Retrieve the file type from the form data.
		Status: r.FormValue("status"), // Retrieve the file status from the form data.
	}

	// Check if the file type exists in the helper's constants.
	fileConfig, exist := helper.Constants.Files[checkFileStatus.Type]
	if !exist {
		// Return an error if the file type is invalid.
		return helper.FileConfig{}, fileStatus{}, fmt.Errorf("invalid file type")
	}

	fileChunk := r.FormValue("chunk")                        // Retrieve the chunk value from the form.
	intFileChunk, err := strconv.ParseInt(fileChunk, 10, 64) // Convert the chunk value to int64.

	if err != nil || intFileChunk < 0 {
		// Return an error if the chunk value is invalid or negative.
		return helper.FileConfig{}, fileStatus{}, fmt.Errorf("invalid chunk number")
	}

	// Ensure that the status is "start" when the chunk number is 0.
	if intFileChunk == 0 && checkFileStatus.Status != "start" {
		return helper.FileConfig{}, fileStatus{}, fmt.Errorf("invalid status: start required for chunk 0")
	}

	// Set the validated chunk number in the fileStatus struct.
	checkFileStatus.Chunk = intFileChunk

	// ChunkId generation/validation based on the current status.
	switch checkFileStatus.Status {
	case "start":
		// Generate a new ChunkId when the upload starts.
		// This ensures the uploaded file is saved with a unique name to prevent overwriting.
		checkFileStatus.ChunkId = uuid.New().String()
	case "uploading", "completed":
		// Validate and retrieve the existing ChunkId from the form.
		chunkId := r.FormValue("chunkId")
		if err := helper.ValidateAndParseUUID(chunkId); err != nil {
			// Return an error if the ChunkId is invalid.
			return helper.FileConfig{}, fileStatus{}, fmt.Errorf("invalid chunkId")
		}
		checkFileStatus.ChunkId = chunkId

		// Check if the chunk size exceeds the maximum allowed size.
		if checkFileStatus.Chunk > (fileConfig.MaxSize / 2) {
			return helper.FileConfig{}, fileStatus{}, fmt.Errorf("chunk size exceeded")
		}
	default:
		// Return an error for any invalid status values.
		return helper.FileConfig{}, fileStatus{}, fmt.Errorf("invalid file status")
	}

	// Return the file configuration, updated fileStatus, and no error.
	return fileConfig, checkFileStatus, nil
}

// totalChunksSize calculates the total size of all chunk files in the specified directory.
func totalChunksSize(directory string) (int64, error) {
	var totalSize int64 // Initialize totalSize to accumulate the size of chunk files.

	// Walk through the directory and calculate the total size of all files.
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err // Return any error encountered during file info retrieval.
		}
		// Only add the size of regular files (not directories).
		if !info.IsDir() {
			totalSize += info.Size() // Accumulate the size of the current file.
		}
		return nil // Continue walking through the directory.
	})

	if err != nil {
		return 0, err // Return error if encountered during the walk.
	}

	return totalSize, nil // Return the total size of chunks.
}

// mergeChunks combines all uploaded file chunks into a single final file.
// It takes a fileStatus structure, the unique filename for the final file,
// the original file name, and its extension as parameters.
func mergeChunks(fileStatus fileStatus, uuidFilename string) error {
	// Construct the path for the final file where all chunks will be merged.
	finalFilePath := filepath.Join(helper.Constants.UploadStorage, uuidFilename)

	// Create or open the final file for writing; if it doesn't exist, it will be created.
	finalFile, err := os.OpenFile(finalFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error creating final file: %w", err) // Return an error if final file creation fails.
	}
	defer finalFile.Close() // Ensure the final file is closed when the function returns.

	intFilesChunk := int(fileStatus.Chunk) // Convert Chunk from int64 to int for iteration.
	// Iterate through all chunks based on the total number of chunks indicated in fileStatus.
	for i := 0; i <= intFilesChunk; i++ {
		// Construct the path for each individual chunk file.
		chunkFilePath := filepath.Join(helper.Constants.UploadStorage, fileStatus.Type+"s", uuidFilename, fmt.Sprintf("chunk_%d", i))

		// Open the chunk file for reading.
		chunkFile, err := os.Open(chunkFilePath)
		if err != nil {
			return fmt.Errorf("error reading chunk %d: %w", i, err) // Return an error if chunk file opening fails.
		}

		// Copy the contents of the chunk file to the final file.
		_, err = io.Copy(finalFile, chunkFile)
		chunkFile.Close() // Close the chunk file after copying.
		if err != nil {
			return fmt.Errorf("error merging chunk %d: %w", i, err) // Return an error if merging fails.
		}
	}
	return nil // Return nil to indicate that the merging was successful.
}

// ChunksStorage handles file upload requests, validates data,
// and manages chunked file uploads by saving chunks to disk.
func ChunksStorage(w http.ResponseWriter, r *http.Request) {
	// Validate form data and retrieve file configuration and status.
	fileConfig, fileStatus, err := checkForm(r)
	if err != nil {
		// Respond with a 400 Bad Request if file data is invalid.
		helper.Response(w, http.StatusBadRequest, "invalid file data", err)
		return
	}

	fileName := fileStatus.Type + "File" // Determine the file name based on the file type.

	// Parse multipart form data with a size limit defined by helper.Constants.MaxChunkSize.
	if err := r.ParseMultipartForm(helper.Constants.MaxChunkSize); err != nil {
		// Respond with a 400 Bad Request if parsing fails.
		helper.Response(w, http.StatusBadRequest, "error parsing form data", err)
		return
	}

	// Retrieve the uploaded file and its header information.
	file, header, err := r.FormFile(fileName)
	if err != nil {
		// Respond with a 400 Bad Request if no file is present.
		helper.Response(w, http.StatusBadRequest, "error reading file - no file present", err)
		return
	}

	// Validate the file type using a custom function.
	if !helper.Constants.IsValidFileType(fileStatus.Type, header.Header.Get("Content-Type")) {
		// Respond with a 415 Unsupported Media Type if the file type is invalid.
		helper.Response(w, http.StatusUnsupportedMediaType, "unsupported "+fileName+" file type", nil)
		file.Close() // Close the file before returning.
		return
	}

	// Check if the file size exceeds the allowed limit by helper.Constants.MaxChunkSize.
	if header.Size > helper.Constants.MaxChunkSize {
		// Respond with a 413 Request Entity Too Large if the file is too large.
		helper.Response(w, http.StatusRequestEntityTooLarge, "file too large", nil)
		file.Close() // Close the file before returning.
		return
	}

	// Extract the file extension and construct a unique filename for the chunk.
	uuidFilename := fmt.Sprintf("%s.%s",
		fileStatus.ChunkId,
		strings.Split(header.Header.Get("Content-Type"), "/")[1],
	)
	chunksDir := helper.Constants.UploadStorage + "/" + fileStatus.Type + "s" + "/" + uuidFilename

	// Determine the chunk file path based on the upload status.
	if fileStatus.Status == "start" {
		// Create a directory for the chunk files if the upload is starting.
		if err := pkg.CreateDir(chunksDir); err != nil {
			// Respond with a 500 Internal Server Error if directory creation fails.
			helper.Response(w, http.StatusInternalServerError, "error while creating dir for chunks", err)
			file.Close() // Close the uploaded file before returning.
			return
		}
	}

	// Save chunk file path based on the upload status and chunk number.
	chunkFilepath := filepath.Join(chunksDir, fmt.Sprintf("chunk_%d", fileStatus.Chunk))

	// Create a new file on disk for the chunk.
	out, err := os.Create(chunkFilepath)
	if err != nil {
		// Close the uploaded file before returning on error.
		file.Close()
		pkg.AddToDirDeleteChan(chunksDir)
		// Respond with a 500 Internal Server Error if chunk file creation fails.
		helper.Response(w, http.StatusInternalServerError, "error creating chunk file", err)
		return
	}
	// The file and output streams will be closed later after copying the file content.

	// Copy the content of the uploaded file to the new file on disk.
	_, err = io.Copy(out, file)
	if err != nil {
		// Close both the uploaded file and the output file before returning on error.
		file.Close()
		out.Close()
		pkg.AddToDirDeleteChan(chunksDir)
		// Respond with a 500 Internal Server Error if saving the chunk file fails.
		helper.Response(w, http.StatusInternalServerError, "error saving chunk file", err)
		return
	}

	// Manually close the uploaded file.
	if err = file.Close(); err != nil {
		// Log a warning if closing the uploaded file fails.
		log.Warn().Err(err).Msgf("Warning: Could not close uploaded file: %s", chunkFilepath)
	}

	// Manually close the output file.
	if err = out.Close(); err != nil {
		// Log a warning if closing the output file fails.
		log.Warn().Err(err).Msgf("Warning: Could not close output file: %s", chunkFilepath)
	}

	// Respond based on the file upload status.
	if fileStatus.Status == "start" {
		// Respond with a 200 OK indicating the upload has started successfully.
		helper.Response(w, http.StatusOK, "file chunk upload started successfully", map[string]string{"newChunkId": fileStatus.ChunkId})
	} else if fileStatus.Status == "uploading" {
		// Respond with a 200 OK indicating the chunk has been uploaded successfully.
		helper.Response(w, http.StatusOK, fmt.Sprintf("file chunk uploaded successfully chunkId: %s", fileStatus.ChunkId), nil)
	} else {
		// Remove all chunk files.
		defer pkg.AddToDirDeleteChan(chunksDir)

		// Check total file size
		totalSize, err := totalChunksSize(chunksDir)
		if err != nil {
			helper.Response(w, http.StatusInternalServerError, "error checking total chunks size", err)
			return
		}
		if totalSize > fileConfig.MaxSize {
			helper.Response(w, http.StatusBadRequest, "total chunks size is greater than valid max size", err)
			return
		}
		// Merge all chunks into a final file once all chunks are uploaded.
		err = mergeChunks(fileStatus, uuidFilename)
		if err != nil {
			// Respond with a 500 Internal Server Error if merging fails.
			helper.Response(w, http.StatusInternalServerError, "error saving file", err)
			return
		}
		// Respond with a 200 OK indicating that the chunk uploading has completed successfully.
		helper.Response(w, http.StatusOK, fmt.Sprintf("file chunk uploading completed successfully chunkId: %s", fileStatus.ChunkId),
			map[string]string{"uuidFilename": uuidFilename})
	}
}
