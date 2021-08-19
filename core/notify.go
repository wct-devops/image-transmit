package core

import (
	"github.com/blinkbean/dingtalk"
	log "github.com/cihub/seelog"
)

type DingTalkAccess struct {
	Token string
	Secret string
}

type Notify interface {
	Send(text string)
}

type DingTalkWapper struct {
	clients []*dingtalk.DingTalk
}

func NewDingTalkWapper(dtas []DingTalkAccess) Notify{
	var clients []*dingtalk.DingTalk
	for _, dta := range dtas {
		cli := dingtalk.InitDingTalkWithSecret(dta.Token, dta.Secret)
		clients = append(clients, cli)
	}

	return &DingTalkWapper{
		clients : clients,
	}
}

func (d *DingTalkWapper) Send(text string) {
	log.Info(I18n.Sprintf("Send DingTalk message, text: %v", text))
	for _, cli := range d.clients {
		err := cli.SendMarkDownMessage(I18n.Sprintf("Image Transmit Notify"), text)
		if err != nil {
			log.Error(err)
		}
	}
}
