package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/nvj9singhnavjot/media-docker/helper"
	"github.com/nvj9singhnavjot/media-docker/pkg"
	"github.com/nvj9singhnavjot/media-docker/pkg/asset"
	"github.com/nvj9singhnavjot/media-docker/validator"
	"github.com/rs/zerolog/log"
)

// fileStatus holds metadata about the file being uploaded, including its type, status, chunk number, and asset id.
type fileStatus struct {
	Type     string // The storage category of the file (e.g. "image", "video", "document").
	Status   string // The current status of the file upload ("start", "uploading", "completed").
	Chunk    int64  // The current chunk number being processed.
	ID       string // The asset id; also the directory the finished asset lives in.
	FileName string // The original file name supplied by the client, used to derive the extension.
}

// checkForm validates the form data from the HTTP request and returns the file configuration, file status, and any error encountered.
func checkForm(r *http.Request) (helper.FileConfig, fileStatus, error) {
	// Initialize a fileStatus struct with the type and status from the form values.
	checkFileStatus := fileStatus{
		Type:     r.FormValue("type"),     // Retrieve the storage category from the form data.
		Status:   r.FormValue("status"),   // Retrieve the file status from the form data.
		FileName: r.FormValue("fileName"), // Original file name; optional, used only to derive the extension.
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

	// Asset id generation/validation based on the current status.
	switch checkFileStatus.Status {
	case "start":
		// Mint the asset id when the upload starts.
		//
		// NOTE: Since v4 this is the final media id, not a throwaway upload
		// handle. The dispatch endpoints reuse it, so an asset has exactly one
		// identifier from its first chunk to the URL it is served under.
		checkFileStatus.ID = uuid.New().String()
	case "uploading", "completed":
		// Validate and retrieve the existing asset id from the form.
		id := r.FormValue("id")
		if err := validator.ValidateAndParseUUID(id); err != nil {
			// Return an error if the id is invalid.
			return helper.FileConfig{}, fileStatus{}, fmt.Errorf("invalid id")
		}
		checkFileStatus.ID = id

		// Check if the chunk count exceeds what the maximum file size allows.
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

// mergeChunks combines all uploaded chunks into the asset's raw file.
//
// The merge writes straight to its final destination inside media storage
// rather than to a staging file that would then need copying across. Assembling
// the file is one full pass over the data either way, so spending that pass on
// the final location avoids a second full copy of every upload.
func mergeChunks(status fileStatus, finalFilePath string) error {
	// Create or open the final file for writing; if it doesn't exist, it will be created.
	finalFile, err := os.OpenFile(finalFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("error creating final file: %w", err) // Return an error if final file creation fails.
	}
	defer finalFile.Close() // Ensure the final file is closed when the function returns.

	chunksDir := asset.StagingDir(status.Type, status.ID)

	intFilesChunk := int(status.Chunk) // Convert Chunk from int64 to int for iteration.
	// Iterate through all chunks based on the total number of chunks indicated in fileStatus.
	for i := 0; i <= intFilesChunk; i++ {
		// Construct the path for each individual chunk file.
		chunkFilePath := filepath.Join(chunksDir, fmt.Sprintf("chunk_%d", i))

		// Open the chunk file for reading.
		chunkFile, err := os.Open(chunkFilePath)
		if err != nil {
			return fmt.Errorf("error reading chunk %d: %w", i, err) // Return an error if chunk file opening fails.
		}

		// Copy the contents of the chunk file to the final file.
		_, err = io.Copy(finalFile, chunkFile)
		closeErr := chunkFile.Close() // Close the chunk file after copying.
		if closeErr != nil {
			log.Warn().Err(closeErr).Msgf("Warning: Could not close uploaded chunk file: %s", chunkFilePath)
		}
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
	fileConfig, status, err := checkForm(r)
	if err != nil {
		// Respond with a 400 Bad Request if file data is invalid.
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusBadRequest, "invalid file data", err)
		return
	}

	fileName := status.Type + "File" // Determine the form field name based on the file type.

	// Parse multipart form data with a size limit defined by helper.Constants.MaxChunkSize.
	if err := r.ParseMultipartForm(helper.Constants.MaxChunkSize); err != nil {
		// Respond with a 400 Bad Request if parsing fails.
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusBadRequest, "error parsing form data", err)
		return
	}

	// Retrieve the uploaded file and its header information.
	file, header, err := r.FormFile(fileName)
	if err != nil {
		// Respond with a 400 Bad Request if no file is present.
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusBadRequest, "error reading file - no file present", err)
		return
	}

	contentType := header.Header.Get("Content-Type")

	// Validate the file type using a custom function.
	if !helper.Constants.IsValidFileType(status.Type, contentType) {
		// Respond with a 415 Unsupported Media Type if the file type is invalid.
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusUnsupportedMediaType, "unsupported "+fileName+" file type", nil)
		file.Close() // Close the file before returning.
		return
	}

	// Check if the file size exceeds the allowed limit by helper.Constants.MaxChunkSize.
	if header.Size > helper.Constants.MaxChunkSize {
		// Respond with a 413 Request Entity Too Large if the file is too large.
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusRequestEntityTooLarge, "file too large", nil)
		file.Close() // Close the file before returning.
		return
	}

	// Chunks are staged in a private directory on the upload storage volume,
	// keyed by the asset id. Only the merged result reaches media storage.
	chunksDir := asset.StagingDir(status.Type, status.ID)

	// Determine the chunk file path based on the upload status.
	if status.Status == "start" {
		// Create a directory for the chunk files if the upload is starting.
		if err := pkg.CreateDir(chunksDir); err != nil {
			// Respond with a 500 Internal Server Error if directory creation fails.
			helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusInternalServerError, "error while creating dir for chunks", err)
			file.Close() // Close the uploaded file before returning.
			return
		}
	}

	// Save chunk file path based on the upload status and chunk number.
	chunkFilepath := filepath.Join(chunksDir, fmt.Sprintf("chunk_%d", status.Chunk))

	// Create a new file on disk for the chunk.
	out, err := os.Create(chunkFilepath)
	if err != nil {
		// Close the uploaded file before returning on error.
		file.Close()
		pkg.AddToDirDeleteChan(chunksDir)
		// Respond with a 500 Internal Server Error if chunk file creation fails.
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusInternalServerError, "error creating chunk file", err)
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
		helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusInternalServerError, "error saving chunk file", err)
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
	switch status.Status {
	case "start":
		// Respond with a 200 OK indicating the upload has started successfully.
		helper.SuccessResponse(w, helper.GetRequestID(r), http.StatusOK, "file chunk upload started successfully", map[string]string{"id": status.ID})

	case "uploading":
		// Respond with a 200 OK indicating the chunk has been uploaded successfully.
		helper.SuccessResponse(w, helper.GetRequestID(r), http.StatusOK, fmt.Sprintf("file chunk uploaded successfully id: %s", status.ID), nil)

	default:
		// Remove all chunk files once the upload is finished, whatever the outcome.
		defer pkg.AddToDirDeleteChan(chunksDir)

		// Check total file size
		totalSize, err := totalChunksSize(chunksDir)
		if err != nil {
			helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusInternalServerError, "error checking total chunks size", err)
			return
		}
		if totalSize > fileConfig.MaxSize {
			helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusBadRequest, "total chunks size is greater than valid max size", nil)
			return
		}

		// Merge all chunks into the asset's raw file and record its metadata.
		if err := finalizeUpload(status, contentType, func(finalPath string) error {
			return mergeChunks(status, finalPath)
		}); err != nil {
			helper.ErrorResponse(w, helper.GetRequestID(r), http.StatusInternalServerError, "error saving file", err)
			return
		}

		// Respond with a 200 OK indicating that the chunk uploading has completed successfully.
		helper.SuccessResponse(w, helper.GetRequestID(r), http.StatusOK, fmt.Sprintf("file chunk uploading completed successfully id: %s", status.ID),
			map[string]string{"id": status.ID})
	}
}

