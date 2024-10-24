/* eslint-disable no-unused-vars */
/*
  IMPORTANT: Please do not modify this file; copy and paste it as-is into your project.

  This file contains the core logic for uploading files to the Media-Docker service.
  It handles the file upload process and manages the interaction with the Media-Docker API.

  Upload Process:
  - If the file upload is successful, the response will include a unique file ID (UUID v4 format)
    along with a file URL or multiple URLs (in case the media generates multiple output formats).
  - Once the file is uploaded, Media-Docker will process it according to the file type
    (e.g., image conversion, video transcoding, etc.).
  
  Post-Upload Response:
  - After processing, your server will receive a message from Media-Docker via Kafka.
  - This message will include the following fields:
  
    ```typescript
    type MediaDockerMessage = {
      id: string; // Unique file identifier (UUID v4 format)
      fileType: "image" | "video" | "videoResolutions" | "audio"; // Type of the uploaded file
      status: "completed" | "failed"; // Status of the file processing
    };
    ```

  - The `id` will match the UUID v4 of the uploaded file.
  - The `fileType` will specify the media type, such as "image", "video", "videoResolutions", or "audio".
  - The `status` field will indicate whether the processing was successful ("completed") or failed ("failed").

  Handling the Response:
  - Based on the `status` ("completed" or "failed"), you can implement further logic in your system:
    - For "completed", you may update your database, notify users, or proceed with further actions.
    - For "failed", you can handle retries or report errors in your application.
  
  Note: This file is designed to ensure smooth integration with Media-Docker. If modifications are 
  necessary, please review them carefully to avoid breaking the upload and response processing functionality.
*/

// Importing file system for handling file operations
import * as fs from "fs";
import * as fsp from "fs/promises";
// Importing Kafka and Consumer classes from kafkajs library for handling Kafka messaging
// Note: Ensure that the kafkajs library is installed in your project by running:
// npm install kafkajs or yarn add kafkajs
import { Kafka, Consumer, logLevel } from "kafkajs";

type FileStatus = {
  type: string;
  status: string;
  chunk: number;
  chunkId?: string;
};

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
 */
type MediaFile = {
  id: string;
  fileUrl: string;
};

/**
 * Specific media types
 */
type Video = MediaFile; // Type for video media files
type Audio = MediaFile; // Type for audio media files
type Image = MediaFile; // Type for image media files

/**
 * Defines the structure for different video resolutions and their corresponding URLs.
 * @typedef {Object} VideoResolutions
 * @property {string} id - Unique identifier for the video
 * @property {Object} fileUrls - Object containing URLs for various video resolutions
 * @property {string} fileUrls.360 - URL for the 360p resolution video
 * @property {string} fileUrls.480 - URL for the 480p resolution video
 * @property {string} fileUrls.720 - URL for the 720p resolution video
 * @property {string} fileUrls.1080 - URL for the 1080p resolution video
 */
type VideoResolutions = {
  id: string;
  fileUrls: {
    "360": string;
    "480": string;
    "720": string;
    "1080": string;
  };
};

/**
 * Message type for Kafka messages
 * @typedef {Object} MediaDockerMessage
 * @property {string} id - Unique identifier for the message
 * @property {"image" | "video" | "videoResolutions" | "audio"} fileType - Type of media file
 * @property {"completed" | "failed"} status - Status of the media processing
 */
export type MediaDockerMessage = {
  id: string;
  fileType: "image" | "video" | "videoResolutions" | "audio";
  status: "completed" | "failed";
};

/**
 * MediaDocker class for handling media uploads and Kafka messages
 */
class MediaDocker {
  private _validFiles = {
    image: ["jpeg", "jpg", "png"], // Supported image file extensions
    video: ["mp4", "webm", "ogg", "mkv"], // Supported video file extensions
    audio: ["mp3", "mpeg", "wav"], // Supported audio file extensions
  };

  private _config = {
    mediaDockerServerKey: "", // API key for authenticating to the media server
    mediaDockerServerBaseURL: "", // Base URL for the media server API
  };

  private kafka!: Kafka; // Kafka instance for message handling
  private consumer!: Consumer; // Kafka consumer for processing messages

