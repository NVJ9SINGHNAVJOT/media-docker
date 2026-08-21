/* eslint-disable no-unused-vars */
/*
  IMPORTANT: Please do not modify this file, copy and paste it as-is into your project.

  This file contains the core logic for uploading files to the Media-Docker service.
  It handles the file upload process and manages the interaction with the Media-Docker API.

  Upload Process:
  - If the file upload is successful, the response will include a unique file ID (UUID v4 format)
    along with a file URL or multiple URLs (in case the media generates multiple output formats).
  - Once the file is uploaded, Media-Docker will process it according to the file type
    (e.g., image conversion, video transcoding, etc.).

  IMPORTANT (v4): the returned fileUrl works immediately.
  - You can store and serve the URL as soon as the upload returns; there is nothing to wait for.
  - Until conversion finishes, the URL serves the file exactly as you uploaded it, so a
    video is playable straight away as a plain progressive download.
  - When the consumer finishes, the same URL starts serving the converted output (an HLS
    playlist for video, a compressed JPEG for images, an MP3 for audio). Nothing on your
    side has to change: the URL never does.
  - "document" and "other" uploads are never converted and are always served as uploaded.

  Every response also carries an originalUrl. Conversion adds a representation rather than
  replacing one, so the file you uploaded stays on disk for the life of the asset and
  originalUrl serves it unchanged, before and after conversion. Use fileUrl to serve the
  best available version, and originalUrl when you want the source bytes: a
  download-original link, a quality comparison, or your own re-processing.

  There is no callback and no message queue to consume. The URL is the entire contract:
  it works the moment the upload returns, and it silently starts serving the converted
  output when conversion finishes. If conversion never succeeds, the URL keeps serving
  your original upload indefinitely -- degraded, not broken. Nothing on your side has to
  hold assets in a "pending" state.

  This file has no third-party dependencies.

  Note: This file is designed to ensure smooth integration with Media-Docker. If modifications are
  necessary, please review them carefully to avoid breaking the upload functionality.
*/

// Importing file system for handling file operations
import * as fs from "fs";
import * as fsp from "fs/promises";

type FileStatus = {
  type: string;
  status: string;
  chunk: number;
  fileName: string;
  // The asset id, returned by the server on the first chunk. It is the final
  // media id, so every later request in the upload refers to the same asset.
  id?: string;
};

/**
 * Storage categories accepted by the server.
 */
type MediaDockerFileType = "image" | "video" | "audio" | "document" | "other";

/**
 * Standardized response format
 * @template T
 * @typedef {Object} Result
 * @property {string} message - Response message
 * @property {T} data - Data payload
 */
type Result<T> = { message: string; data: T };

/**
 * Media file structure for common properties
 * @typedef {Object} MediaFile
 * @property {string} id - Unique identifier for the media file
 * @property {string} fileUrl - URL of the media file
 * @property {string} originalUrl - URL that always serves the file as uploaded
 */
type MediaFile = {
  id: string;
  fileUrl: string;
  // Always the upload exactly as you sent it, before and after conversion. The
  // raw file is never deleted, so this URL stays valid for as long as the asset
  // does.
  originalUrl: string;
};

/**
 * Specific media types
 */
type Video = MediaFile; // Type for video media files
type Audio = MediaFile; // Type for audio media files
type Image = MediaFile; // Type for image media files
type Document = MediaFile; // Type for document files, stored and served as uploaded
type Other = MediaFile; // Type for arbitrary files, stored and served as uploaded

/**
 * Defines the structure for different video resolutions and their corresponding URLs.
 * @typedef {Object} VideoResolutions
 * @property {string} id - Unique identifier for the video
 * @property {Object} fileUrls - Object containing URLs for various video resolutions
 * @property {string} fileUrls.360 - URL for the 360p resolution video
 * @property {string} fileUrls.480 - URL for the 480p resolution video
 * @property {string} fileUrls.720 - URL for the 720p resolution video
 * @property {string} fileUrls.1080 - URL for the 1080p resolution video
 * @property {string} originalUrl - URL that always serves the file as uploaded
 */
