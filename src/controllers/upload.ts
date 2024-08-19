import { errRes } from "@/utils/error";
import { Request, Response } from "express";
import { v4 as uuidv4 } from "uuid";
import fs from "fs";
import { deleteFile } from "@/utils/deleteFile";
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
    deleteFile(req.file as Express.Multer.File);
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
