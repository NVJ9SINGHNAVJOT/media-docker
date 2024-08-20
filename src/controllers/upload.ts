import { errRes } from "@/utils/error";
import { Request, Response } from "express";
import { v4 as uuidv4 } from "uuid";
import fs from "fs";
import { deleteFile } from "@/utils/delete";
import executeFFmpegCommand from "@/utils/executeFFmpegCommand";

export const uploadVideo = async (req: Request, res: Response): Promise<Response> => {
  try {
    if (!req.file) {
      return errRes(res, 400, "video file not present");
    }

    const id = uuidv4();
    const videoPath = req.file.path;
    const outputPath = `media_docker_files/videos/${id}`;
    const hlsPath = `${outputPath}/index.m3u8`;

    fs.mkdirSync(outputPath, { recursive: true });

    // ffmpeg
    const ffmpegCommand = `ffmpeg -i ${videoPath} -codec:v libx264 -codec:a aac -hls_time 10 -hls_playlist_type vod -hls_segment_filename "${outputPath}/segment%03d.ts" -start_number 0 ${hlsPath}`;

    // executing command
    const executed = await executeFFmpegCommand(ffmpegCommand);
    deleteFile(req.file);
    if (executed) {
      return res.status(200).json({
        success: true,
        message: "video converted to HLS format",
        videoUrl: `http://localhost:7000/media_docker_files/videos/${id}/index.m3u8`,
      });
    }

    return errRes(res, 500, "error while executing ffmpeg conversion");
  } catch (error) {
    if (req.file) {
      deleteFile(req.file);
    }
    return errRes(res, 500, "error while uploading video", error instanceof Error ? error.message : "unknown");
  }
};

export const uploadVideoResolutions = async (req: Request, res: Response): Promise<Response> => {
  try {
    if (!req.file) {
      return errRes(res, 400, "video file not present");
    }

    const id = uuidv4();
    const videoPath = req.file.path;

    const outputPath360 = `media_docker_files/videos/${id}/360`;
    const outputPath480 = `media_docker_files/videos/${id}/480`;
    const outputPath720 = `media_docker_files/videos/${id}/720`;
    const outputPath1080 = `media_docker_files/videos/${id}/1080`;
    const hlsPath360 = `${outputPath360}/index.m3u8`;
    const hlsPath480 = `${outputPath480}/index.m3u8`;
    const hlsPath720 = `${outputPath720}/index.m3u8`;
    const hlsPath1080 = `${outputPath1080}/index.m3u8`;

    fs.mkdirSync(outputPath360, { recursive: true });
    fs.mkdirSync(outputPath480, { recursive: true });
    fs.mkdirSync(outputPath720, { recursive: true });
    fs.mkdirSync(outputPath1080, { recursive: true });

    // ffmpeg
    const commands = [
      `ffmpeg -i ${videoPath} -codec:v libx264 -codec:a aac -vf "scale=640:360" -hls_time 10 -hls_playlist_type vod -hls_segment_filename "${outputPath360}/segment%03d.ts" -start_number 0 ${hlsPath360}`,
      `ffmpeg -i ${videoPath} -codec:v libx264 -codec:a aac -vf "scale=854:480" -hls_time 10 -hls_playlist_type vod -hls_segment_filename "${outputPath480}/segment%03d.ts" -start_number 0 ${hlsPath480}`,
      `ffmpeg -i ${videoPath} -codec:v libx264 -codec:a aac -vf "scale=1280:720" -hls_time 10 -hls_playlist_type vod -hls_segment_filename "${outputPath720}/segment%03d.ts" -start_number 0 ${hlsPath720}`,
      `ffmpeg -i ${videoPath} -codec:v libx264 -codec:a aac -vf "scale=1920:1080" -hls_time 10 -hls_playlist_type vod -hls_segment_filename "${outputPath1080}/segment%03d.ts" -start_number 0 ${hlsPath1080}`,
    ];

    // executing command
    const executed = commands.map((command) => executeFFmpegCommand(command));

    const result = await Promise.all(executed);

    deleteFile(req.file);
    if (!result.includes(false)) {
      return res.status(200).json({
        success: true,
        message: "video converted to HLS format",
        videoUrls: {
          360: `http://localhost:7000/media_docker_files/videos/${id}/360/index.m3u8`,
          480: `http://localhost:7000/media_docker_files/videos/${id}/420/index.m3u8`,
          720: `http://localhost:7000/media_docker_files/videos/${id}/720/index.m3u8`,
          1080: `http://localhost:7000/media_docker_files/videos/${id}/1080/index.m3u8`,
        },
      });
    }

    return errRes(res, 500, "error while executing ffmpeg conversions");
  } catch (error) {
    if (req.file) {
      deleteFile(req.file);
    }
    return errRes(
      res,
      500,
      "error while uploading video resolutions",
      error instanceof Error ? error.message : "unknown"
    );
  }
};
