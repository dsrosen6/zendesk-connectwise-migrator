package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/apis/zendesk"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
	"log/slog"
	"time"
)

type orgCheckerModel struct {
	orgs            allOrgs
	migrationClient *migration.Client
	status          status
	orgsNotInPsa    []zendesk.Organization
	done            bool
}

type allOrgs struct {
	master      []zendesk.Organization
	checked     []zendesk.Organization
	withTickets []zendesk.Organization
	inPsa       []zendesk.Organization
	notInPsa    []zendesk.Organization
	erroredOrgs []erroredOrg
}

type erroredOrg struct {
	org zendesk.Organization
	err error
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
			slog.Debug("got orgs", "total", len(m.orgs.master))
			m.status = comparingOrgs
			for _, org := range m.orgs.master {
				cmds = append(cmds, m.checkOrg(org))
			}
		case switchStatusMsg(done):
			m.done = true
		}
	}

	if m.status == comparingOrgs && !m.done {
		if len(m.orgs.master) == len(m.orgs.checked) {
			slog.Debug("done")
			m.status = done
		}
	}

	return m, tea.Sequence(cmds...)
}

func (m *orgCheckerModel) View() string {
	var st string
	switch m.status {
	case gettingZendeskOrgs:
		st = runSpinner("Getting Zendesk orgs")
	case comparingOrgs:
		st = runSpinner("Checking orgs")
	case done:
		st = " Done checking orgs"
	}

	stats := fmt.Sprintf(" Checked: %d/%d\n With Tickets: %d\n In PSA: %d/%d\n",
		len(m.orgs.checked), len(m.orgs.master), len(m.orgs.withTickets), len(m.orgs.inPsa), len(m.orgs.withTickets))

	return fmt.Sprintf("%s\n\n%s", st, stats)
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

		m.orgs.master = orgs
		return switchStatusMsg(comparingOrgs)
	}
}

func (m *orgCheckerModel) checkOrg(org zendesk.Organization) tea.Cmd {
	return func() tea.Msg {
		q := &zendesk.SearchQuery{}
		start, end, err := convertDateStringsToTimeTime(m.migrationClient.Cfg.Zendesk.StartDate, m.migrationClient.Cfg.Zendesk.EndDate)
		if err != nil {
			return timeConvertErrMsg{err}
		}
		if start != (time.Time{}) {
			q.TicketCreatedAfter = start
		}

		if end != (time.Time{}) {
			q.TicketCreatedBefore = end
		}

		q.TicketsOrganizationId = org.Id

		tickets, err := m.migrationClient.ZendeskClient.GetTicketsWithQuery(ctx, *q)
		if err != nil {
			m.orgs.erroredOrgs = append(m.orgs.erroredOrgs, erroredOrg{org: org, err: err})
		}

		if len(tickets) > 0 {
			m.orgs.withTickets = append(m.orgs.withTickets, org)
			if m.orgInPsa(org) {
				m.orgs.inPsa = append(m.orgs.inPsa, org)
			} else {
				m.orgs.notInPsa = append(m.orgs.inPsa)
			}
		}

		m.orgs.checked = append(m.orgs.checked, org)
		return nil
	}
}

func (m *orgCheckerModel) orgInPsa(org zendesk.Organization) bool {
	_, err := m.migrationClient.MatchZdOrgToCwCompany(ctx, org)
	return err == nil
}
