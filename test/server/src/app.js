import "dotenv/config"
import { model, Schema } from "mongoose"
import express from "express"
import cors from "cors"
import { connect } from "mongoose"

const mongoDBConfig = async (retries = 5) => {
     const uri = process.env.MONGODB_URI
     try {
          await connect(uri, { serverSelectionTimeoutMS: 5000 });
          console.log('✅ Success connected with MongoDB of TEST')
     } catch (err) {
          if (retries > 0) {
               console.warn(`MongoDB connection failed. Retrying... (${retries} left)`);
               await new Promise(r => setTimeout(r, 3000));
               return mongoDBConfig(retries - 1);
          }
          console.error('MongoDB connection error:', err);
          process.exit(1);
     }
}

const usersSchema = new Schema({
     fullname: {
          type: String,
          required: true,
          trim: true,
          index: true,
     },
     username: {
          type: String,
          required: true,
          trim: true,
          index: true,
     },
     email: {
          type: String,
          unique: true,
          required: true,
          trim: true,
          lowercase: true,
     },
     code: {
          type: String,
          default: '+20',
     },
     phone: {
          type: String,
          default: '',
          index: true,
     },
     status: {
          type: String,
          enum: ['Confirmed', 'Deleted', 'Pending', 'Blocked', 'Ban'],
          default: 'Pending',
     },
}, {
     timestamps: true
})

export const userModel = model("User", usersSchema)

const app = express()
let totalRequests = 0

export function countRequests(req, res, next) {
     totalRequests += 1
     req.requestNumber = totalRequests
     console.log(`Request #${req.requestNumber}: ${req.method} ${req.originalUrl}`)
     next()
}

app.use(countRequests)

export const allowedOrigins = [
     process.env.SITE_URL_TEST,
     process.env.SITE_URL_LIVE,
]
const corsOptions = {
     origin: (origin, callback) => {
          if (!origin || allowedOrigins.includes(origin)) {
               callback(null, true)
          } else {
               callback(new Error('Not allowed by CORS'))
          }
     },
     credentials: true,
     optionsSuccessStatus: 200,
}

app.use(cors(corsOptions))
app.use(
     express.json({
          limit: '2mb',
     }),
)
app.use(
     express.urlencoded({
          extended: true,
     }),
)

app.get("/api/health", async (req, res) => {
     try {
          res.status(200).json({ code: 200, status: "OK", data: "GG" })
     }
     catch (err) {
          console.error(err)
          res.status(500).json({ code: 500, status: "Error" })
     }
})


app.get("/api/user/all", async (req, res) => {
     try {
          const users = await userModel.find({ status: "Confirmed" }, { _id: 1 }).limit(10).sort({ createdAt: -1 })
          res.status(200).json({ code: 200, status: "OK", data: users })
     }
     catch (err) {
          console.error(err)
          res.status(500).json({ code: 500, status: "Error" })
     }
})

const PORT = Number(process.env.PORT) || 9000

async function startServer() {
     try {
          await Promise.all([
               mongoDBConfig(),
          ]).then(() => {
               app.listen(PORT, () => {
                    console.log('🌐 Server is running on:', `${process.env.API_LINK}`)
               })
          })
     } catch (error) {
          console.error(error)
          process.exit(1)
     }
}

if (process.env.NODE_ENV !== 'test') {
     process.on('SIGTERM', () => void gracefulShutdown('SIGTERM'))
     process.on('SIGINT', () => void gracefulShutdown('SIGINT'))
     startServer()
}

export { app }
