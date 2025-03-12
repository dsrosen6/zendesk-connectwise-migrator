package migration

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
	"log/slog"
)

type checkOrgsModel struct {
	Client      *Client
	zendeskOrgs []zendesk.Organization
}

type gotOrgs string

func newCheckOrgsModel(c *Client) checkOrgsModel {
	return checkOrgsModel{
		Client:      c,
		zendeskOrgs: []zendesk.Organization{},
	}
}

func (co checkOrgsModel) Init() tea.Cmd {
	return co.getOrgsByTag(co.Client.Cfg.Zendesk.TagsToMigrate)
}

func (co checkOrgsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return co, tea.Quit
		}
	}
	return co, nil
}

func (co checkOrgsModel) View() string {
	var s string
	s = fmt.Sprintf("Total Zendesk Orgs Received: %d\n", len(co.zendeskOrgs))

	return s
}

func (co checkOrgsModel) getOrgsByTag(tags []string) tea.Cmd {
	return func() tea.Msg {
		slog.Debug("starting GetOrgsByTag")
		orgs, err := co.Client.ZendeskClient.GetOrganizationsWithQuery(ctx, tags)
		if err != nil {
			return err
		}

		co.zendeskOrgs = orgs
		slog.Debug("got initial zendesk orgs", "len", len(co.zendeskOrgs))
		return gotOrgs("gotOrgs")
	}
}
