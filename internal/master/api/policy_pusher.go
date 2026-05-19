package api

import (
	"log"

	"vaultfleet/internal/master/db"
	"vaultfleet/internal/master/events"
	"vaultfleet/pkg/protocol"
)

type PolicyPusherHub interface {
	IsOnline(agentID string) bool
	Send(agentID string, msg interface{}) error
}

type PolicyLookupFunc func(agentID string) (*protocol.Message, bool)

type PolicyChangedPusher struct {
	DB     *db.Database
	Hub    PolicyPusherHub
	Lookup PolicyLookupFunc
}

func NewPolicyChangedPusher(database *db.Database, hub PolicyPusherHub, lookup PolicyLookupFunc) *PolicyChangedPusher {
	return &PolicyChangedPusher{DB: database, Hub: hub, Lookup: lookup}
}

func (p *PolicyChangedPusher) Handle(event events.Event) {
	if p == nil || p.Hub == nil || p.Lookup == nil {
		return
	}
	if action := eventAction(event.Payload); action == "ack" {
		return
	}
	agentID := eventAgentID(event.Payload)
	if agentID == "" || !p.Hub.IsOnline(agentID) {
		return
	}
	msg, ok := p.Lookup(agentID)
	if !ok || msg == nil {
		return
	}
	if err := p.Hub.Send(agentID, *msg); err != nil {
		log.Printf("push policy to agent %s failed: %v", agentID, err)
	}
}

func eventAction(payload any) string {
	switch value := payload.(type) {
	case map[string]any:
		if action, ok := value["action"].(string); ok {
			return action
		}
	}
	return ""
}
