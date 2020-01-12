package rtmp

import (
	"fmt"

	"github.com/notedit/gst"
)

type RtmpPusher struct {
	pipeline *gst.Pipeline
	appsrc   *gst.Element
}

func NewRtmpPusher(rtmpUrl string) (*RtmpPusher, error) {

	err := gst.CheckPlugins([]string{"flv", "rtmp"})

	if err != nil {
		return nil, err
	}

	pipelineStr := "appsrc is-live=true do-timestamp=true name=src ! h264parse ! video/x-h264,stream-format=(string)avc ! flvmux ! rtmpsink sync=false location='%s live=1'"

	pipelineStr = fmt.Sprintf(pipelineStr, rtmpUrl)

	pipeline, err := gst.ParseLaunch(pipelineStr)

	if err != nil {
		return nil, err
	}

	pusher := &RtmpPusher{
		pipeline: pipeline,
		appsrc:   pipeline.GetByName("src"),
	}

	return pusher, nil
}

func (p *RtmpPusher) Start() {

	p.pipeline.SetState(gst.StatePlaying)
}

func (p *RtmpPusher) Stop() {

	p.pipeline.SetState(gst.StateNull)
}

func (p *RtmpPusher) Push(buffer []byte) {

	err := p.appsrc.PushBuffer(buffer)

	if err != nil {
		fmt.Println("push buffer error", err)
	}
}
