import { logger } from "@/logger/logger";
import fs from "fs";

export function deleteFile(file: Express.Multer.File) {
  if (fs.existsSync(file.path)) {
    fs.unlink(file.path, (unlinkError) => {
      if (unlinkError) {
        logger.error("error deleting file from uploadStorage", { error: unlinkError });
      }
    });
  }
}

export function deleteFiles(files: Express.Multer.File[] | { [fieldname: string]: Express.Multer.File[] }) {
  if (Array.isArray(files)) {
    // If files is already an array, directly process it
    files.forEach((file) => {
      deleteFile(file);
    });
  } else {
    // Otherwise, extract the array of files from the dictionary
    const fileArray = Object.values(files).flat();
    fileArray.forEach((file) => {
      deleteFile(file);
    });
  }
}
