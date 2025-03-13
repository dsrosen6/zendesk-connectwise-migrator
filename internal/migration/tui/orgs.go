package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/apis/zendesk"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
	"log/slog"
)

type orgCheckerModel struct {
	orgs            []zendesk.Organization
	migrationClient *migration.Client
	status          status
	totalChecked    int
	orgsNotInPsa    []zendesk.Organization
	done            bool
}

type switchStatusMsg string

type status string

const (
	gettingZendeskOrgs status = "gettingZendeskOrgs"
	comparingOrgs      status = "comparingOrgs"
	done               status = "done"
)

func newOrgCheckerModel(mc *migration.Client) *orgCheckerModel {
	return &orgCheckerModel{
		migrationClient: mc,
		status:          gettingZendeskOrgs,
	}
}

func (m *orgCheckerModel) Init() tea.Cmd {
	return m.getOrgs()
}

func (m *orgCheckerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.status == done {
			return m, switchModel(newMainMenuModel(m.migrationClient))
		}

	case switchStatusMsg:
		switch msg {
		case switchStatusMsg(comparingOrgs):
			slog.Debug("got orgs", "total", len(m.orgs))
			m.status = comparingOrgs
			for _, org := range m.orgs {
				cmds = append(cmds, m.compareOrg(org))
			}
		case switchStatusMsg(done):
			m.done = true
		}
	}

	if m.status == comparingOrgs && !m.done {
		if len(m.orgs) == m.totalChecked {
			slog.Debug("done")
			m.status = done
		}
	}

	return m, tea.Sequence(cmds...)
}

func (m *orgCheckerModel) View() string {
	var s string
	switch m.status {
	case gettingZendeskOrgs:
		s = runSpinner("Getting Zendesk orgs")
	case comparingOrgs:
		s = runSpinner(fmt.Sprintf("Comparing orgs...total checked: %d", m.totalChecked))
	case done:
		s = "Done checking orgs\n\n"
		if len(m.orgsNotInPsa) > 0 {
			s += "Orgs not in PSA:\n\n"
			for _, org := range m.orgsNotInPsa {
				s += fmt.Sprintf("%s\n", org.Name)
			}
		} else {
			s += "All orgs are in PSA."
		}

		s += "\n\nPress any key to return to the main menu (ctrl+c or esc to quit)"
	}

	return s
}

func (m *orgCheckerModel) getOrgs() tea.Cmd {
	slog.Debug("starting getOrgs")
	return func() tea.Msg {
		q := &zendesk.SearchQuery{}
		tags := m.migrationClient.Cfg.Zendesk.TagsToMigrate
		if len(tags) > 0 {
			q.Tags = tags
		}

		orgs, err := m.migrationClient.ZendeskClient.GetOrganizationsWithQuery(ctx, *q)
		if err != nil {
			slog.Error("error getting orgs", "err", err)
			return apiErrMsg{err}
		}

		m.orgs = orgs
		slog.Debug("done getting orgs", "total", len(m.orgs))
		return switchStatusMsg(comparingOrgs)
	}
}

func (m *orgCheckerModel) compareOrg(org zendesk.Organization) tea.Cmd {
	slog.Debug("starting compareOrgs")
	return func() tea.Msg {
		m.totalChecked += 1
		_, err := m.migrationClient.MatchZdOrgToCwCompany(ctx, org)
		if err != nil {
			m.orgsNotInPsa = append(m.orgsNotInPsa, org)
			return nil
		}

		return nil
	}
}
