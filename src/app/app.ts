import express, { Request, Response } from "express";
import cookieParser from "cookie-parser";
import cors from "cors";
import uploadRouter from "@/routes/uploadRouter";
import logging from "@/middlewares/logging";
import destroyRouter from "@/routes/destroyRouter";
import serverKey from "@/middlewares/serverKey";

const app = express();

app.use(
  cors({
    origin: "*",
    credentials: true,
    methods: ["POST", "DELETE"],
  })
);
app.use(express.urlencoded({ extended: true }));
app.use(express.json());
app.use(cookieParser());

// logging details
app.use(logging);

// server only accessible with serverKey
app.use(serverKey);

// routes
app.use("/api/v1/uploads", uploadRouter);
app.use("/api/v1/destroy", destroyRouter);

app.get("/", (_req: Request, res: Response) => {
  res.json({
    success: true,
    message: "server is up and running.",
  });
});

export default app;
