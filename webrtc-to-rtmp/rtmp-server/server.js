const { NodeMediaServer } = require('node-media-server');


const config = {
  rtmp: {
    port: 1935,
    chunk_size: 1024,
    gop_cache: true,
    ping: 60,
    ping_timeout: 30
  }
};

var nms = new NodeMediaServer(config)
nms.run();



