package migration

import (
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
	"log/slog"
	"strconv"
	"strings"
	"sync"
)

const (
	totalConcurrentUsers = 50
)

func (m *Model) getUsersToMigrate(org *orgMigrationDetails) tea.Cmd {
	return func() tea.Msg {
		slog.Debug("getUsersToMigrate: called", "orgName", org.ZendeskOrg.Name)
		users, err := m.client.ZendeskClient.GetOrganizationUsers(m.ctx, org.ZendeskOrg.Id)
		if err != nil {
			slog.Error("getUsersToMigrate: error getting users for org", "orgName", org.ZendeskOrg.Name, "error", err)
			m.writeToOutput(badRedOutput("ERROR", fmt.Sprintf("couldn't get users for %s: %s", org.ZendeskOrg.Name, err)), errOutput)
			m.orgsCheckedForUsers++
			m.userMigrationErrors++
			return nil
		}
		slog.Info("getUsersToMigrate: got users for org", "orgName", org.ZendeskOrg.Name, "totalUsers", len(users))

		m.mu.Lock()
		for _, user := range users {
			idString := strconv.Itoa(user.Id)
			m.data.UsersToMigrate[idString] = &userMigrationDetails{ZendeskUser: &user, PsaCompany: org.PsaOrg}
		}
		m.mu.Unlock()

		m.orgsCheckedForUsers++
		return nil
	}
}

func (m *Model) migrateUsers(users map[string]*userMigrationDetails) tea.Cmd {
	return func() tea.Msg {
		slog.Debug("migrateUsers: called")

		sem := make(chan struct{}, totalConcurrentUsers)
		var wg sync.WaitGroup

		for _, user := range users {
			sem <- struct{}{}
			wg.Add(1)

			go func(user *userMigrationDetails) {
				defer wg.Done()
				defer func() { <-sem }()

				slog.Debug("migrateUsers: migrating user", "userName", user.ZendeskUser.Name)
				m.migrateUser(user)
				if user.PsaContact != nil {
					slog.Debug("migrateUsers: migrated user", "userName", user.ZendeskUser.Name, "psaContactId", user.PsaContact.Id)
				} else {
					slog.Debug("migrateUsers: user not migrated", "userName", user.ZendeskUser.Name)
				}
			}(user)
		}

		wg.Wait()
		slog.Debug("migrateUsers: done")

		if m.client.Cfg.StopAfterUsers {
			slog.Info("migrateUsers: stopping after user migration as per configuration")
			return switchStatusMsg(done)
		}

		slog.Info("migrateUsers: all users migrated, beginning ticket migration")
		return switchStatusMsg(gettingPsaTickets)
	}
}

