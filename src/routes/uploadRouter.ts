import { uploadVideo, uploadVideoResolutions } from "@/controllers/upload";
import { videoFile } from "@/middlewares/multer";
import express, { Router } from "express";

const uploadRouter: Router = express.Router();

uploadRouter.post("/video", videoFile, uploadVideo);
uploadRouter.post("/videoResolutions", videoFile, uploadVideoResolutions);

export default uploadRouter;
