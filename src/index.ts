// initialization environment for server
import dotenv from "dotenv";
dotenv.config();

// setup server config
import { logger, loggerConfig } from "@/logger/logger";
import app from "@/app/app";

function main() {
  // setup logger
  loggerConfig(`${process.env["ENVIRONMENT"]}`);

  // get port number
  const PORT = parseInt(`${process.env["PORT"]}`) || 7000;

  // setup server
  const httpServer = app;

  httpServer.listen(PORT, () => {
    logger.info("server running...");
  });
}

main();
