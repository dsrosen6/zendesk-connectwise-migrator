package tui

import (
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
	"log/slog"
	"sort"
	"strings"
)

type userMigrationModel struct {
	client          *migration.Client
	data            *MigrationData
	form            *huh.Form
	formHeight      int
	selectedOrgs    []*orgMigrationDetails
	allOrgsSelected bool
	userMigTotals

	status userMigStatus
	done   bool
}

type userMigTotals struct {
	totalOrgsToMigrateUsers int
	totalOrgsChecked        int
	totalUsersToProcess     int
	totalNewUsersCreated    int
	totalUsersProcessed     int
	totalUsersSkipped       int
	totalOrgsDone           int
	totalErrors             int
}

type userMigStatus string
type switchUserMigStatusMsg string

func switchUserMigStatus(s userMigStatus) tea.Cmd {
	return func() tea.Msg {
		return switchUserMigStatusMsg(s)
	}
}

type initFormMsg struct{}

func initForm() tea.Cmd {
	return func() tea.Msg {
		return initFormMsg{}
	}
}

const (
	noOrgs         userMigStatus = "noOrgs"
	waitingForOrgs userMigStatus = "waitingForOrgs"
	pickingOrgs    userMigStatus = "pickingOrgs"
	gettingUsers   userMigStatus = "gettingUsers"
	migratingUsers userMigStatus = "migratingUsers"
	userMigDone    userMigStatus = "userMigDone"
)

func newUserMigrationModel(mc *migration.Client, data *MigrationData) *userMigrationModel {
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

	case initFormMsg:
		if len(m.data.Orgs) == 0 {
			slog.Warn("got initFormMsg, but no orgs")
			m.data.writeToOutput(warnYellowOutput("WARNING", "no organizations")) // TODO: something more specific
			return m, switchUserMigStatus(noOrgs)
		} else {
			slog.Debug("got initFormMsg", "totalOrgs", len(m.data.Orgs))
			m.form = m.orgSelectionForm()
			cmds = append(cmds, m.form.Init(), switchUserMigStatus(pickingOrgs))
			return m, tea.Sequence(cmds...)
		}

	case switchUserMigStatusMsg:
		slog.Debug("user migration: got switchUserMigStatusMsg", "status", msg)
		switch msg {
		case switchUserMigStatusMsg(noOrgs):
			m.status = noOrgs

		case switchUserMigStatusMsg(waitingForOrgs):
			m.status = waitingForOrgs
		case switchUserMigStatusMsg(pickingOrgs):
			slog.Debug("got pickingOrgs status")
			m.form = m.orgSelectionForm()
			m.status = pickingOrgs
			cmds = append(cmds, m.form.Init())
			return m, tea.Sequence(cmds...)

		case switchUserMigStatusMsg(gettingUsers):
			m.userMigTotals = userMigTotals{}
			m.status = gettingUsers
			for _, org := range m.data.Orgs {
				if org.OrgMigrated && org.UserMigSelected {
					m.totalOrgsToMigrateUsers++
					cmds = append(cmds, m.getUsersToMigrate(org))
				}
			}

			slog.Debug("user migration: orgs picked", "totalOrgs", m.totalOrgsToMigrateUsers)
			return m, tea.Sequence(cmds...) // TODO: Switch to Batch when ready for speed

		case switchUserMigStatusMsg(migratingUsers):
			m.status = migratingUsers
			for _, org := range m.data.Orgs {
				if org.UserMigSelected {
					m.totalUsersToProcess += len(org.UsersToMigrate)
					cmds = append(cmds, m.migrateOrgUsers(org))
				}
			}

			return m, tea.Sequence(cmds...)

		case switchUserMigStatusMsg(userMigDone):
			m.status = userMigDone
			cmds = append(cmds, saveDataCmd())
		}
	}

	if m.status == pickingOrgs {
		form, cmd := m.form.Update(msg)
		cmds = append(cmds, cmd)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
		}

		// Once the form is submitted, mark all selected orgs as needing migration
		if m.form.State == huh.StateCompleted {

			if m.allOrgsSelected {
				for _, org := range m.data.Orgs {
					if !org.OrgMigrated {
						continue
					}
					org.UserMigSelected = true
				}
			} else {
				for _, org := range m.selectedOrgs {
					org.UserMigSelected = true
				}
			}

			cmds = append(cmds, switchUserMigStatus(gettingUsers))
		}
	}

	if m.status == gettingUsers && m.totalOrgsToMigrateUsers == m.totalOrgsChecked {
		slog.Debug("all orgs have been checked")
		cmds = append(cmds, switchUserMigStatus(migratingUsers))
	}

	if m.status == migratingUsers && m.totalUsersToProcess == m.totalUsersProcessed {
		cmds = append(cmds, switchUserMigStatus(userMigDone))
	}

	return m, tea.Batch(cmds...)
}

