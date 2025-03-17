package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
	"log/slog"
	"sort"
)

type userMigrationModel struct {
	migrationClient *migration.Client
	form            *huh.Form
	formHeight      int
	migrationData   *migrationData
	allOrgsChecked  bool
	status          userMigStatus
	done            bool
}

type userMigStatus string
type switchUserMigStatusMsg string

func switchUserMigStatus(s userMigStatus) tea.Cmd {
	return func() tea.Msg {
		return switchUserMigStatus(s)
	}
}

const (
	noOrgs       userMigStatus = "noOrgs"
	pickingOrgs  userMigStatus = "pickingOrgs"
	gettingUsers userMigStatus = "gettingUsers"
)

func newUserMigrationModel(mc *migration.Client, data *migrationData) *userMigrationModel {
	m := &userMigrationModel{
		migrationClient: mc,
		migrationData:   data,
		status:          pickingOrgs,
	}

	m.form = m.orgSelectionForm()

	return m
}

func (m *userMigrationModel) Init() tea.Cmd {
	return tea.Batch(m.form.Init())
}

func (m *userMigrationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case calculateDimensionsMsg:
		m.form.WithHeight(verticalLeftForMainView)
		
	case switchUserMigStatusMsg:
		slog.Debug("user migration: got switchUserMigStatusMsg", "status", msg)
		switch msg {
		case switchUserMigStatusMsg(gettingUsers):
			if m.allOrgsChecked {
				m.migrationData.orgsToMigrateUsers = m.migrationData.readyOrgs
			}
			slog.Debug("user migration: orgs picked", "totalOrgs", len(m.migrationData.readyOrgs))
		}
	}

	if len(m.migrationData.readyOrgs) == 0 {
		m.status = noOrgs
	}

	form, cmd := m.form.Update(msg)
	cmds = append(cmds, cmd)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	if m.form.State == huh.StateCompleted {
		cmds = append(cmds, switchUserMigStatus(gettingUsers))
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
				Value(&m.migrationData.orgsToMigrateUsers),
		).WithHideFunc(func() bool { return m.allOrgsChecked == true }),
	).WithHeight(verticalLeftForMainView).WithShowHelp(false).WithTheme(migration.CustomHuhTheme())
}

func (m *userMigrationModel) orgOptions() []huh.Option[*orgMigrationDetails] {
	var orgOptions []huh.Option[*orgMigrationDetails]
	for _, org := range m.migrationData.readyOrgs {
		opt := huh.Option[*orgMigrationDetails]{
			Key:   org.zendeskOrg.Name,
			Value: org,
		}

		orgOptions = append(orgOptions, opt)
	}

	sort.Slice(orgOptions, func(i, j int) bool {
		return orgOptions[i].Key < orgOptions[j].Key
	})

	return orgOptions
}
