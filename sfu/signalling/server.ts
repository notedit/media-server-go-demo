import { EventEmitter } from 'events'
import * as bodyParser from 'body-parser'
import express from 'express'
import { Response, Request } from 'express'
import socketio from 'socket.io'
import * as path from 'path'
import * as http from 'http'


const app = express()

const socketServer = socketio({
    pingInterval: 20000,
    pingTimeout: 5000,
    transports: ['websocket']
})

const sessions: Map<string, Map<string,any>> = new Map()
const streams: Map<string, Array<string>> = new Map()


socketServer.on('connection', async (socket: SocketIO.Socket) => {

    let room = socket.handshake.query.room 
    let user = socket.handshake.query.user

    let userMap:Map<string,any> = new Map()

    userMap.set(user, {
        user:user,
        socket: socket,
        streams: {}
    })

    sessions.set(room, userMap)

    socket.join(room)

    socket.on('disconnected', async () => {

    })
})


app.post('/join', async (req: Request, res: Response) => {

    
})


app.post('/publish', async (req: Request, res:Response) => {

})


app.post('/unpublish', async (req: Request, res:Response) => {


})


app.post('/play', async (req: Request, res:Response) => {

})


app.post('/unplay', async (req: Request, res:Response) => {

})


const httpServer = app.listen(3000,'0.0.0.0', ()=> {
    console.log("s")
})
socketServer.attach(httpServer)




