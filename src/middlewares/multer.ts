import multer, { FileFilterCallback } from "multer";
import fs from "fs";
import path from "path";
import { v4 as uuidv4 } from "uuid";
import { Request } from "express";

// create uploadStorage directory if not exist
fs.mkdirSync("uploadStorage", { recursive: true });

// only valid files will be allowed by multer
const validFiles = {
  image: ["image/jpeg", "image/jpg", "image/png"],
  video: ["video/mp4", "video/webm", "video/ogg"],
  audio: ["audio/mp3", "audio/mpeg", "audio/wav"],
};

// storage configuration for multer
const storage = multer.diskStorage({
  destination: (_req, _file, cb) => {
    cb(null, "uploadStorage");
  },
  filename: function (_req, file, cb) {
    cb(null, file.fieldname + "-" + uuidv4() + path.extname(file.originalname));
  },
});

// filter for file or files
const filterVideo = (_req: Request, file: Express.Multer.File, cb: FileFilterCallback) => {
  if (validFiles.video.includes(file.mimetype)) {
    cb(null, true); // accept file
  } else {
    cb(null, false); // reject file
    return cb(new Error("Only .mp4, .webm and .ogg format allowed!"));
  }
};
const filterImage = (_req: Request, file: Express.Multer.File, cb: FileFilterCallback) => {
  if (validFiles.image.includes(file.mimetype)) {
    cb(null, true); // accept file
  } else {
    cb(null, false); // reject file
    return cb(new Error("Only .jpeg, .jpg and .png format allowed!"));
  }
};
const filterAudio = (_req: Request, file: Express.Multer.File, cb: FileFilterCallback) => {
  if (validFiles.audio.includes(file.mimetype)) {
    cb(null, true); // accept file
  } else {
    cb(null, false); // reject file
    return cb(new Error("Only .mp3, .mpeg, and .wav format allowed!"));
  }
};

// limits for multer
const singleFileLimit = {
  fileSize: 1024 * 1024 * 1000, // individual file size limit to 1 GB
  files: 1, // maximum number of files to 1
};

// multer uploads
const multerUploadImage = multer({
  storage: storage,
  limits: singleFileLimit,
  fileFilter: filterImage,
});
const multerUploadVideo = multer({
  storage: storage,
  limits: singleFileLimit,
  fileFilter: filterVideo,
});
const multerUploadAudio = multer({
  storage: storage,
  limits: singleFileLimit,
  fileFilter: filterAudio,
});

// multer configuration for file or files in request
const imageFile = multerUploadImage.single("imageFile");
const videoFile = multerUploadVideo.single("videoFile");
const audioFile = multerUploadAudio.single("audioFile");

export { imageFile, videoFile, audioFile };
