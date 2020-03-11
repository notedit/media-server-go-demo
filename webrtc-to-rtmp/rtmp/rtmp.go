package rtmp

import (
	"fmt"

	"github.com/notedit/gst"
)

const pipelinestring = "appsrc is-live=true do-timestamp=true name=videosrc ! h264parse ! video/x-h264,stream-format=(string)avc ! muxer.   appsrc is-live=true do-timestamp=true name=audiosrc ! opusparse ! opusdec ! audioconvert ! faac ! muxer.  flvmux name=muxer ! rtmpsink sync=false location='%s live=1'"


type RtmpPusher struct {
	pipeline *gst.Pipeline
	videosrc *gst.Element
	audiosrc *gst.Element
}


func NewRtmpPusher(rtmpUrl string) (*RtmpPusher, error) {

	err := gst.CheckPlugins([]string{"flv", "rtmp","faac"})

	if err != nil {
		return nil, err
	}

	pipelineStr := fmt.Sprintf(pipelinestring, rtmpUrl)

	pipeline, err := gst.ParseLaunch(pipelineStr)

	if err != nil {
		return nil, err
	}

	videosrc := pipeline.GetByName("videosrc")
	audiosrc := pipeline.GetByName("audiosrc")

	pusher := &RtmpPusher{
		pipeline: pipeline,
		videosrc: videosrc,
		audiosrc: audiosrc,
	}

	return pusher, nil
}

func (p *RtmpPusher) Start() {

	p.pipeline.SetState(gst.StatePlaying)
}

func (p *RtmpPusher) Stop() {

	p.pipeline.SetState(gst.StateNull)
}

func (p *RtmpPusher) Push(buffer []byte, audio bool) {

	var err error
	if audio {
		err = p.audiosrc.PushBuffer(buffer)
	}  else {
		err = p.videosrc.PushBuffer(buffer)
	}

	if err != nil {
		fmt.Println("push buffer error", err)
	}
}
