package autoassigner

import (
	"io"
	"testing"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	tmodels "github.com/abhinavxd/libredesk/internal/team/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

type mockTeamStore struct {
	teams   []tmodels.Team
	members map[int][]tmodels.TeamMember
}

func (m *mockTeamStore) GetAll() ([]tmodels.Team, error) {
	return m.teams, nil
}

func (m *mockTeamStore) GetMembers(teamID int) ([]tmodels.TeamMember, error) {
	return m.members[teamID], nil
}

type mockConversationStore struct {
	unassigned   []models.Conversation
	activeCounts map[int]int
	claimed      map[string]int
}

func (m *mockConversationStore) GetUnassignedConversations() ([]models.Conversation, error) {
	return m.unassigned, nil
}

func (m *mockConversationStore) ClaimUnassignedConversation(uuid string, userID, expectedTeamID int, user umodels.User) error {
	m.claimed[uuid] = userID
	m.activeCounts[userID]++
	return nil
}

func (m *mockConversationStore) ActiveUserConversationsCount(userID int) (int, error) {
	return m.activeCounts[userID], nil
}

func TestAutoassignerMaxOpenConversations(t *testing.T) {
	lo := logf.New(logf.Opts{Writer: io.Discard})

	teamStore := &mockTeamStore{
		teams: []tmodels.Team{
			{
				ID:                         1,
				ConversationAssignmentType: AssignmentTypeRoundRobin,
			},
		},
		members: map[int][]tmodels.TeamMember{
			1: {
				{ID: 101, AvailabilityStatus: umodels.Online, TeamID: 1, MaxOpenConversations: 2}, // 上限 2
				{ID: 102, AvailabilityStatus: umodels.Online, TeamID: 1, MaxOpenConversations: 5}, // 上限 5
			},
		},
	}

	convoStore := &mockConversationStore{
		unassigned: []models.Conversation{
			{UUID: "c1", AssignedTeamID: null.IntFrom(1)},
		},
		activeCounts: map[int]int{
			101: 2, // 101 已达到上限 2
			102: 1, // 102 未达到上限 5
		},
		claimed: make(map[string]int),
	}

	engine, err := New(teamStore, convoStore, umodels.User{ID: 1}, &lo)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	if err := engine.reloadBalancer(); err != nil {
		t.Fatalf("failed to reload balancer: %v", err)
	}

	if err := engine.assignConversations(); err != nil {
		t.Fatalf("failed to assign conversations: %v", err)
	}

	// 101 满员，应分配给 102
	assignedUser, ok := convoStore.claimed["c1"]
	if !ok {
		t.Fatalf("expected conversation c1 to be claimed, but was not")
	}
	if assignedUser != 102 {
		t.Errorf("expected conversation c1 to be assigned to user 102, got %d", assignedUser)
	}
}

func TestAutoassignerAllAgentsCapped(t *testing.T) {
	lo := logf.New(logf.Opts{Writer: io.Discard})

	teamStore := &mockTeamStore{
		teams: []tmodels.Team{
			{
				ID:                         1,
				ConversationAssignmentType: AssignmentTypeRoundRobin,
			},
		},
		members: map[int][]tmodels.TeamMember{
			1: {
				{ID: 101, AvailabilityStatus: umodels.Online, TeamID: 1, MaxOpenConversations: 2},
			},
		},
	}

	convoStore := &mockConversationStore{
		unassigned: []models.Conversation{
			{UUID: "c1", AssignedTeamID: null.IntFrom(1)},
		},
		activeCounts: map[int]int{
			101: 2, // 已满员
		},
		claimed: make(map[string]int),
	}

	engine, err := New(teamStore, convoStore, umodels.User{ID: 1}, &lo)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	if err := engine.reloadBalancer(); err != nil {
		t.Fatalf("failed to reload balancer: %v", err)
	}

	if err := engine.assignConversations(); err != nil {
		t.Fatalf("failed to assign conversations: %v", err)
	}

	// 唯一客服已满员，不应被认领
	if _, ok := convoStore.claimed["c1"]; ok {
		t.Errorf("expected conversation c1 not to be claimed when agent is capped")
	}
}
