import express, { Router } from "express";
import { imageFile } from "@/middlewares/multer";

const destroyRouter: Router = express.Router();

export default destroyRouter;
