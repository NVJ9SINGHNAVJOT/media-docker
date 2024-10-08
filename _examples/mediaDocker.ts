/*
  NOTE: do not edit this file just copy paste to your project.
  this file contains file uploading logic to Media-Docker.
*/

import * as fs from "fs/promises";

type Result<T> = { message: string; data: T };

class MediaDocker {
  // only valid files will be allowed to upload
  private _validFiles = {
    image: ["jpeg", "jpg", "png"],
    video: ["mp4", "webm", "ogg", "mkv"],
    audio: ["mp3", "mpeg", "wav"],
  };

  private _config = {
    serverKey: "",
    serverBaseUrl: "",
  };
  // main fucntion for uploading file to media-docker server
  private async uploadFileToMediaDockerServer<T>(
    filePath: string,
    apiEndPoint: string,
    formValues?: { [key: string]: unknown }
  ): Promise<Result<T>> {
    if (this._config.serverKey === "") {
      throw new Error("mediaDocker is not connected");
    }
    // file extension
    let fileType = filePath.split(".").pop();
    const ext = fileType;
    if (!fileType) {
      throw new Error("invalid file type");
    } else if (this._validFiles.audio.includes(fileType)) {
      fileType = "audio";
    } else if (this._validFiles.image.includes(fileType)) {
      fileType = "image";
    } else if (this._validFiles.video.includes(fileType)) {
      fileType = "video";
    } else {
      throw new Error("invalid file type");
    }

    const content = await fs.readFile(filePath);

    const formData = new FormData();
    formData.append(fileType + "File", new Blob([content], { type: `${fileType}/${ext}` }));

    // Add formValues to formData
    if (formValues) {
      Object.keys(formValues).forEach((key) => {
        formData.append(key, `${formValues[key]}`);
      });
    }

    const response = await fetch(this._config.serverBaseUrl + `/api/v1/uploads/${apiEndPoint}`, {
      method: "POST",
      body: formData as FormData,
      headers: {
        Authorization: this._config.serverKey,
      },
    });

    const resData = await response.json();
    if (response.status !== 201) {
      throw new Error("message" in resData ? resData.message : "unknown");
    }

    return resData;
  }

  async connect(serverKey: string, serverBaseURL: string): Promise<void> {
    const response = await fetch(serverBaseURL + "/api/v1/connections/connect", {
      method: "GET",
      headers: {
        Authorization: serverKey,
      },
    });

    const resData = await response.json();

    if (response.status !== 200) {
      throw new Error("message" in resData ? resData.message : "unknown");
    }

    this._config.serverKey = serverKey;
    this._config.serverBaseUrl = serverBaseURL;
  }

  // In optional parameter quality can be in between 40 and 100
  async uploadVideo(filePath: string, quality?: number): Promise<string> {
    if (quality && (quality < 40 || quality > 100)) {
      throw new Error("Quality must be between 40 and 100");
    }
    const res = await this.uploadFileToMediaDockerServer<{ fileUrl: string }>(filePath, "video", { quality });
    return res.data.fileUrl;
  }

  async uploadVideoResolutions(filePath: string): Promise<{
    "360": string;
    "480": string;
    "720": string;
    "1080": string;
  }> {
    const res = await this.uploadFileToMediaDockerServer<{
      "360": string;
      "480": string;
      "720": string;
      "1080": string;
    }>(filePath, "videoResolutions");
    return res.data;
  }

  async uploadImage(filePath: string): Promise<string> {
    const res = await this.uploadFileToMediaDockerServer<{ fileUrl: string }>(filePath, "image");
    return res.data.fileUrl;
  }

  async uploadAudio(filePath: string, bitrate?: "128k" | "192k" | "256k" | "320k"): Promise<string> {
    const res = await this.uploadFileToMediaDockerServer<{ fileUrl: string }>(filePath, "audio", { bitrate });
    return res.data.fileUrl;
  }

  async deleteFile(id: string, type: "image" | "video" | "audio"): Promise<boolean> {
    if (this._config.serverKey === "") {
      throw new Error("mediaDocker is not connected");
    }
    const response = await fetch(this._config.serverBaseUrl + "/api/v1/destroys/deleteFile", {
      method: "DELETE",
      body: JSON.stringify({ id: id, type: type }),
      headers: {
        Authorization: this._config.serverKey,
        "Content-Type": "application/json",
      },
    });

    if (response.status === 200) {
      return true;
    }

    const resData: { message: string } = await response.json();
    throw new Error(resData.message);
  }
}

const mediaDocker = new MediaDocker();
export default mediaDocker;
