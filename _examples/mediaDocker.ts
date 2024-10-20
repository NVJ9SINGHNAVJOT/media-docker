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

// Importing file system promises for handling file operations
import * as fs from "fs/promises";
// Importing Kafka and Consumer classes from kafkajs library for handling Kafka messaging
// Note: Ensure that the kafkajs library is installed in your project by running:
// npm install kafkajs or yarn add kafkajs
import { Kafka, Consumer, logLevel } from "kafkajs";

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
    serverKey: "", // API key for authenticating to the media server
    serverBaseUrl: "", // Base URL for the media server API
  };

  private kafka: Kafka; // Kafka instance for message handling
  private consumer: Consumer; // Kafka consumer for processing messages

  /**
   * Initializes a Kafka instance with client ID, brokers, and retry settings,
   * and creates a consumer with the specified group ID and session configurations.
   * This ensures that the consumer can handle message processing within a 1-minute window.
   * The default log level is set to WARN.
   */
  constructor() {
    // Initialize Kafka instance with client ID and broker addresses
    this.kafka = new Kafka({
      clientId: "media-docker-response-client", // Unique identifier for this Kafka client
      brokers: ["media-docker-kafka-0:9092", "media-docker-kafka-1:9092", "media-docker-kafka-2:9092"], // Kafka brokers to connect
      retry: {
        retries: 5, // Number of retries for connection failure
      },
      logLevel: logLevel.WARN, // Logging Kafka events
    });

    // Initialize Kafka consumer with specified group ID and heartbeat settings
    this.consumer = this.kafka.consumer({
      groupId: "media-docker-response-group", // Consumer group for coordinated consumption
      heartbeatInterval: 3000, // Send heartbeats every 3 seconds to indicate alive status
      sessionTimeout: 60000, // Session timeout of 60 seconds
    });
  }

  /**
   * Main function for uploading a file to the media-docker server
   * @template T
   * @param {string} filePath - Path to the file being uploaded
   * @param {string} apiEndPoint - API endpoint for the upload
   * @param {Object<string, unknown>} [formValues] - Optional additional form values
   * @returns {Promise<Result<T>>} - Result containing the response data
   */
  private async uploadFileToMediaDockerServer<T>(
    filePath: string,
    apiEndPoint: string,
    formValues?: { [key: string]: unknown }
  ): Promise<Result<T>> {
    if (this._config.serverKey === "") {
      throw new Error("mediaDocker is not connected"); // Ensure the server key is set
    }

    // Determine file extension from file path
    let fileType = filePath.split(".").pop();
    const ext = fileType; // Store file extension for MIME type

    if (!fileType) {
      throw new Error("Invalid file type: File extension is missing");
    }

    // Check based on apiEndPoint and ensure fileType matches for each case
    if (apiEndPoint === "audio") {
      if (!this._validFiles.audio.includes(fileType)) {
        throw new Error(`Invalid file type: ${fileType} is not allowed for audio endpoint`);
      }
      fileType = "audio"; // Set file type as audio
    } else if (apiEndPoint === "image") {
      if (!this._validFiles.image.includes(fileType)) {
        throw new Error(`Invalid file type: ${fileType} is not allowed for image endpoint`);
      }
      fileType = "image"; // Set file type as image
    } else if (apiEndPoint === "video") {
      if (!this._validFiles.video.includes(fileType)) {
        throw new Error(`Invalid file type: ${fileType} is not allowed for video endpoint`);
      }
      fileType = "video"; // Set file type as video
    } else if (apiEndPoint === "videoResolutions") {
      if (!this._validFiles.video.includes(fileType)) {
        throw new Error(`Invalid file type: ${fileType} is not allowed for videoResolutions endpoint`);
      }
      fileType = "video"; // Set file type as video
    } else {
      // If the apiEndPoint is not one of the above, throw an error
      throw new Error(`Invalid API endpoint: ${apiEndPoint}`);
    }

    // Read the file content
    const content = await fs.readFile(filePath);
    const formData = new FormData();
    // Append file content to formData
    formData.append(fileType + "File", new Blob([content], { type: `${fileType}/${ext}` }));

    // Add optional formValues to formData
    if (formValues) {
      Object.keys(formValues).forEach((key) => {
        const value = formValues[key];
        if (value !== null && value !== undefined) {
          formData.append(key, `${value}`);
        }
      });
    }

    // Send the file to the media-docker server
    const response = await fetch(this._config.serverBaseUrl + `/api/v1/uploads/${apiEndPoint}`, {
      method: "POST", // HTTP method for the upload
      body: formData as FormData, // Form data containing the file and other fields
      headers: {
        Authorization: this._config.serverKey, // Authorization header with server key
      },
    });

    const resData = await response.json();
    if (response.status !== 201) {
      throw new Error("message" in resData ? resData.message : "unknown"); // Handle errors from the server
    }
    return resData; // Return the server response
  }

  /**
   * Connect to both the media server and Kafka broker with provided credentials.
   * This function handles the connections to both services, ensuring that the
   * media server is authenticated and the Kafka consumer is ready to receive messages.
   * If either connection fails, an error is thrown.
   *
   * Once Kafka is connected, the consumer will run in the background, continuously
   * listening for incoming messages. If the connection to Kafka fails, it will retry
   * up to 10 times before giving up.
   *
   * The provided message handler function will be called for each
   * incoming Kafka message, allowing for custom processing logic to be applied.
   * This ensures that the application can react to messages as they arrive
   * in real-time.
   *
   * @param {string} serverKey - API key for the media server.
   * @param {string} serverBaseURL - Base URL for the media server API.
   * @param {(message: MediaDockerMessage) => Promise<void>} messageHandler -
   * Function to handle incoming Kafka messages. This function is called for each
   * message received from the Kafka topic.
   *
   * @returns {Promise<void>} - Resolves when both the media server and Kafka are
   * successfully connected, or rejects if any connection fails.
   */
  async connectMediaDockerAndKafka(
    serverKey: string,
    serverBaseURL: string,
    messageHandler: (message: MediaDockerMessage) => Promise<void>
  ): Promise<void> {
    // Connect to the media server using the provided API key and base URL
    const response = await fetch(serverBaseURL + "/api/v1/connections/connect", {
      method: "GET",
      headers: {
        Authorization: serverKey, // Set the Authorization header with the serverKey
      },
    });

    // Parse the response JSON from the media server
    const resData = await response.json();

    // Check if the response status is not 200 (success). If it fails, log and throw an error.
    if (response.status !== 200) {
      this.log("error", resData.message || "unknown"); // Log the error message (if any) from the server response
      throw new Error("message" in resData ? resData.message : "unknown"); // Throw the error for further handling
    }

    // If successful, store the connection details (serverKey and serverBaseURL)
    this._config.serverKey = serverKey;
    this._config.serverBaseUrl = serverBaseURL;
    this.log("info", "Connected to media server successfully."); // Log successful connection to the media server

    // Connect to Kafka and set up message handling for incoming messages
    await this.consumer.connect(); // Attempt to connect to the Kafka broker
    this.log("info", "Connected to Kafka successfully."); // Log successful connection to Kafka

    // Start handling messages from Kafka with the provided messageHandler function
    this.handleConsumer(messageHandler);
    this.log("info", "Starting Kafka message consumption now."); // Log that message consumption has begun
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
    try {
      await this.consumer.disconnect(); // Disconnect the Kafka consumer
      this.log("info", "Disconnected from Kafka successfully."); // Log successful disconnection
    } catch (error) {
      this.log("error", `Error during Kafka disconnection: ${error}`); // Log any errors during disconnection
    }
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
          this.log("info", "Reconnected to Kafka successfully."); // Log successful reconnection
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
                this.log("error", `Received null or empty message from topic "${topic}", partition "${partition}"`);
                return; // Skip processing if message is invalid
              }

              // Parse message value
              value = JSON.parse(message.value.toString()) as MediaDockerMessage;

              // Call the provided message handler for custom processing
              await messageHandler(value);
            } catch (error) {
              // Log errors that occur during message processing, including the parsed value if available
              this.log(
                "error",
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
        this.log("error", `Connection attempt failed: ${error}`);
        this.log("info", `Retrying connection (${attempt + 1}/10)...`); // Log retry attempt
        await this.delay(2000); // Wait before retrying
      }
    }

    // Log failure if maximum connection attempts are reached
    this.log("error", "Max connection attempts reached. Could not connect to Kafka.");
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
   * @param {"info" | "error" | "success"} level - Severity level of the log
   * @param {string} message - Log message
   */
  private log(level: "info" | "error" | "success", message: string): void {
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
    if (this._config.serverKey === "") {
      throw new Error("mediaDocker is not connected"); // Ensure the server key is set
    }

    const response = await fetch(this._config.serverBaseUrl + "/api/v1/destroys/deleteFile", {
      method: "DELETE", // HTTP method for deletion
      body: JSON.stringify({ id: id, type: type }), // Body containing the file ID and type
      headers: {
        Authorization: this._config.serverKey, // Authorization header with server key
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
