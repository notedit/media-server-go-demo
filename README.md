# media-server-go-demo


media server go demo  for  https://github.com/notedit/media-server-go



## Build 
#### Ubuntu 18.04.2 LTS
```shell
mkdir wk_webrtc && cd wk_webrtc
git clone --recurse-submodules https://github.com/notedit/media-server-go-native.git
git clone https://github.com/notedit/media-server-go.git
git clone https://github.com/notedit/media-server-go-demo.git

# build media-server-go-native
cd media-server-go-native && make -j 4

# build media-server-go
cd ../media-server-go && go build 

# build media-server-go-demo
cd ../media-server-go-demo

# broadcast 
go build 

# recording
go build 

# rtmp-to-webrtc 
sudo apt-get install libgstreamer1.0-0 gstreamer1.0-plugins-base gstreamer1.0-libav gstreamer1.0-plugins-bad libgstreamer-plugins-bad1.0-dev 
go build

# rtp-streamer
the same with rtmp-to-webrtc 

# server-to-server 
go build 

# sfu
go build 

# video-mixer
go build 

# webrtc-to-hls
go build

# webrtc-to-rtmp
go build 

```

## Examples

- [WebRTC-Broadcast](https://github.com/notedit/media-server-go-demo/tree/master/broadcast): WebRTC publish and play 
- [WebRTC-Record](https://github.com/notedit/media-server-go-demo/tree/master/recording): WebRTC record
- [RTMP-To-WebRTC](https://github.com/notedit/media-server-go-demo/tree/master/rtmp-to-webrtc): Rtmp to webrtc
- [Server-To-Server](https://github.com/notedit/media-server-go-demo/tree/master/server-to-server): WebRTC server relay
- [WebRTC-To-RTMP](https://github.com/notedit/media-server-go-demo/tree/master/webrtc-to-rtmp): WebRTC to rtmp
- [WebRTC-To-HLS](https://github.com/notedit/media-server-go-demo/tree/master/webrtc-to-hls): WebRTC to hls