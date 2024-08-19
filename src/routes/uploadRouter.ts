import { uploadVideo } from "@/controllers/upload";
import { videoFile } from "@/middlewares/multer";
import express, { Router } from "express";

const uploadRouter: Router = express.Router();

uploadRouter.post("/video", videoFile, uploadVideo);

export default uploadRouter;