func (m *userMigrationModel) View() string {
	var s string
	switch m.status {
	case noOrgs:
		s = "No orgs have been loaded! Please return to the main menu and select Organizations, then return."
	case waitingForOrgs:
		s = runSpinner("Org migration is running - please wait")
	case pickingOrgs:
		s = m.form.View()
	case gettingUsers:
		s = runSpinner("Getting users")
	case migratingUsers:
		counterStatus := fmt.Sprintf("Processing users: %d/%d", m.totalUsersProcessed, m.totalUsersToProcess)
		s = runSpinner(counterStatus)
	case userMigDone:
		s = fmt.Sprintf("User migration done - press %s to run again\n\n%s", textNormalAdaptive("SPACE"), m.constructSummary())
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
				Value(&m.allOrgsSelected),
		),
		huh.NewGroup(
			huh.NewMultiSelect[*orgMigrationDetails]().
				Title("Pick the orgs you'd like to migrate users for").
				Description("Use Space to select, and Enter/Return to submit").
				Options(m.orgOptions()...).
				Value(&m.selectedOrgs),
		).WithHideFunc(func() bool { return m.allOrgsSelected == true }),
	).WithHeight(verticalLeftForMainView).WithShowHelp(false).WithTheme(migration.CustomHuhTheme())
}

