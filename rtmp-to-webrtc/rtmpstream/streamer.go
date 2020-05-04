package rtmpstream

import (
	"bytes"
	"fmt"

	"github.com/notedit/gst"
	"github.com/notedit/media-server-go"
	"github.com/notedit/rtmp-lib/aac"
	"github.com/notedit/rtmp-lib/av"
	"github.com/notedit/rtmp-lib/h264"
	"github.com/notedit/sdp"
)

// decodebin  !  x264enc aud=false bframes=0 speed-preset=veryfast key-int-max=15 ! video/x-h264,stream-format=byte-stream,profile=baseline !
// gst-launch-1.0 filesrc location=example.aac ! decodebin ! audioconvert ! audioresample ! audio/x-raw,rate=48000 ! opusenc ! decodebin  ! autoaudiosink
//var audio2rtp = "appsrc is-live=true name=appsrc ! decodebin ! audioconvert ! audioresample ! audio/x-raw,rate=48000 !  opusenc ! opusdec ! autoaudiosink"
// do-timestamp=true is-live=true

var audio2rtp = "appsrc do-timestamp=true is-live=true  name=appsrc ! decodebin ! audioconvert ! audioresample ! opusenc ! rtpopuspay timestamp-offset=0 pt=%d ! udpsink host=127.0.0.1 port=%d"
var video2rtp = "appsrc do-timestamp=true is-live=true  name=appsrc ! h264parse ! rtph264pay timestamp-offset=0 config-interval=-1 pt=%d ! udpsink host=127.0.0.1 port=%d"



type RtmpStream struct {
	streams        []av.CodecData
	videoCodecData h264.CodecData
	audioCodecData aac.CodecData
	audioPipeline  *gst.Pipeline
	videoPipeline  *gst.Pipeline
	audiosrc       *gst.Element
	videosrc       *gst.Element

	videoout <-chan []byte
	audioout <-chan []byte

	adtsheader []byte

	spspps bool

	videoSession    *mediaserver.StreamerSession
	audioSession    *mediaserver.StreamerSession
	audioCapability *sdp.Capability
	videoCapability *sdp.Capability
}

// NewMediaTransform  create media transform
func NewRtmpStreamer(audio *sdp.Capability, video *sdp.Capability) *RtmpStream {
	streamer := &RtmpStream{}
	streamer.audioCapability = audio
	streamer.videoCapability = video
	return streamer
}

// WriteHeader got sps and pps
func (self *RtmpStream) WriteHeader(streams []av.CodecData) error {

	self.streams = streams

	for _, stream := range streams {
		if stream.Type() == av.H264 {
			h264Codec := stream.(h264.CodecData)
			self.videoCodecData = h264Codec

			videoMediaInfo := sdp.MediaInfoCreate("video", self.videoCapability)

			self.videoSession = mediaserver.NewStreamerSession(videoMediaInfo)

			video2rtpstr := fmt.Sprintf(video2rtp, videoMediaInfo.GetCodec("h264").GetType(), self.videoSession.GetLocalPort())

			videoPipeline, err := gst.ParseLaunch(video2rtpstr)
			if err != nil {
				panic(err)
			}

			self.videoPipeline = videoPipeline
			self.videosrc = videoPipeline.GetByName("appsrc")

			videoPipeline.SetState(gst.StatePlaying)

		}
		if stream.Type() == av.AAC {
			aacCodec := stream.(aac.CodecData)
			self.audioCodecData = aacCodec

			audioMediaInfo := sdp.MediaInfoCreate("audio", self.audioCapability)

			self.audioSession = mediaserver.NewStreamerSession(audioMediaInfo)

			audio2rtpstr := fmt.Sprintf(audio2rtp, audioMediaInfo.GetCodec("opus").GetType(), self.audioSession.GetLocalPort())
			//audio2rtpstr = audio2rtp
			audioPipeline, err := gst.ParseLaunch(audio2rtpstr)
			if err != nil {
				panic(err)
			}

			self.adtsheader = make([]byte, 7)

			self.audioPipeline = audioPipeline
			self.audiosrc = audioPipeline.GetByName("appsrc")
			audioPipeline.SetState(gst.StatePlaying)

		}
	}

	return nil
}

// WritePacket
func (self *RtmpStream) WritePacket(packet av.Packet) error {

	stream := self.streams[packet.Idx]

	if stream.Type() == av.H264 {

		if !self.spspps {
			var b bytes.Buffer
			b.Write([]byte{0, 0, 0, 1})
			b.Write(self.videoCodecData.SPS())
			b.Write([]byte{0, 0, 0, 1})
			b.Write(self.videoCodecData.PPS())
			self.videosrc.PushBuffer(b.Bytes())
			self.spspps = true
		}

		pktnalus, _ := h264.SplitNALUs(packet.Data)
		for _, nalu := range pktnalus {
			var b bytes.Buffer
			b.Write([]byte{0, 0, 0, 1})
			b.Write(nalu)
			self.videosrc.PushBuffer(b.Bytes())
		}

	}

	if stream.Type() == av.AAC {

		adtsbuffer := []byte{}
		aac.FillADTSHeader(self.adtsheader, self.audioCodecData.Config, 1024, len(packet.Data))
		adtsbuffer = append(adtsbuffer, self.adtsheader...)
		adtsbuffer = append(adtsbuffer, packet.Data...)

		self.audiosrc.PushBuffer(adtsbuffer)
	}

	return nil
}

// WriteTrailer
func (self *RtmpStream) WriteTrailer() error {
	return nil
}

func (self *RtmpStream) HasVideo() bool {
	if self.videoPipeline != nil {
		return true
	}
	return false
}

func (self *RtmpStream) HasAudio() bool {
	if self.videoPipeline != nil {
		return true
	}
	return false
}

func (self *RtmpStream) GetVideoTrack() *mediaserver.IncomingStreamTrack {

	if self.videoSession != nil {
		return self.videoSession.GetIncomingStreamTrack()
	}
	return nil
}

func (self *RtmpStream) GetAuidoTrack() *mediaserver.IncomingStreamTrack {

	if self.audioSession != nil {
		return self.audioSession.GetIncomingStreamTrack()
	}
	return nil
}
