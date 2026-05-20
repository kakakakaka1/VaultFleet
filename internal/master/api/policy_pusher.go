package api

import (
	"context"
	"log"

	"vaultfleet/internal/master/commands"
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
	DB       *db.Database
	Hub      PolicyPusherHub
	Lookup   PolicyLookupFunc
	Commands *commands.Service
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
	if p.Commands == nil {
		if err := p.Hub.Send(agentID, *msg); err != nil {
			log.Printf("push policy to agent %s failed: %v", agentID, err)
		}
		return
	}

	policyID, storageID := p.currentPolicyRefs(agentID)
	if _, err := p.Commands.CreateCommand(context.Background(), commands.CreateCommandInput{
		AgentID:   agentID,
		Type:      protocol.TypePolicyPush,
		Message:   *msg,
		PolicyID:  policyID,
		StorageID: storageID,
	}); err != nil {
		log.Printf("create policy command for agent %s failed: %v", agentID, err)
		return
	}
	if err := p.Commands.DispatchPendingForAgent(context.Background(), agentID, 10); err != nil {
		log.Printf("dispatch policy command for agent %s failed: %v", agentID, err)
	}
}

func (p *PolicyChangedPusher) currentPolicyRefs(agentID string) (string, string) {
	if p == nil || p.DB == nil || p.DB.DB == nil {
		return "", ""
	}
	var policy db.BackupPolicy
	if err := p.DB.DB.
		Where("agent_id = ? AND synced = ?", agentID, false).
		Order("updated_at DESC").
		First(&policy).Error; err != nil {
		return "", ""
	}
	return policy.ID, policy.StorageID
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