  /**
   * Uploads the file to the specified storage API endpoint.
   * This function handles the HTTP POST request to send file data to the server.
   *
   * @param {FormData} formData - The form data containing the file and any associated fields.
   * @param {"chunksStorage" | "fileStorage"} api - The API endpoint to use for the file upload.
   * @returns {Promise<Response>} - A promise that resolves to the server's response.
   */
  private async uploadToStorage(formData: FormData, api: "chunksStorage" | "fileStorage") {
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

    // Extract the file extension from the file path to determine its type.
    let fileType = filePath.split(".").pop();
    const ext = fileType; // Save the file extension for setting MIME type later.

    // Validate that the file has an extension, throw an error if missing.
    if (!fileType) {
      throw new Error("Invalid file type: File extension is missing");
    }

    // Validate the file type based on the provided API endpoint (audio, image, video, etc.).
    if (apiEndPoint === "audio") {
      if (!this._validFiles.audio.includes(fileType)) {
        throw new Error(`Invalid file type: ${fileType} is not allowed for audio endpoint`);
      }
      fileType = "audio";
    } else if (apiEndPoint === "image") {
      if (!this._validFiles.image.includes(fileType)) {
        throw new Error(`Invalid file type: ${fileType} is not allowed for image endpoint`);
      }
      fileType = "image";
    } else if (apiEndPoint === "video") {
      if (!this._validFiles.video.includes(fileType)) {
        throw new Error(`Invalid file type: ${fileType} is not allowed for video endpoint`);
      }
      fileType = "video";
    } else if (apiEndPoint === "videoResolutions") {
      if (!this._validFiles.video.includes(fileType)) {
        throw new Error(`Invalid file type: ${fileType} is not allowed for videoResolutions endpoint`);
      }
      fileType = "video";
    } else {
      // Throw an error for any unsupported API endpoints.
      throw new Error(`Invalid API endpoint: ${apiEndPoint}`);
    }

    // Set the size for each file chunk (2 MB for chunked uploads).
    const CHUNK_SIZE = 2 * 1024 * 1024; // 2 MB chunk size.
    const stats = fs.statSync(filePath); // Retrieve the file size.
    const totalChunks = Math.ceil(stats.size / CHUNK_SIZE); // Calculate the total number of chunks.
    let uuidFilename = ""; // Variable to store the UUID filename returned from the server.

    if (totalChunks <= 1) {
      // If the file size is less than or equal to 2 MB, upload the file in a single request.
      const content = await fsp.readFile(filePath); // Read the entire file content.
      const formData = new FormData();
      formData.append(fileType + "File", new Blob([content], { type: `${fileType}/${ext}` })); // Append file content to FormData.
      formData.append("type", fileType); // Append file type to FormData.
      const response = await this.uploadToStorage(formData, "fileStorage"); // Send file to the file storage API.
      const resData = await response.json(); // Parse the server's JSON response.

      // Handle errors in the server response, if any.
      if (response.status !== 200) {
        throw new Error("message" in resData ? resData.message : "unknown");
      }

      // Store the UUID filename returned by the server for future reference.
      uuidFilename = resData.data.uuidFilename;
    } else {
      // If the file is larger than 2 MB, perform chunked uploads.
      const fileStream = fs.createReadStream(filePath, { highWaterMark: CHUNK_SIZE }); // Create a stream to read file chunks.

      const fileStatus: FileStatus = {
        type: fileType, // Set file type for the upload.
        status: "start", // Initial upload status.
        chunk: 0, // Starting chunk index.
      };

      // Iterate over each chunk of the file and upload it.
      for await (const chunk of fileStream) {
        const formData = new FormData(); // FormData object for the current chunk.
        formData.append(`${fileType}File`, new Blob([chunk], { type: `${fileType}/${ext}` })); // Append the current chunk.

        // Set the file status for the last chunk to 'completed'.
        if (fileStatus.chunk === totalChunks - 1) {
          fileStatus.status = "completed";
        }

        // Append fileStatus fields (e.g., chunkId, status) to the formData for the current upload.
        Object.keys(fileStatus).forEach((key) => {
          const value = fileStatus[key as keyof FileStatus];
          if (value !== null && value !== undefined) {
            formData.append(key, `${value}`);
          }
        });

        // Upload the current chunk to the chunksStorage API.
        const response = await this.uploadToStorage(formData, "chunksStorage");
        const resData = await response.json();

        // Handle errors in the server response, if any.
        if (response.status !== 200) {
          fileStream.close(); // Close the file stream on error.
          throw new Error("message" in resData ? resData.message : "unknown");
        }

        // For the first chunk, set the status to 'uploading' and store the new chunkId.
        if (fileStatus.chunk === 0) {
          fileStatus.status = "uploading";
          fileStatus.chunkId = resData.data.newChunkId;
        } else if (fileStatus.chunk === totalChunks - 1) {
          // For the last chunk, store the UUID filename from the server response.
          uuidFilename = resData.data.uuidFilename;
        }

        // Increment the chunk index for the next iteration.
        fileStatus.chunk++;
      }

      fileStream.close(); // Close the file stream after the chunked upload is complete.
    }

    // Ensure the data object exists and add the UUID filename for the final metadata upload.
    data = data || {}; // Initialize an empty object if no data is provided.
    data.uuidFilename = uuidFilename; // Add the UUID filename to the data.

    // Send the final metadata (including the UUID filename) to the media-docker server.
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
   * Establishes connections to both the media server and the Kafka broker.
   * This function handles the authentication to the media server and sets up
   * the Kafka consumer to process incoming messages from the specified topics.
   *
   * The connection to the media server is authenticated using the provided API key
   * and base URL, while the Kafka consumer connects to the specified brokers and
   * listens for messages within the defined consumer group.
   *
   * Use `localhost` when running media Docker services locally (for development).
   * When deploying in Docker or production, use the appropriate container or server URLs.
   *
   * Upon successful connection to Kafka, the consumer will continuously listen
   * for incoming messages. If any connection fails, an error is thrown and
   * logged. If Kafka fails to connect, it retries up to 5 times before giving up.
   *
   * The provided message handler function will be invoked for each Kafka message
   * received, enabling custom message processing logic to be executed in real-time.
   *
   * @param {string} mediaDockerServerKey - The API key required for authenticating
   * with the media server.
   * @param {"http://localhost:7007" | "http://media-docker-server:7007"} mediaDockerServerBaseURL -
   * The base URL for the media server API. Use `localhost` for development and
   * `media-docker-server` for Docker or production environments.
   * @param {"localhost:9092" | "media-docker-kafka-0:9092,media-docker-kafka-1:9092,media-docker-kafka-2:9092"} kafkaBrokers -
   * Comma-separated list of Kafka broker addresses. Use `localhost` for development
   * and Docker container addresses for Docker or production environments.
   * @param {(message: MediaDockerMessage) => Promise<void>} messageHandler -
   * Callback function to process each incoming Kafka message.
   *
   * @returns {Promise<void>} - Resolves once both the media server and Kafka broker
   * are connected, or rejects if a connection fails.
   */
  async connectMediaDockerAndKafka(
    mediaDockerServerKey: string,
    mediaDockerServerBaseURL: "http://localhost:7007" | "http://media-docker-server:7007",
    kafkaBrokers: "localhost:9092" | "media-docker-kafka-0:9092,media-docker-kafka-1:9092,media-docker-kafka-2:9092",
    messageHandler: (message: MediaDockerMessage) => Promise<void>
  ): Promise<void> {
    // Initialize the Kafka instance with client ID and broker addresses
    this.kafka = new Kafka({
      clientId: "media-docker-response-client", // Unique client ID for Kafka
      brokers: kafkaBrokers.split(","), // Connect to the specified Kafka brokers
      retry: {
        retries: 5, // Retry Kafka connection 5 times on failure
      },
      logLevel: logLevel.WARN, // Log Kafka events at WARN level
    });

    // Initialize the Kafka consumer with the specified group ID and heartbeat settings
    this.consumer = this.kafka.consumer({
      groupId: "media-docker-response-group", // Consumer group for coordinated message consumption
      heartbeatInterval: 3000, // Heartbeat interval to maintain connection (3 seconds)
      sessionTimeout: 60000, // Session timeout duration (60 seconds)
    });

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

    // Connect the Kafka consumer to the Kafka brokers
    await this.consumer.connect();
    this.log("INFO", "Connected to Kafka successfully."); // Log successful Kafka connection

    // Start processing incoming Kafka messages using the provided message handler
    this.handleConsumer(messageHandler);
    this.log("INFO", "Kafka message consumption has started."); // Log the start of Kafka message consumption
  }

  /**
   * Disconnect from Kafka, shutting down the consumer gracefully.
   * This method ensures that any ongoing message processing is completed
   * before disconnecting. It logs the disconnection status and any errors
   * that occur during the process.
   *
   * @returns {Promise<void>} - Resolves when the consumer is successfully disconnected,
   * or rejects if an error occurs during disconnection.
   */
  async disconnect(): Promise<void> {
    await this.consumer.disconnect(); // Disconnect the Kafka consumer
    this.log("INFO", "Disconnected from Kafka successfully."); // Log successful disconnection
  }

  /**
   * Handle incoming Kafka messages by connecting to the broker and
   * processing messages as they arrive. This function attempts to connect
   * to Kafka up to 10 times if the initial connection fails. If successful,
   * it subscribes to the specified topic and runs a background process that
   * continuously listens for messages, invoking the provided message handler
   * for each one.
   *
   * If an error occurs while processing a message, it logs the error
   * information for debugging purposes. The retry mechanism ensures that
   * transient network issues do not prevent the application from connecting
   * to Kafka, providing robustness in message handling.
   *
   * @param {(message: MediaDockerMessage) => Promise<void>} messageHandler -
   * Function to handle incoming messages. This function is called for each
   * message received from the Kafka topic.
   *
   * @returns {Promise<void>} - Resolves when the consumer is running
   * and actively listening for messages.
   */
  private async handleConsumer(messageHandler: (message: MediaDockerMessage) => Promise<void>): Promise<void> {
    // Attempt to connect to Kafka up to 10 times
    for (let attempt = 0; attempt < 10; attempt++) {
      try {
        if (attempt > 0) {
          // Reconnect to the Kafka broker only on subsequent attempts
          await this.consumer.connect();
          this.log("INFO", "Reconnected to Kafka successfully."); // Log successful reconnection
        }

        // Subscribe to the specified topic
        await this.consumer.subscribe({ topics: ["media-docker-files-response"], fromBeginning: true });

        // Run the consumer to process incoming messages
        await this.consumer.run({
          eachMessage: async ({ topic, partition, message }) => {
            // Declare value and initialize to null
            let value: MediaDockerMessage | null = null;

            try {
              // Check if the message value is null or empty
              if (!message.value) {
                this.log("ERROR", `Received null or empty message from topic "${topic}", partition "${partition}"`);
                return; // Skip processing if message is invalid
              }

              // Parse message value
              value = JSON.parse(message.value.toString()) as MediaDockerMessage;

              // Call the provided message handler for custom processing
              await messageHandler(value);
            } catch (error) {
              // Log errors that occur during message processing, including the parsed value if available
              this.log(
                "ERROR",
                `Error processing message from topic "${topic}", partition "${partition}": ${error}. Parsed message value: ${value ? JSON.stringify(value) : "Not available"}`
              );
              this.delay(1000); // Delay for 1 second
            }
          },
        });

        // Successfully connected and running
        return;
      } catch (error) {
        // Log the connection error
        this.log("ERROR", `Connection attempt failed: ${error}`);
        this.log("INFO", `Retrying connection (${attempt + 1}/10)...`); // Log retry attempt
        await this.delay(2000); // Wait before retrying
      }
    }

    // Log failure if maximum connection attempts are reached
    this.log("ERROR", "Max connection attempts reached. Could not connect to Kafka.");
  }

  /**
   * Delay execution for a specified number of milliseconds
   * @param {number} ms - Delay in milliseconds
   * @returns {Promise<void>} - Resolves after the delay
   */
  private delay(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
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
   * Upload a video file to the media server
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
   * Upload video resolutions to the media server
   * @param {string} filePath - Path to the video resolutions file
   * @returns {Promise<Result<VideoResolutions>>} - Result containing video resolutions upload response
   */
  async uploadVideoResolutions(filePath: string): Promise<Result<VideoResolutions>> {
    const res = await this.uploadFileToMediaDockerServer<VideoResolutions>(filePath, "videoResolutions");
    return res; // Return the response from the upload
  }

  /**
   * Upload an image file to the media server
   * @param {string} filePath - Path to the image file being uploaded
   * @returns {Promise<Result<Image>>} - Result containing image upload response
   */
  async uploadImage(filePath: string): Promise<Result<Image>> {
    const res = await this.uploadFileToMediaDockerServer<Image>(filePath, "image");
    return res; // Return the response from the upload
  }

  /**
   * Upload an audio file to the media server
   * @param {string} filePath - Path to the audio file being uploaded
   * @param {"128k" | "192k" | "256k" | "320k"} [bitrate] - Optional bitrate for the audio file
   * @returns {Promise<Result<Audio>>} - Result containing audio upload response
   */
  async uploadAudio(filePath: string, bitrate?: "128k" | "192k" | "256k" | "320k"): Promise<Result<Audio>> {
    const res = await this.uploadFileToMediaDockerServer<Audio>(filePath, "audio", { bitrate });
    return res; // Return the response from the upload
  }

  /**
   * Delete a media file from the server
   * @param {string} id - ID of the media file to be deleted
   * @param {"image" | "video" | "audio"} type - Type of the media file
   * @returns {Promise<void>} - Resolves when deletion is successful
   */
  async deleteFile(id: string, type: "image" | "video" | "audio"): Promise<void> {
    if (this._config.mediaDockerServerKey === "") {
      throw new Error("mediaDocker is not connected"); // Ensure the server key is set
    }

    const response = await fetch(this._config.mediaDockerServerBaseURL + "/api/v1/destroys/deleteFile", {
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