func (m *Model) migrateUser(user *userMigrationDetails) {
	if user.ZendeskUser.Email == "" {
		slog.Warn("migrateUser: zendesk user has no email address - skipping", "userName", user.ZendeskUser.Name)
		m.writeToOutput(warnYellowOutput("WARN", fmt.Sprintf("user has no email address, skipping migration: %s", user.ZendeskUser.Name)), warnOutput)
		m.usersProcessed++
		return
	}

	var err error
	user.PsaContact, err = m.matchZdUserToCwContact(user.ZendeskUser)
	if err != nil {

		if errors.Is(err, psa.NoUserFoundErr{}) {
			slog.Debug("migrateUser: user does not exist in psa - attempting to create new user", "userEmail", user.ZendeskUser.Email)
			user.PsaContact, err = m.createPsaContact(user)
			if err != nil {
				slog.Error("migrateUser: error creating user", "userEmail", user.ZendeskUser.Email, "zendeskUserId", user.ZendeskUser.Id, "error", err)
				m.writeToOutput(badRedOutput("ERROR", fmt.Sprintf("couldn't create user %s: %s", user.ZendeskUser.Email, err)), errOutput)
				m.usersProcessed++
				m.userMigrationErrors++
				return
			} else {
				slog.Debug("migrateUser: created new psa user", "userName", user.ZendeskUser.Email, "psaContactId", user.PsaContact.Id)
			}

		} else {
			slog.Error("migrateUser: error matching zendesk user to psa user", "userEmail", user.ZendeskUser.Email, "zendeskUserId", user.ZendeskUser.Id, "error", err)
			m.writeToOutput(badRedOutput("ERROR", fmt.Sprintf("couldn't match zendesk user %s to psa user: %s", user.ZendeskUser.Email, err)), errOutput)
			m.usersProcessed++
			m.userMigrationErrors++
			return
		}
	}

	slog.Debug("migrateUser: matched zendesk user to psa user", "userEmail", user.ZendeskUser.Email, "psaContactId", user.PsaContact.Id)
	if user.ZendeskUser.UserFields.PSAContactId != user.PsaContact.Id {
		if err := m.updateContactFieldValue(user); err != nil {
			slog.Error("migrateUser: error updating user contact field value in zendesk", "userEmail", user.ZendeskUser.Email, "zendeskUserId", user.ZendeskUser.Id, "psaContactId", user.PsaContact.Id, "error", err)
			m.writeToOutput(badRedOutput("ERROR", fmt.Sprintf("couldn't update contact field in zendesk for user %s: %s", user.ZendeskUser.Email, err)), errOutput)
			m.usersProcessed++
			m.userMigrationErrors++
			return
		}
	} else {
		slog.Debug("migrateUser: user already has psa contact id field - skipping", "userEmail", user.ZendeskUser.Email, "zendeskUserId", user.ZendeskUser.Id, "psaContactId", user.PsaContact.Id)
		m.mu.Lock()
		m.data.UsersInPsa[strconv.Itoa(user.ZendeskUser.Id)] = user
		m.mu.Unlock()

		m.usersProcessed++
		return
	}

	slog.Info("migrateUser: new user migrated", "userEmail", user.ZendeskUser.Email, "psaContactId", user.PsaContact.Id)
	m.mu.Lock()
	m.data.UsersInPsa[strconv.Itoa(user.ZendeskUser.Id)] = user
	m.mu.Unlock()

	m.usersProcessed++
	m.newUsersCreated++
	return
}

func (m *Model) matchZdUserToCwContact(user *zendesk.User) (*psa.Contact, error) {
	if user == nil {
		return nil, errors.New("user is nil")
	}

	if user.Email == "" {
		return nil, errors.New("user email is empty")
	}

	contact, err := m.client.CwClient.GetContactByEmail(m.ctx, user.Email)
	if err != nil {
		return nil, err
	}
	return contact, nil
}

func (m *Model) createPsaContact(user *userMigrationDetails) (*psa.Contact, error) {
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
			CommunicationType: "Email",
		},
	}

	return m.client.CwClient.PostContact(m.ctx, c)
}

type ZendeskFieldAlreadySetErr struct{}

func (e ZendeskFieldAlreadySetErr) Error() string {
	return "zendesk user already has psa contact id field"
}

func (m *Model) updateContactFieldValue(user *userMigrationDetails) error {
	if user.ZendeskUser.UserFields.PSAContactId == user.PsaContact.Id {
		return ZendeskFieldAlreadySetErr{}
	}

	if user.PsaContact.Id != 0 {
		user.ZendeskUser.UserFields.PSAContactId = user.PsaContact.Id

		var err error
		user.ZendeskUser, err = m.client.ZendeskClient.UpdateUser(m.ctx, user.ZendeskUser)
		if err != nil {
			return fmt.Errorf("updating user with PSA contact id: %w", err)
		}

		slog.Debug("updated zendesk user with PSA contact id", "userEmail", user.ZendeskUser.Email)
		return nil
	} else {
		slog.Error("user psa id is 0 - cannot update psa_contact field in zendesk", "userEmail", user.ZendeskUser.Email, "zendeskUserId", user.ZendeskUser.Id)
		return errors.New("user psa id is 0 - cannot update psa_contact field in zendesk")
	}
}

func separateName(name string) (string, string) {
	nameParts := strings.Split(name, " ")
	if len(nameParts) == 1 {
		return nameParts[0], ""
	}

	return nameParts[0], strings.Join(nameParts[1:], " ")
}
