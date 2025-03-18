package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
	"log/slog"
	"sort"
	"strings"
)

type userMigrationModel struct {
	client         *migration.Client
	data           *migrationData
	form           *huh.Form
	formHeight     int
	selectedOrgs   []*orgMigrationDetails
	allOrgsChecked bool
	status         userMigStatus
	done           bool
}

type userMigStatus string
type switchUserMigStatusMsg string

func switchUserMigStatus(s userMigStatus) tea.Cmd {
	return func() tea.Msg {
		return switchUserMigStatusMsg(s)
	}
}

const (
	noOrgs         userMigStatus = "noOrgs"
	pickingOrgs    userMigStatus = "pickingOrgs"
	gettingUsers   userMigStatus = "gettingUsers"
	migratingUsers userMigStatus = "migratingUsers"
	userMigDone    userMigStatus = "userMigDone"
)

func newUserMigrationModel(mc *migration.Client, data *migrationData) *userMigrationModel {
	m := &userMigrationModel{
		client: mc,
		data:   data,
		status: noOrgs,
	}

	return m
}

func (m *userMigrationModel) Init() tea.Cmd {
	return nil
}

func (m *userMigrationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case calculateDimensionsMsg:
		if m.form != nil {
			m.form.WithHeight(verticalLeftForMainView)
		}

	case switchUserMigStatusMsg:
		slog.Debug("user migration: got switchUserMigStatusMsg", "status", msg)
		switch msg {
		case switchUserMigStatusMsg(pickingOrgs):
			slog.Debug("got pickingOrgs status")
			m.form = m.orgSelectionForm()
			m.status = pickingOrgs
			cmds = append(cmds, m.form.Init())
			return m, tea.Batch(cmds...)

		case switchUserMigStatusMsg(gettingUsers):
			m.status = gettingUsers
			var orgsToMigrateUsers []*orgMigrationDetails
			for _, org := range m.data.orgs {
				if org.readyUsers && org.userMigSelected {
					orgsToMigrateUsers = append(orgsToMigrateUsers, org)
				}
			}

			slog.Debug("user migration: orgs picked", "totalOrgs", len(m.data.orgs))
			for _, org := range orgsToMigrateUsers {
				cmds = append(cmds, m.getUsersToMigrate(org))
			}
			return m, tea.Batch(cmds...)

		}
	}

	if len(m.data.orgs) == 0 {
		m.status = noOrgs
	}

	if m.status == pickingOrgs {
		form, cmd := m.form.Update(msg)
		cmds = append(cmds, cmd)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
		}

		if m.form.State == huh.StateCompleted {
			for _, org := range m.selectedOrgs {
				org.userMigSelected = true
			}

			cmds = append(cmds, switchUserMigStatus(gettingUsers))
		}

	}

	return m, tea.Batch(cmds...)
}

func (m *userMigrationModel) View() string {
	var s string
	switch m.status {
	case noOrgs:
		s = "No orgs have been loaded! Please return to the main menu and select Organizations, then return."
	case pickingOrgs:
		s = m.form.View()
	case gettingUsers:
		s = "Getting users"
	}

	return s
}

func (m *userMigrationModel) orgSelectionForm() *huh.Form {
	return huh.NewForm(

		huh.NewGroup(
			huh.NewSelect[bool]().
				Title("Migrate all confirmed orgs?").
				Description("If not, select the organizations you want to migrate on the next screen.").
				Options(
					huh.NewOption("All Orgs", true),
					huh.NewOption("Select Orgs", false)).
				Value(&m.allOrgsChecked),
		),
		huh.NewGroup(
			huh.NewMultiSelect[*orgMigrationDetails]().
				Title("Pick the orgs you'd like to migrate users for").
				Description("Use Space to select, and Enter/Return to submit").
				Options(m.orgOptions()...).
				Value(&m.selectedOrgs),
		).WithHideFunc(func() bool { return m.allOrgsChecked == true }),
	).WithHeight(verticalLeftForMainView).WithShowHelp(false).WithTheme(migration.CustomHuhTheme())
}

func (m *userMigrationModel) orgOptions() []huh.Option[*orgMigrationDetails] {
	var orgOptions []huh.Option[*orgMigrationDetails]
	for _, org := range m.data.orgs {
		if org.readyUsers {
			opt := huh.Option[*orgMigrationDetails]{
				Key:   org.zendeskOrg.Name,
				Value: org,
			}

			orgOptions = append(orgOptions, opt)
		}
	}

	sort.Slice(orgOptions, func(i, j int) bool {
		return orgOptions[i].Key < orgOptions[j].Key
	})

	return orgOptions
}

func (m *userMigrationModel) getUsersToMigrate(org *orgMigrationDetails) tea.Cmd {
	return func() tea.Msg {
		users, err := m.client.ZendeskClient.GetOrganizationUsers(ctx, org.zendeskOrg.Id)
		if err != nil {
			slog.Error("getting users for org", "orgName", org.zendeskOrg.Name, "error", err)
			org.userMigErrors = append(org.userMigErrors, fmt.Errorf("getting users: %w", err))
			return nil
		}

		for _, user := range users {
			slog.Debug("got user", "orgName", org.zendeskOrg.Name, "userName", user.Name)
			org.usersToMigrate = append(org.usersToMigrate, &userMigrationDetails{zendeskUser: &user})
		}

		slog.Info("got users for org", "orgName", org.zendeskOrg.Name, "totalUsers", len(org.usersToMigrate))
		return nil
	}
}

func separateName(name string) (string, string) {
	nameParts := strings.Split(name, " ")
	if len(nameParts) == 1 {
		return nameParts[0], ""
	}

	return nameParts[0], strings.Join(nameParts[1:], " ")
}