type VideoResolutions = {
  id: string;
  // The adaptive URL. Once converted it serves a master playlist listing every
  // resolution, so a player can switch quality on its own. Prefer this over
  // picking a fixed resolution from fileUrls.
  fileUrl: string;
  fileUrls: {
    "360": string;
    "480": string;
    "720": string;
    "1080": string;
  };
  // The source video, at whatever resolution it was uploaded in. It is not part
  // of the ladder and is not adaptive: it is the file itself.
  originalUrl: string;
};

/**
 * MediaDocker class for handling media uploads
 */
class MediaDocker {
  private _validFiles: Record<string, string[] | null> = {
    image: ["jpeg", "jpg", "png"], // Supported image file extensions
    video: ["mp4", "webm", "ogg", "mkv"], // Supported video file extensions
    audio: ["mp3", "mpeg", "wav"], // Supported audio file extensions
    document: ["pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx", "odt", "txt", "csv"],
    // null means any extension is accepted; "other" exists for exactly that.
    other: null,
  };

  // MIME types for the extensions above. The server derives the stored extension
  // from the file name rather than this value, but it still checks the type
  // against its allowlist, so it has to be plausible.
  private _mimeTypes: Record<string, string> = {
    jpeg: "image/jpeg",
    jpg: "image/jpg",
    png: "image/png",
    mp4: "video/mp4",
    webm: "video/webm",
    ogg: "video/ogg",
    mkv: "video/mkv",
    mp3: "audio/mp3",
    mpeg: "audio/mpeg",
    wav: "audio/wav",
    pdf: "application/pdf",
    doc: "application/msword",
    docx: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    xls: "application/vnd.ms-excel",
    xlsx: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    ppt: "application/vnd.ms-powerpoint",
    pptx: "application/vnd.openxmlformats-officedocument.presentationml.presentation",
    odt: "application/vnd.oasis.opendocument.text",
    txt: "text/plain",
    csv: "text/csv",
  };

  private _config = {
    mediaDockerServerKey: "", // API key for authenticating to the media server
    mediaDockerServerBaseURL: "", // Base URL for the media server API
  };

  /**
   * Uploads the file to the specified storage API endpoint.
   * This function handles the HTTP POST request to send file data to the server.
   *
   * @param {FormData} formData - The form data containing the file and any associated fields.
   * @param {"chunksStorage" | "fileStorage"} api - The API endpoint to use for the file upload.
   * @returns {Promise<Response>} - A promise that resolves to the server's response.
   */
  private async uploadToStorage(formData: FormData, api: "chunks-storage" | "file-storage") {
    return await fetch(this._config.mediaDockerServerBaseURL + `/api/v1/uploads/${api}`, {
      method: "POST", // HTTP method for the upload
      body: formData as FormData, // Form data containing the file and other fields
      headers: {
        Authorization: this._config.mediaDockerServerKey, // Authorization header with server key
      },
    });
  }

