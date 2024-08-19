// initialization environment for server
import dotenv from "dotenv";
dotenv.config();

// setup server config
import fs from "fs";
import { logger, loggerConfig } from "@/logger/logger";
import app from "@/app/app";

function main() {
  // setup logger
  loggerConfig(`${process.env["ENVIRONMENT"]}`);

  /*
    create media_docker_files folder if not exists.
    media_docker_files folder will store all media files.
  */
  if (!fs.existsSync("media_docker_files")) {
    fs.mkdirSync("media_docker_files", { recursive: true });
  }

  // get port number
  const PORT = parseInt(`${process.env["PORT"]}`) || 7000;

  // setup server
  const httpServer = app;

  httpServer.listen(PORT, () => {
    logger.info("server running...");
  });
}

main();
