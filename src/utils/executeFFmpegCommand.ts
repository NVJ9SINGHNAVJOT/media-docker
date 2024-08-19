import { logger } from "@/logger/logger";
import { exec } from "child_process";

export default function executeFFmpegCommand(command: string): Promise<boolean> {
  return new Promise((resolve, reject) => {
    exec(command, (error) => {
      if (error) {
        logger.error("error while converting video", { error: error.message, command: command });
        reject(false);
      } else {
        resolve(true);
      }
    });
  });
}