  /**
   * Main function for uploading a file to the media-docker server.
   * This function handles both single-file and chunked file uploads, depending on the file size,
   * and communicates with the media-docker server for validation and metadata handling.
   *
   * @template T - The type of the response data.
   * @param {string} filePath - The file system path to the file being uploaded.
   * @param {string} apiEndPoint - The API endpoint for the upload (e.g., 'audio', 'image', 'video').
   * @param {object} [data] - Optional data for file upload, which can include additional metadata.
   * @returns {Promise<Result<T>>} - A promise that resolves to the result containing the server response data.
   */
  private async uploadFileToMediaDockerServer<T>(
    filePath: string,
    apiEndPoint: string,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    data?: any
  ): Promise<Result<T>> {
    // Check if the server key is set, ensuring a valid connection to the media-docker server.
    if (this._config.mediaDockerServerKey === "") {
      throw new Error("mediaDocker is not connected"); // Error if server key is missing.
    }

    // Extract the file name and extension. The server stores the file under this
    // extension, which is why it is sent with every request: deriving it from the
    // MIME type does not work for document formats.
    const fileName = filePath.split(/[/\\]/).pop() as string;
    const ext = fileName.includes(".") ? (fileName.split(".").pop() as string).toLowerCase() : "";

    // Map the endpoint to the storage category the server files it under.
    const fileType = this.storageTypeFor(apiEndPoint);

    // Validate the extension against the category's allowlist. A null allowlist
    // ("other") accepts anything.
    const allowed = this._validFiles[fileType];
    if (allowed) {
      if (!ext) {
        throw new Error("Invalid file type: File extension is missing");
      }
      if (!allowed.includes(ext)) {
        throw new Error(`Invalid file type: ${ext} is not allowed for ${apiEndPoint} endpoint`);
      }
    }

    const mimeType = this._mimeTypes[ext] ?? "application/octet-stream";

    // Set the size for each file chunk (2 MB for chunked uploads).
    // NOTE: this must match helper.Constants.MaxChunkSize on the server.
    const CHUNK_SIZE = 2 * 1024 * 1024; // 2 MB chunk size.
    const stats = fs.statSync(filePath); // Retrieve the file size.
    const totalChunks = Math.ceil(stats.size / CHUNK_SIZE); // Calculate the total number of chunks.
    let id = ""; // The asset id assigned by the server.

    if (totalChunks <= 1) {
      // If the file size is less than or equal to 2 MB, upload the file in a single request.
      const content = await fsp.readFile(filePath); // Read the entire file content.
      const formData = new FormData();
      formData.append(fileType + "File", new Blob([content], { type: mimeType })); // Append file content to FormData.
      formData.append("type", fileType); // Append file type to FormData.
      formData.append("fileName", fileName); // Used by the server to derive the stored extension.
      const response = await this.uploadToStorage(formData, "file-storage"); // Send file to the file storage API.
      const resData = await response.json(); // Parse the server's JSON response.

      // Handle errors in the server response, if any.
      if (response.status !== 200) {
        throw new Error("message" in resData ? resData.message : "unknown");
      }

      // Store the asset id returned by the server.
      id = resData.data.id;
    } else {
      // If the file is larger than 2 MB, perform chunked uploads.
      const fileStream = fs.createReadStream(filePath, { highWaterMark: CHUNK_SIZE }); // Create a stream to read file chunks.

      const fileStatus: FileStatus = {
        type: fileType, // Set file type for the upload.
        status: "start", // Initial upload status.
        chunk: 0, // Starting chunk index.
        fileName: fileName, // Original file name.
      };

      // Iterate over each chunk of the file and upload it.
      for await (const chunk of fileStream) {
        const formData = new FormData(); // FormData object for the current chunk.
        formData.append(`${fileType}File`, new Blob([chunk], { type: mimeType })); // Append the current chunk.

        // Set the file status for the last chunk to 'completed'.
        if (fileStatus.chunk === totalChunks - 1) {
          fileStatus.status = "completed";
        }

        // Append fileStatus fields (e.g., id, status) to the formData for the current upload.
        Object.keys(fileStatus).forEach((key) => {
          const value = fileStatus[key as keyof FileStatus];
          if (value !== null && value !== undefined) {
            formData.append(key, `${value}`);
          }
        });

        // Upload the current chunk to the chunks-storage API.
        const response = await this.uploadToStorage(formData, "chunks-storage");
        const resData = await response.json();

        // Handle errors in the server response, if any.
        if (response.status !== 200) {
          fileStream.close(); // Close the file stream on error.
          throw new Error("message" in resData ? resData.message : "unknown");
        }

        // The first chunk mints the asset id; every later request reuses it.
        if (fileStatus.chunk === 0) {
          fileStatus.status = "uploading";
          fileStatus.id = resData.data.id;
          id = resData.data.id;
        }

        // Increment the chunk index for the next iteration.
        fileStatus.chunk++;
      }

      fileStream.close(); // Close the file stream after the chunked upload is complete.
    }

    // Claim the stored upload and, where applicable, queue its conversion.
    data = data || {}; // Initialize an empty object if no data is provided.
    data.id = id;

    // Send the final metadata (including the asset id) to the media-docker server.
    const response = await fetch(this._config.mediaDockerServerBaseURL + `/api/v1/uploads/${apiEndPoint}`, {
      method: "POST", // HTTP method for sending metadata.
      body: JSON.stringify(data), // Send the metadata as JSON.
      headers: {
        "Content-Type": "application/json", // Set content type to JSON.
        Authorization: this._config.mediaDockerServerKey, // Include the server key in the headers.
      },
    });

    const resData = await response.json(); // Parse the server's response.
    if (response.status !== 201) {
      throw new Error("message" in resData ? resData.message : "unknown"); // Handle errors from the server.
    }

    return resData; // Return the server's response indicating successful upload.
  }