func (m *userMigrationModel) orgOptions() []huh.Option[*orgMigrationDetails] {
	var orgOptions []huh.Option[*orgMigrationDetails]
	for _, org := range m.data.Orgs {
		if org.OrgMigrated {
			opt := huh.Option[*orgMigrationDetails]{
				Key:   org.ZendeskOrg.Name,
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
		users, err := m.client.ZendeskClient.GetOrganizationUsers(ctx, org.ZendeskOrg.Id)
		if err != nil {
			slog.Error("getting users for org", "orgName", org.ZendeskOrg.Name, "error", err)
			m.data.writeToOutput(badRedOutput("ERROR", fmt.Errorf("couldn't get users for org %s: %v", org.ZendeskOrg.Name, err)))
			m.totalOrgsChecked++
			m.totalErrors++
			return nil
		}

		for _, user := range users {
			slog.Debug("got user", "orgName", org.ZendeskOrg.Name, "userName", user.Name)
			idString := fmt.Sprintf("%d", user.Id)
			if _, ok := org.UsersToMigrate[idString]; !ok {
				slog.Debug("adding user to org", "orgName", org.ZendeskOrg.Name, "userName", user.Name)
				org.addUserToUsersMap(idString, &userMigrationDetails{ZendeskUser: &user, PsaCompany: org.PsaOrg})
			} else {
				slog.Debug("user already in org", "orgName", org.ZendeskOrg.Name, "userName", user.Name)
			}
		}

		slog.Info("got users for org", "orgName", org.ZendeskOrg.Name, "totalUsers", len(org.UsersToMigrate))
		m.totalOrgsChecked++
		return nil
	}
}

func (md *orgMigrationDetails) addUserToUsersMap(idString string, user *userMigrationDetails) {
	md.UsersToMigrate[idString] = user
}

func (m *userMigrationModel) migrateOrgUsers(org *orgMigrationDetails) tea.Cmd {
	return func() tea.Msg {
		for _, user := range org.UsersToMigrate {
			if user.Migrated {
				slog.Debug("user already migrated", "userEmail", user.ZendeskUser.Email)
				m.data.writeToOutput(goodBlueOutput("NO ACTION", fmt.Sprintf("User already migrated: %s", user.ZendeskUser.Email)))
				m.totalUsersProcessed++
				continue
			}

			if user.ZendeskUser.Email == "" {
				slog.Warn("zendesk user has no email address - skipping", "userName", user.ZendeskUser.Name)
				m.data.writeToOutput(warnYellowOutput("WARNING", fmt.Sprintf("User has no email address - skipping: %s", user.ZendeskUser.Name)))
				m.totalUsersProcessed++
				m.totalUsersSkipped++
				continue
			}

			var err error
			user.PsaContact, err = m.matchZdUserToCwContact(user.ZendeskUser)
			if err != nil {

				if errors.Is(err, psa.NoUserFoundErr{}) {
					slog.Debug("user does not exist in psa - attempting to create new user", "userEmail", user.ZendeskUser.Email)
					user.PsaContact, err = m.createPsaContact(user)

					if err != nil {
						slog.Error("creating user", "userName", user.ZendeskUser.Email, "error", err)
						m.data.writeToOutput(badRedOutput("ERROR", fmt.Errorf("creating user %s: %v", user.ZendeskUser.Email, err)))
						m.totalUsersProcessed++
						m.totalErrors++
						continue
					}

					slog.Debug("matched zendesk user to psa user", "userEmail", user.ZendeskUser.Email)
					m.totalNewUsersCreated++

				} else {
					slog.Error("matching zendesk user to psa user", "userEmail", user.ZendeskUser.Email, "error", err)
					m.data.writeToOutput(badRedOutput("ERROR", fmt.Errorf("matching zendesk user to psa user %s: %v", user.ZendeskUser.Email, err)))
					m.totalUsersProcessed++
					m.totalErrors++
					continue
				}
			}

			if err := m.updateContactFieldValue(user); err != nil {
				if errors.Is(err, ZendeskFieldAlreadySetErr{}) {
					user.Migrated = true
					slog.Debug("zendesk user already has psa contact id field", "userEmail", user.ZendeskUser.Email, "psaContactId", user.PsaContact.Id)
					m.data.writeToOutput(goodBlueOutput("NO ACTION", fmt.Sprintf("User already migrated: %s", user.ZendeskUser.Email)))
					m.totalUsersProcessed++
					continue
				}

				slog.Error("updating user contact field value in zendesk", "userEmail", user.ZendeskUser.Email, "error", err)
				m.data.writeToOutput(badRedOutput("ERROR", fmt.Errorf("updating contact field in zendesk for %s: %v", user.ZendeskUser.Email, err)))
				m.totalUsersProcessed++
				m.totalErrors++
				continue
			}

			if user.PsaContact != nil && user.ZendeskUser.UserFields.PSAContactId == user.PsaContact.Id {
				user.Migrated = true
				slog.Info("user is fully migrated", "userEmail", user.ZendeskUser.Email, "psaContactId", user.PsaContact.Id)
				m.data.writeToOutput(goodGreenOutput("SUCCESS", fmt.Sprintf("User fully migrated: %s", user.ZendeskUser.Email)))
				m.totalUsersProcessed++
			}
		}
		org.UserMigSelected = false
		m.totalOrgsDone++
		return nil
	}
}

func (m *userMigrationModel) matchZdUserToCwContact(user *zendesk.User) (*psa.Contact, error) {
	contact, err := m.client.CwClient.GetContactByEmail(ctx, user.Email)
	if err != nil {
		return nil, err
	}
	return contact, nil
}

func (m *userMigrationModel) createPsaContact(user *userMigrationDetails) (*psa.Contact, error) {
	c := &psa.ContactPostBody{}
	c.FirstName, c.LastName = separateName(user.ZendeskUser.Name)

	if user.PsaCompany == nil {
		return nil, errors.New("user psa company is nil")
	}

	c.Company.Id = user.PsaCompany.Id

	c.CommunicationItems = []psa.CommunicationItem{
		{
			Type:              psa.CommunicationItemType{Name: "Email"},
			Value:             user.ZendeskUser.Email,
			DefaultFlag:       true,
			CommunicationType: "Email",
		},
	}

	return m.client.CwClient.PostContact(ctx, c)
}

type ZendeskFieldAlreadySetErr struct{}

func (e ZendeskFieldAlreadySetErr) Error() string {
	return "zendesk user already has psa contact id field"
}

func (m *userMigrationModel) updateContactFieldValue(user *userMigrationDetails) error {
	if user.ZendeskUser.UserFields.PSAContactId == user.PsaContact.Id {
		slog.Debug("zendesk user already has PSA contact id field",
			"userEmail", user.ZendeskUser.Email,
			"psaContactId", user.PsaContact.Id,
		)
		return ZendeskFieldAlreadySetErr{}
	}

	if user.PsaContact.Id != 0 {
		user.ZendeskUser.UserFields.PSAContactId = user.PsaContact.Id

		var err error
		user.ZendeskUser, err = m.client.ZendeskClient.UpdateUser(ctx, user.ZendeskUser)
		if err != nil {
			return fmt.Errorf("updating user with PSA contact id: %w", err)
		}

		slog.Info("updated zendesk user with PSA contact id", "userEmail", user.ZendeskUser.Email)
		return nil
	} else {
		slog.Error("user psa id is 0 - cannot update psa_contact field in zendesk", "userName", user.ZendeskUser.Name)
		return errors.New("user psa id is 0 - cannot update psa_contact field in zendesk")
	}
}

func (m *userMigrationModel) constructSummary() string {
	return fmt.Sprintf("%s %d/%d\n"+
		"%s %d/%d\n"+
		"%s %d\n"+
		"%s %d\n"+
		"%s %d\n",
		textNormalAdaptive("Orgs Processed:"), m.totalOrgsChecked, m.totalOrgsToMigrateUsers,
		textNormalAdaptive("Users Processed:"), m.totalUsersProcessed, m.totalUsersToProcess,
		textNormalAdaptive("New Users Created:"), m.totalNewUsersCreated,
		textNormalAdaptive("Users Skipped:"), m.totalUsersSkipped,
		textNormalAdaptive("Errors:"), m.totalErrors)
}

func separateName(name string) (string, string) {
	nameParts := strings.Split(name, " ")
	if len(nameParts) == 1 {
		return nameParts[0], ""
	}

	return nameParts[0], strings.Join(nameParts[1:], " ")
}
