import { EventEmitter } from 'events'
import bodyParser from 'body-parser'
import express from 'express'
import { Response, Request } from 'express'
import socketio from 'socket.io'

import * as request from './request'


const app = express()
app.use(bodyParser.json())
app.use(bodyParser.urlencoded({extended: true}))


const socketServer = socketio({
    pingInterval: 20000,
    pingTimeout: 5000,
    transports: ['websocket']
})

const rooms: Map<string, Map<string,Peer>> = new Map()

interface Peer {
    roomId:string
    peerId:string
    streams:Map<string,any>
}

socketServer.on('connection', async (socket: SocketIO.Socket) => {

    const roomId = socket.handshake.query.roomId
    const peerId = socket.handshake.query.peerId

    if (!rooms.get(roomId)) {
        rooms.set(roomId,new Map())
    }

    socket.on('join', async (data:any, ack:Function) => {

        const room = rooms.get(roomId)

        const peerData = {
            roomId:roomId,
            peerId:peerId,
            streams: new Map()
        }

        room.set(peerId, peerData)
        socket.join(roomId)

        let info = {
            roomId: roomId,
            streams: []
        }
    
        for (let peer of room.values()) {
            for (let stream of peer.streams.keys()) {
                info.streams.push({
                    publisherId: stream,
                    data: peer.streams.get(stream)
                })
            }
        }

        ack(info)
    })

    socket.on('publish', async (data:any, ack:Function) => {

        const sdp = data.sdp
        const publisherId = data.stream.publisherId
        const streamData = data.stream.data
    
        const ret = await request.publish(publisherId,sdp,streamData)
    
        const answer = ret.sdp
        const streamId = ret.streamId

        const peer = getPeer(roomId,peerId)

        peer.streams.set(publisherId, streamData)

        ack({sdp:answer})

        socket.to(roomId).emit('streampublished', {
            stream: {
                publisherId: streamId,
                data: streamData
            }
        })
    })

    socket.on('unpublish', async (data:any, ack:Function) => {

        const publisherId = data.stream.publisherId

        const peer = getPeer(roomId, peerId)

        await request.unpublish(publisherId)

        peer.streams.delete(publisherId)

        ack({})

        socket.emit('streamunpublished', {
            stream: {
                publisherId: publisherId,
                data: {}
            }
        })

    })

    
    socket.on('subscribe', async (data:any, ack:Function) => {

        const sdp = data.sdp
        const publisherId = data.stream.publisherId

        const ret = await request.play(publisherId, sdp) // sdp  outgoingId

        ack({
            sdp: ret.sdp,
            stream: {
                subscriberId: ret.outgoingId,
                data: getPublisher(roomId,publisherId)
            }
        })

    })

    socket.on('unsubscribe', async (data:any, ack:Function) => {

        const publisherId = data.stream.publisherId
        const subscriberId = data.stream.subscriberId

        await request.unplay(publisherId, subscriberId)

        ack({})
    })


    socket.on('disconnected', async () => {

        const room = rooms.get(roomId)
        const peer = getPeer(roomId,peerId)

        for (let stream of peer.streams.keys()) {
            socket.to(roomId).emit('streamunpublished', {
                stream: {
                    publisherId: stream,
                    data: {}
                }
            })
        }

        for (let stream of peer.streams.keys()) {
            await request.unpublish(stream)
        }

        room.delete(peerId)

        socket.leaveAll()
    })
})


const getPeer = (roomId:string,peerId:string):Peer => {

    const room = rooms.get(roomId)
    if (room) {
        return room.get(peerId)
    }
    return null
}

const getPublisher = (roomId:string, publisherId:string) => {

    const room = rooms.get(roomId)

    if (!room) {
        return {}
    } 

    for (let peer of room.values()) {
        for (let stream of peer.streams.keys()) {
            if (stream === publisherId) {
                return peer.streams.get(stream)
            }
        }
    }

    return {}
}


const httpServer = app.listen(3000,'0.0.0.0', ()=> {
    console.log("s")
})
socketServer.attach(httpServer)