  /**
   * Maps an upload endpoint to the storage category the server files it under.
   * Note that both video endpoints share the "video" category: they differ in
   * how the file is converted, not in where it is stored.
   *
   * @param {string} apiEndPoint - The upload endpoint.
   * @returns {MediaDockerFileType} - The storage category.
   */
  private storageTypeFor(apiEndPoint: string): MediaDockerFileType {
    switch (apiEndPoint) {
      case "video":
      case "video-resolutions":
        return "video";
      case "image":
        return "image";
      case "audio":
        return "audio";
      case "document":
        return "document";
      case "other":
        return "other";
      default:
        throw new Error(`Invalid API endpoint: ${apiEndPoint}`);
    }
  }

  /**
   * Authenticates against the media-docker server.
   *
   * This is the only handshake there is: it validates the API key and stores the
   * key and base URL for every later call. Nothing is held open afterwards --
   * each upload is an ordinary HTTP request -- so there is no connection to
   * manage and nothing to tear down.
   *
   * Use `localhost` when running media Docker services locally (for development).
   * When deploying in Docker or production, use the appropriate container or server URLs.
   *
   * @param {string} mediaDockerServerKey - The API key required for authenticating
   * with the media server.
   * @param {"http://localhost:7007" | "http://media-docker-server:7007"} mediaDockerServerBaseURL -
   * The base URL for the media server API. Use `localhost` for development and
   * `media-docker-server` for Docker or production environments.
   *
   * @returns {Promise<void>} - Resolves once the key has been accepted, rejects if it has not.
   */
  async connect(
    mediaDockerServerKey: string,
    mediaDockerServerBaseURL: "http://localhost:7007" | "http://media-docker-server:7007"
  ): Promise<void> {
    // Connect to the media server using the provided API key and base URL
    const response = await fetch(mediaDockerServerBaseURL + "/api/v1/connections/connect", {
      method: "GET",
      headers: {
        Authorization: mediaDockerServerKey, // Include the API key in the Authorization header
      },
    });

    // Parse the response from the media server
    const resData = await response.json();

    // Check if the server returned a success status code (200)
    if (response.status !== 200) {
      this.log("ERROR", resData.message || "unknown"); // Log any error message returned by the server
      throw new Error(resData.message || "unknown"); // Throw an error if connection fails
    }

    // Store the media server connection details for future use
    this._config.mediaDockerServerKey = mediaDockerServerKey;
    this._config.mediaDockerServerBaseURL = mediaDockerServerBaseURL;
    this.log("INFO", "Connected to media server successfully."); // Log successful media server connection
  }

  /**
   * Log messages to console or other logging services
   * @param {"INFO" | "ERROR" | "SUCCESS"} level - Severity level of the log
   * @param {string} message - Log message
   */
  private log(level: "INFO" | "ERROR" | "SUCCESS", message: string): void {
    console.log(`[${level.toUpperCase()}] [Media-Docker] ${message}`); // Log message to console
  }

  /**
   * Upload a video file to the media server.
   *
   * fileUrl serves the upload immediately and switches to an HLS stream once the
   * consumer finishes; originalUrl keeps serving the file you sent, either way.
   *
   * @param {string} filePath - Path to the video file being uploaded
   * @param {number} [quality] - Optional quality level between 40 and 100
   * @returns {Promise<Result<Video>>} - Result containing video upload response
   */
  async uploadVideo(filePath: string, quality?: number): Promise<Result<Video>> {
    if (quality && (quality < 40 || quality > 100)) {
      throw new Error("Quality must be between 40 and 100"); // Validate quality range
    }
    const res = await this.uploadFileToMediaDockerServer<Video>(filePath, "video", { quality });
    return res; // Return the response from the upload
  }