// finalizeUpload creates the asset directory, writes the raw file through the
// supplied write function, and records the asset's metadata.
//
// From the moment this returns the asset is publicly resolvable: the raw upload
// is already in media storage and can be served while conversion is pending. It
// is written with dispatched=false, so an upload the caller never hands off is
// reaped by the janitor rather than lingering forever.
func finalizeUpload(status fileStatus, contentType string, write func(finalPath string) error) error {
	ext := helper.Constants.SanitizeExt(status.Type, status.FileName, contentType)
	assetDir := asset.Dir(status.Type, status.ID)

	if err := pkg.CreateDir(assetDir); err != nil {
		return fmt.Errorf("error creating asset directory: %w", err)
	}

	if err := write(asset.OriginalPath(status.Type, status.ID, ext)); err != nil {
		// Do not leave a half-written asset behind; it would resolve to a
		// truncated file.
		pkg.AddToDirDeleteChan(assetDir)
		return err
	}

	if err := asset.WriteMeta(asset.Meta{
		ID:           status.ID,
		Type:         status.Type,
		Ext:          ext,
		OriginalName: status.FileName,
		Dispatched:   false,
		CreatedAt:    time.Now(),
	}); err != nil {
		pkg.AddToDirDeleteChan(assetDir)
		return err
	}

	return nil
}
