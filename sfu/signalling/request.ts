import fetch from 'node-fetch'


const baseURL = 'http://localhost:5000'


const publish = async (streamId: string, sdp: string, data?:any) => {

    let res = await fetch(baseURL + '/api/publish', {
        method: 'post',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            sdp: sdp,
            streamId: streamId,
            data: data 
        })
    })

    let ret = await res.json()
    console.dir(ret)
    return ret.d
}



const unpublish = async (streamId: string) => {

    // streamId
    let res = await fetch(baseURL + '/api/unpublish', {
        method: 'post',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            streamId: streamId
        })
    })

    let ret = await res.json()
    console.dir(ret)
    return ret.d
}


const play = async (streamId: string, sdp: string) => {

    // const sdp = req.body.sdp
    // const streamId = req.body.streamId
    
    let res = await fetch(baseURL + '/api/play', {
        method: 'post',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            streamId: streamId,
            sdp: sdp
        })
    })

    let ret = await res.json()
    console.dir(ret)
    return ret.d
    
}


const unplay = async (streamId: string, outgoingId:string) => {

    // const streamId = req.body.streamId
    // const outgoingId = req.body.outgoingId 

    let res = await fetch(baseURL + '/api/unplay', {
        method: 'post',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            streamId: streamId,
            outgoingId: outgoingId
        })
    })

    let ret = await res.json()
    console.dir(ret)
    return ret.d
}


export {
    publish,
    unpublish,
    play,
    unplay
}