  /**
   * Upload video resolutions to the media server.
   *
   * Returns the adaptive fileUrl, one URL per rung of the ladder, and
   * originalUrl for the source video at its uploaded resolution -- useful when
   * the upload is higher quality than the top rung.
   *
   * @param {string} filePath - Path to the video resolutions file
   * @returns {Promise<Result<VideoResolutions>>} - Result containing video resolutions upload response
   */
  async uploadVideoResolutions(filePath: string): Promise<Result<VideoResolutions>> {
    const res = await this.uploadFileToMediaDockerServer<VideoResolutions>(filePath, "video-resolutions");
    return res; // Return the response from the upload
  }

  /**
   * Upload an image file to the media server.
   *
   * fileUrl serves a compressed JPEG once converted; originalUrl keeps serving
   * the image at its uploaded format and quality.
   *
   * @param {string} filePath - Path to the image file being uploaded
   * @param {number} [compression] - Optional ffmpeg quality level, 1 (best) to 31 (worst)
   * @returns {Promise<Result<Image>>} - Result containing image upload response
   */
  async uploadImage(filePath: string, compression?: number): Promise<Result<Image>> {
    if (compression && (compression < 1 || compression > 31)) {
      throw new Error("Compression must be between 1 and 31"); // Validate compression range
    }
    const res = await this.uploadFileToMediaDockerServer<Image>(filePath, "image", { compression });
    return res; // Return the response from the upload
  }

  /**
   * Upload an audio file to the media server.
   *
   * fileUrl serves an MP3 once converted; originalUrl keeps serving the audio as
   * uploaded, which matters when the source was lossless.
   *
   * @param {string} filePath - Path to the audio file being uploaded
   * @param {"128k" | "192k" | "256k" | "320k"} [bitrate] - Optional bitrate for the audio file
   * @returns {Promise<Result<Audio>>} - Result containing audio upload response
   */
  async uploadAudio(filePath: string, bitrate?: "128k" | "192k" | "256k" | "320k"): Promise<Result<Audio>> {
    const res = await this.uploadFileToMediaDockerServer<Audio>(filePath, "audio", { bitrate });
    return res; // Return the response from the upload
  }

  /**
   * Upload a document to the media server.
   *
   * Documents are stored and served exactly as uploaded; no conversion happens,
   * so the URL is final from the moment the upload completes. They are always
   * served as a download rather than rendered in the browser. originalUrl is
   * returned for consistency and addresses the same bytes as fileUrl.
   *
   * @param {string} filePath - Path to the document being uploaded
   * @returns {Promise<Result<Document>>} - Result containing the upload response
   */
  async uploadDocument(filePath: string): Promise<Result<Document>> {
    const res = await this.uploadFileToMediaDockerServer<Document>(filePath, "document");
    return res; // Return the response from the upload
  }

  /**
   * Upload a file of any type to the media server.
   *
   * Like documents, these are stored and served exactly as uploaded and are
   * always served as a download. Use this for files that do not fit the other
   * categories; no extension allowlist is applied.
   *
   * @param {string} filePath - Path to the file being uploaded
   * @returns {Promise<Result<Other>>} - Result containing the upload response
   */
  async uploadOther(filePath: string): Promise<Result<Other>> {
    const res = await this.uploadFileToMediaDockerServer<Other>(filePath, "other");
    return res; // Return the response from the upload
  }

  /**
   * Delete a media file from the server
   * @param {string} id - ID of the media file to be deleted
   * @param {MediaDockerFileType} type - Type of the media file
   * @returns {Promise<void>} - Resolves when deletion is successful
   */
  async deleteFile(id: string, type: MediaDockerFileType): Promise<void> {
    if (this._config.mediaDockerServerKey === "") {
      throw new Error("mediaDocker is not connected"); // Ensure the server key is set
    }

    const response = await fetch(this._config.mediaDockerServerBaseURL + "/api/v1/destroys/delete-file", {
      method: "DELETE", // HTTP method for deletion
      body: JSON.stringify({ id: id, type: type }), // Body containing the file ID and type
      headers: {
        Authorization: this._config.mediaDockerServerKey, // Authorization header with server key
        "Content-Type": "application/json", // Set content type to JSON
      },
    });

    if (response.status === 200) {
      return; // Just resolve if deletion was successful
    }

    const resData = await response.json();
    throw Error("message" in resData ? resData.message : "unknown"); // Handle errors from the server
  }
}

// Create an instance of the MediaDocker class
const mediaDocker = new MediaDocker();
export default mediaDocker; // Export the instance for external use
